package protocol

// BudgetActual reports resource consumption for a single dispatch (worker → server).
type BudgetActual struct {
	TokensIn  int     `json:"tokens_in,omitempty"`
	TokensOut int     `json:"tokens_out,omitempty"`
	WallMs    int64   `json:"wall_ms,omitempty"`
	CostUSD   float64 `json:"cost_usd,omitempty"`
}

// BudgetRemaining reports remaining resource envelope (server → worker).
type BudgetRemaining struct {
	Tokens  int     `json:"tokens,omitempty"`
	TimeMs  int64   `json:"time_ms,omitempty"`
	CostUSD float64 `json:"cost_usd,omitempty"`
}

// BudgetSummary is the aggregated budget in a status response.
type BudgetSummary struct {
	ElapsedMs      int64   `json:"elapsed_ms"`
	TotalTokensIn  int     `json:"total_tokens_in"`
	TotalTokensOut int     `json:"total_tokens_out"`
	TotalCostUSD   float64 `json:"total_cost_usd"`
	AvgStepMs      int64   `json:"avg_step_ms,omitempty"`
	AvgStepTokens  int     `json:"avg_step_tokens,omitempty"`
}
