package referee

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/dpopsuev/troupe/signal"
)

// Referee scores agent behavior by watching the event stream.
type Referee struct {
	scorecard Scorecard
	ruleIdx   map[string]*ScorecardRule
	log       *slog.Logger

	mu     sync.Mutex
	score  int
	events []ScoredEvent // chrono log
}

// ScoredEvent records an event and its score contribution.
type ScoredEvent struct {
	Timestamp time.Time `json:"ts"`
	Kind      string    `json:"kind"`
	Source    string    `json:"source"`
	Weight    int       `json:"weight"`
	Rule      string    `json:"rule"` // "matched", "default", "unknown"
}

// Result is the final scorecard output.
type Result struct {
	Name      string                 `json:"name"`
	Pass      bool                   `json:"pass"`
	Score     int                    `json:"score"`
	Threshold int                    `json:"threshold"`
	Events    []ScoredEvent          `json:"events"`
	Buckets   map[string]BucketStats `json:"buckets"` // event kind → count + total weight
}

// BucketStats summarizes events by kind.
type BucketStats struct {
	Count       int `json:"count"`
	TotalWeight int `json:"total_weight"`
}

// New creates a Referee that scores events against the given Scorecard.
func New(sc Scorecard, opts ...RefereeOption) *Referee {
	r := &Referee{
		scorecard: sc,
		ruleIdx:   sc.ruleIndex(),
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// RefereeOption configures a Referee.
type RefereeOption func(*Referee)

// WithRefereeLogger sets structured logging for YELLOW instrumentation.
func WithRefereeLogger(l *slog.Logger) RefereeOption {
	return func(r *Referee) { r.log = l }
}

// Subscribe wires the Referee to an EventLog. Call once at setup time.
func (r *Referee) Subscribe(log signal.EventLog) {
	log.OnEmit(r.scoreEvent)
}

// scoreEvent is the EventLog.OnEmit callback.
func (r *Referee) scoreEvent(e signal.Event) {
	r.mu.Lock()
	defer r.mu.Unlock()

	weight := r.scorecard.UnknownEventWeight
	rule := "unknown"

	if sr, ok := r.ruleIdx[e.Kind]; ok {
		weight = sr.Weight
		rule = "matched"
	} else if r.scorecard.DefaultWeight != 0 {
		weight = r.scorecard.DefaultWeight
		rule = "default"
	}

	r.score += weight
	r.events = append(r.events, ScoredEvent{
		Timestamp: e.Timestamp,
		Kind:      e.Kind,
		Source:    e.Source,
		Weight:    weight,
		Rule:      rule,
	})
}

// Score returns the current score (can be called during a run).
func (r *Referee) Score() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.score
}

// Result returns the final scorecard with pass/fail, chrono dump, and buckets.
func (r *Referee) Result() Result {
	r.mu.Lock()
	defer r.mu.Unlock()

	buckets := make(map[string]BucketStats, len(r.events))
	for _, e := range r.events {
		b := buckets[e.Kind]
		b.Count++
		b.TotalWeight += e.Weight
		buckets[e.Kind] = b
	}

	evCopy := make([]ScoredEvent, len(r.events))
	copy(evCopy, r.events)

	res := Result{
		Name:      r.scorecard.Name,
		Pass:      r.score >= r.scorecard.Threshold,
		Score:     r.score,
		Threshold: r.scorecard.Threshold,
		Events:    evCopy,
		Buckets:   buckets,
	}

	// YELLOW: log result.
	if r.log != nil {
		level := slog.LevelInfo
		if !res.Pass {
			level = slog.LevelWarn
		}
		r.log.Log(context.Background(), level, "referee result",
			slog.String("action", res.Name),
			slog.String("status", fmt.Sprintf("pass=%t score=%d/%d events=%d", res.Pass, res.Score, res.Threshold, len(res.Events))),
		)
	}

	return res
}

// Reset clears the score and event log for reuse.
func (r *Referee) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.score = 0
	r.events = nil
}
