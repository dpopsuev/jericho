package arsenal

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Arsenal combines catalog snapshots with consumer preferences for model selection.
type Arsenal struct {
	snapshots   map[string]*Snapshot
	active      string // resolved pin
	discoverers []ModelDiscoverer
	log         *slog.Logger
}

// WithLogger sets the structured logger for discovery instrumentation.
func (a *Arsenal) WithLogger(l *slog.Logger) *Arsenal {
	a.log = l
	return a
}

// discoverTimeout is the max wall-clock time for all discoverers combined.
const discoverTimeout = 5 * time.Second

// NewArsenal loads the embedded catalog and resolves the pin.
// Pin "" or "latest" uses the newest snapshot. Explicit pin (e.g. "2026-03")
// selects that snapshot.
func NewArsenal(pin string) (*Arsenal, error) {
	names, err := availableSnapshots()
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return nil, ErrEmptyCatalog
	}

	a := &Arsenal{snapshots: make(map[string]*Snapshot)}

	for _, name := range names {
		snap, err := loadSnapshot(name)
		if err != nil {
			return nil, err
		}
		a.snapshots[name] = snap
	}

	// Resolve pin.
	if pin == "" || pin == "latest" {
		a.active = names[len(names)-1] // alphabetically last = latest
	} else {
		if _, ok := a.snapshots[pin]; !ok {
			return nil, fmt.Errorf("%w: %q (available: %v)", ErrBadPin, pin, names)
		}
		a.active = pin
	}

	return a, nil
}

// RegisterDiscoverer adds a live model discoverer. Call before Discover().
func (a *Arsenal) RegisterDiscoverer(d ModelDiscoverer) {
	if d != nil {
		a.discoverers = append(a.discoverers, d)
	}
}

// Discover fans out to all registered discoverers and merges results
// into the active snapshot. Timeout: 5 seconds for all providers.
// Non-fatal: discovery failures are collected but don't prevent startup.
func (a *Arsenal) Discover(ctx context.Context) []error {
	if len(a.discoverers) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, discoverTimeout)
	defer cancel()

	type result struct {
		models []DiscoveredModel
		err    error
	}

	results := make([]result, len(a.discoverers))
	var wg sync.WaitGroup

	for i, d := range a.discoverers {
		wg.Add(1)
		go func(idx int, disc ModelDiscoverer) {
			defer wg.Done()
			models, err := disc.Discover(ctx)
			results[idx] = result{models, err}
		}(i, d)
	}
	wg.Wait()

	snap := a.snapshots[a.active]
	var errs []error
	start := time.Now()
	for i, r := range results {
		provider := a.discoverers[i].Provider()
		if r.err != nil {
			// ORANGE: log discovery failure.
			errs = append(errs, fmt.Errorf("%s: %w", provider, r.err))
			if a.log != nil {
				a.log.WarnContext(ctx, "model discovery failed",
					slog.String("provider", provider),
					slog.String("error", r.err.Error()),
				)
			}
			continue
		}
		MergeDiscovery(snap, r.models)

		// YELLOW: log discovery success per provider.
		if a.log != nil {
			available := 0
			for _, m := range r.models {
				if m.Available {
					available++
				}
			}
			a.log.InfoContext(ctx, "model discovery completed",
				slog.String("provider", provider),
				slog.Int("total", len(r.models)),
				slog.Int("available", available),
			)
		}
	}
	elapsed := time.Since(start)

	// YELLOW: log overall discovery summary.
	if a.log != nil && len(a.discoverers) > 0 {
		a.log.InfoContext(ctx, "arsenal discovery finished",
			slog.Int("providers", len(a.discoverers)),
			slog.Int("errors", len(errs)),
			slog.Duration("elapsed", elapsed),
		)
	}

	return errs
}

// Pin returns the active snapshot name.
func (a *Arsenal) Pin() string { return a.active }

// Available returns all snapshot names.
func (a *Arsenal) Available() []string {
	names := make([]string, 0, len(a.snapshots))
	for name := range a.snapshots {
		names = append(names, name)
	}
	return names
}

// Pick performs an imperative 1:1 map lookup. Returns the model resolved
// through the given source, with source modifiers applied.
func (a *Arsenal) Pick(modelID, sourceID string) (ResolvedAgent, error) {
	snap := a.snapshots[a.active]

	model, ok := snap.Models[modelID]
	if !ok {
		return ResolvedAgent{}, fmt.Errorf("%w: model %q", ErrNotFound, modelID)
	}

	source, ok := snap.Sources[sourceID]
	if !ok {
		return ResolvedAgent{}, fmt.Errorf("%w: source %q", ErrNotFound, sourceID)
	}

	if !canAccess(source, modelID) {
		return ResolvedAgent{}, fmt.Errorf("%w: source %q cannot access model %q", ErrNotFound, sourceID, modelID)
	}

	return resolve(model, source), nil
}

// Select performs declarative intent-based selection. Filters, gates,
// scores, ranks, then exits through Pick(). The intent parameter is
// reserved for future trait inference — currently unused.
func (a *Arsenal) Select(_ string, prefs *Preferences) (ResolvedAgent, error) {
	snap := a.snapshots[a.active]

	type candidate struct {
		model  *ModelEntry
		source *SourceEntry
		score  float64
	}

	var candidates []candidate

	for _, source := range snap.Sources {
		// Source filter.
		if !prefs.Sources.matches(source.Source) {
			continue
		}

		// Iterate models this source can access.
		for modelID, model := range snap.Models {
			if !canAccess(source, modelID) {
				continue
			}

			// Source provider mask — when a source declares a provider,
			// only models from that provider are reachable through it.
			if source.Provider != "" && model.Provider != source.Provider {
				continue
			}

			// Provider filter.
			if !prefs.Providers.matches(model.Provider) {
				continue
			}

			// Model filter.
			if !prefs.Models.matches(model.ID) {
				continue
			}

			// Availability gate — skip models marked unavailable by discovery.
			// Available==false means discovery ran and model wasn't found.
			// Only filter when discovery has run (at least one model has Available==true).
			if snap.discoveryRan && !model.Available {
				continue
			}

			// Cost ceiling.
			if prefs.MaxCost > 0 && model.Cost.InputPerM > prefs.MaxCost {
				continue
			}

			// Min traits gate.
			if !model.Traits.MeetsMinimum(prefs.MinTraits) {
				continue
			}

			// Score.
			score := model.Traits.Score(prefs.Weights)
			candidates = append(candidates, candidate{model, source, score})
		}
	}

	if len(candidates) == 0 {
		return ResolvedAgent{}, ErrNoCandidate
	}

	// Rank — highest score wins.
	best := candidates[0]
	for _, c := range candidates[1:] {
		if c.score > best.score {
			best = c
		}
	}

	return resolve(best.model, best.source), nil
}

// resolve applies source modifiers to a model and returns a ResolvedAgent.
func resolve(model *ModelEntry, source *SourceEntry) ResolvedAgent {
	effContext := model.Context
	if source.Mods.ContextCap > 0 && source.Mods.ContextCap < effContext {
		effContext = source.Mods.ContextCap
	}

	overhead := source.Mods.TokenOverhead
	if overhead == 0 {
		overhead = 1.0
	}

	pipeline := source.Mods.Pipeline
	if pipeline == "" {
		pipeline = "direct"
	}

	return ResolvedAgent{
		Model:      model.ID,
		Provider:   model.Provider,
		Source:     source.Source,
		Traits:     model.Traits,
		EffContext: effContext,
		Overhead:   overhead,
		Pipeline:   pipeline,
		Cost:       model.Cost,
	}
}
