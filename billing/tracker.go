package billing

import (
	"sync"
	"time"
)

// Tracker records and summarizes token usage.
type Tracker interface {
	Record(r *TokenRecord)
	Summary() TokenSummary
}

// TokenRecord captures token usage for a single circuit step dispatch.
type TokenRecord struct {
	CaseID         string    `json:"case_id"`
	Step           string    `json:"step"`
	Node           string    `json:"node,omitempty"`
	PromptBytes    int       `json:"prompt_bytes"`
	ArtifactBytes  int       `json:"artifact_bytes"`
	PromptTokens   int       `json:"prompt_tokens"`
	ArtifactTokens int       `json:"artifact_tokens"`
	Timestamp      time.Time `json:"timestamp"`
	WallClockMs    int64     `json:"wall_clock_ms"`
}

// CaseTokenSummary aggregates token usage for a single case.
type CaseTokenSummary struct {
	PromptTokens   int   `json:"prompt_tokens"`
	ArtifactTokens int   `json:"artifact_tokens"`
	TotalTokens    int   `json:"total_tokens"`
	Steps          int   `json:"steps"`
	WallClockMs    int64 `json:"wall_clock_ms"`
}

// StepTokenSummary aggregates token usage across all cases for a single step.
type StepTokenSummary struct {
	PromptTokens   int `json:"prompt_tokens"`
	ArtifactTokens int `json:"artifact_tokens"`
	TotalTokens    int `json:"total_tokens"`
	Invocations    int `json:"invocations"`
}

// CostConfig holds pricing for token cost estimation.
type CostConfig struct {
	InputPricePerMToken  float64
	OutputPricePerMToken float64
}

// DefaultCostConfig returns typical LLM pricing.
func DefaultCostConfig() CostConfig {
	return CostConfig{
		InputPricePerMToken:  3.0,
		OutputPricePerMToken: 15.0,
	}
}

// TokenSummary is the aggregate view of all token usage in a calibration run.
type TokenSummary struct {
	TotalPromptTokens   int                         `json:"total_prompt_tokens"`
	TotalArtifactTokens int                         `json:"total_artifact_tokens"`
	TotalTokens         int                         `json:"total_tokens"`
	TotalCostUSD        float64                     `json:"total_cost_usd"`
	PerCase             map[string]CaseTokenSummary `json:"per_case"`
	PerStep             map[string]StepTokenSummary `json:"per_step"`
	PerNode             map[string]StepTokenSummary `json:"per_node,omitempty"`
	TotalSteps          int                         `json:"total_steps"`
	TotalWallClockMs    int64                       `json:"total_wall_clock_ms"`
}

// TokenRecordHook is called after each token record is appended.
// Use it to bridge token tracking with external systems (e.g., Prometheus).
type TokenRecordHook func(r TokenRecord, costUSD float64)

// InMemoryTracker is a thread-safe in-memory token tracker.
type InMemoryTracker struct {
	mu      sync.Mutex
	records []TokenRecord
	cost    CostConfig
	hooks   []TokenRecordHook
}

// NewTracker creates an InMemoryTracker with default cost config.
func NewTracker() *InMemoryTracker {
	return &InMemoryTracker{cost: DefaultCostConfig()}
}

// NewTrackerWithCost creates an InMemoryTracker with custom pricing.
func NewTrackerWithCost(c CostConfig) *InMemoryTracker {
	return &InMemoryTracker{cost: c}
}

// OnRecord registers a hook invoked after each Record call.
func (t *InMemoryTracker) OnRecord(hook TokenRecordHook) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.hooks = append(t.hooks, hook)
}

// Record appends a token record (thread-safe) and invokes hooks.
func (t *InMemoryTracker) Record(r *TokenRecord) {
	t.mu.Lock()
	t.records = append(t.records, *r)
	inputCost := float64(r.PromptTokens) / 1_000_000 * t.cost.InputPricePerMToken
	outputCost := float64(r.ArtifactTokens) / 1_000_000 * t.cost.OutputPricePerMToken
	costUSD := inputCost + outputCost
	hooks := make([]TokenRecordHook, len(t.hooks))
	copy(hooks, t.hooks)
	t.mu.Unlock()

	for _, h := range hooks {
		h(*r, costUSD)
	}
}

// Summary computes the aggregate token summary.
func (t *InMemoryTracker) Summary() TokenSummary {
	t.mu.Lock()
	defer t.mu.Unlock()

	s := TokenSummary{
		PerCase: make(map[string]CaseTokenSummary),
		PerStep: make(map[string]StepTokenSummary),
		PerNode: make(map[string]StepTokenSummary),
	}

	for _, r := range t.records {
		s.TotalPromptTokens += r.PromptTokens
		s.TotalArtifactTokens += r.ArtifactTokens
		s.TotalSteps++
		s.TotalWallClockMs += r.WallClockMs

		cs := s.PerCase[r.CaseID]
		cs.PromptTokens += r.PromptTokens
		cs.ArtifactTokens += r.ArtifactTokens
		cs.TotalTokens += r.PromptTokens + r.ArtifactTokens
		cs.Steps++
		cs.WallClockMs += r.WallClockMs
		s.PerCase[r.CaseID] = cs

		ss := s.PerStep[r.Step]
		ss.PromptTokens += r.PromptTokens
		ss.ArtifactTokens += r.ArtifactTokens
		ss.TotalTokens += r.PromptTokens + r.ArtifactTokens
		ss.Invocations++
		s.PerStep[r.Step] = ss

		if r.Node != "" {
			ns := s.PerNode[r.Node]
			ns.PromptTokens += r.PromptTokens
			ns.ArtifactTokens += r.ArtifactTokens
			ns.TotalTokens += r.PromptTokens + r.ArtifactTokens
			ns.Invocations++
			s.PerNode[r.Node] = ns
		}
	}

	s.TotalTokens = s.TotalPromptTokens + s.TotalArtifactTokens

	inputCost := float64(s.TotalPromptTokens) / 1_000_000 * t.cost.InputPricePerMToken
	outputCost := float64(s.TotalArtifactTokens) / 1_000_000 * t.cost.OutputPricePerMToken
	s.TotalCostUSD = inputCost + outputCost

	return s
}

// EstimateTokens converts byte count to estimated token count (bytes / 4).
func EstimateTokens(bytes int) int {
	if bytes <= 0 {
		return 0
	}
	return bytes / 4
}
