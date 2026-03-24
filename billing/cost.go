package billing

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// CostBill is a structured cost report for any dispatch or calibration run.
type CostBill struct {
	Title       string
	Subtitle    string
	Timestamp   string
	CaseLines   []CostBillCaseLine
	StepLines   []CostBillStepLine
	TotalIn     int
	TotalOut    int
	TotalTokens int
	TotalCost   float64
	TotalSteps  int
	WallClock   time.Duration
	CaseCount   int
}

// CostBillCaseLine is one row in the per-case cost table.
type CostBillCaseLine struct {
	CaseID  string
	Label   string
	Detail  string
	Steps   int
	In      int
	Out     int
	Total   int
	CostUSD float64
	WallMs  int64
}

// CostBillStepLine is one row in the per-step cost table.
type CostBillStepLine struct {
	Step        string
	DisplayName string
	Invocations int
	In          int
	Out         int
	Total       int
	CostUSD     float64
}

// CostBillOption configures CostBill construction via functional options.
type CostBillOption func(*costBillConfig)

type costBillConfig struct {
	title        string
	subtitle     string
	cost         CostConfig
	stepOrder    []string
	stepNameFn   func(string) string
	caseLabelFn  func(string) string
	caseDetailFn func(string) string
}

// WithTitle sets the bill title (default: "Cost Bill").
func WithTitle(t string) CostBillOption {
	return func(c *costBillConfig) { c.title = t }
}

// WithSubtitle sets a subtitle line.
func WithSubtitle(s string) CostBillOption {
	return func(c *costBillConfig) { c.subtitle = s }
}

// WithCostConfig overrides the default pricing.
func WithCostConfig(cc CostConfig) CostBillOption {
	return func(c *costBillConfig) { c.cost = cc }
}

// WithStepOrder defines the ordering for step rows.
// Steps not in this list appear at the end in alphabetical order.
func WithStepOrder(order []string) CostBillOption {
	return func(c *costBillConfig) { c.stepOrder = order }
}

// WithStepNames provides a function to map step IDs to display names.
func WithStepNames(fn func(string) string) CostBillOption {
	return func(c *costBillConfig) { c.stepNameFn = fn }
}

// WithCaseLabels provides a function to map case IDs to labels (first column).
func WithCaseLabels(fn func(string) string) CostBillOption {
	return func(c *costBillConfig) { c.caseLabelFn = fn }
}

// WithCaseDetails provides a function to map case IDs to detail strings.
func WithCaseDetails(fn func(string) string) CostBillOption {
	return func(c *costBillConfig) { c.caseDetailFn = fn }
}

// BuildCostBill constructs a CostBill from a TokenSummary.
func BuildCostBill(ts *TokenSummary, opts ...CostBillOption) *CostBill {
	if ts == nil {
		return nil
	}

	cfg := costBillConfig{
		title: "Cost Bill",
		cost:  DefaultCostConfig(),
	}
	for _, o := range opts {
		o(&cfg)
	}

	bill := &CostBill{
		Title:       cfg.title,
		Subtitle:    cfg.subtitle,
		Timestamp:   time.Now().UTC().Format("2006-01-02 15:04 UTC"),
		TotalIn:     ts.TotalPromptTokens,
		TotalOut:    ts.TotalArtifactTokens,
		TotalTokens: ts.TotalTokens,
		TotalCost:   ts.TotalCostUSD,
		TotalSteps:  ts.TotalSteps,
		WallClock:   time.Duration(ts.TotalWallClockMs) * time.Millisecond,
		CaseCount:   len(ts.PerCase),
	}

	// Per-case lines.
	caseIDs := make([]string, 0, len(ts.PerCase))
	for id := range ts.PerCase {
		caseIDs = append(caseIDs, id)
	}
	sort.Strings(caseIDs)

	for _, id := range caseIDs {
		cs := ts.PerCase[id]
		inCost := float64(cs.PromptTokens) / 1_000_000 * cfg.cost.InputPricePerMToken
		outCost := float64(cs.ArtifactTokens) / 1_000_000 * cfg.cost.OutputPricePerMToken

		label := id
		if cfg.caseLabelFn != nil {
			if l := cfg.caseLabelFn(id); l != "" {
				label = l
			}
		}
		detail := ""
		if cfg.caseDetailFn != nil {
			detail = cfg.caseDetailFn(id)
		}

		bill.CaseLines = append(bill.CaseLines, CostBillCaseLine{
			CaseID:  id,
			Label:   label,
			Detail:  detail,
			Steps:   cs.Steps,
			In:      cs.PromptTokens,
			Out:     cs.ArtifactTokens,
			Total:   cs.TotalTokens,
			CostUSD: inCost + outCost,
			WallMs:  cs.WallClockMs,
		})
	}

	// Per-step lines.
	bill.StepLines = buildStepLines(ts, &cfg)

	return bill
}

func buildStepLines(ts *TokenSummary, cfg *costBillConfig) []CostBillStepLine {
	lines := make([]CostBillStepLine, 0, len(ts.PerStep))

	ordered := make(map[string]bool, len(cfg.stepOrder))
	for _, step := range cfg.stepOrder {
		ordered[step] = true
		ss, ok := ts.PerStep[step]
		if !ok {
			continue
		}
		lines = append(lines, makeStepLine(step, ss, cfg))
	}

	// Remaining steps in alphabetical order.
	var remaining []string
	for step := range ts.PerStep {
		if !ordered[step] {
			remaining = append(remaining, step)
		}
	}
	sort.Strings(remaining)
	for _, step := range remaining {
		lines = append(lines, makeStepLine(step, ts.PerStep[step], cfg))
	}

	return lines
}

func makeStepLine(step string, ss StepTokenSummary, cfg *costBillConfig) CostBillStepLine {
	inCost := float64(ss.PromptTokens) / 1_000_000 * cfg.cost.InputPricePerMToken
	outCost := float64(ss.ArtifactTokens) / 1_000_000 * cfg.cost.OutputPricePerMToken
	displayName := step
	if cfg.stepNameFn != nil {
		if n := cfg.stepNameFn(step); n != "" {
			displayName = n
		}
	}
	return CostBillStepLine{
		Step:        step,
		DisplayName: displayName,
		Invocations: ss.Invocations,
		In:          ss.PromptTokens,
		Out:         ss.ArtifactTokens,
		Total:       ss.TotalTokens,
		CostUSD:     inCost + outCost,
	}
}

// FormatCostBill produces a markdown-formatted cost bill.
func FormatCostBill(bill *CostBill) string {
	if bill == nil {
		return ""
	}

	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("# %s\n\n", bill.Title))
	if bill.Subtitle != "" {
		b.WriteString(fmt.Sprintf("> %s | %s\n\n", bill.Subtitle, bill.Timestamp))
	} else {
		b.WriteString(fmt.Sprintf("> %s\n\n", bill.Timestamp))
	}

	// Summary.
	b.WriteString("## Summary\n\n")
	b.WriteString(fmtMDTable(
		[]string{"Metric", "Value"},
		summaryRows(bill),
		nil,
	))
	b.WriteString("\n\n")

	// Per-case.
	if len(bill.CaseLines) > 0 {
		b.WriteString("## Per-case costs\n\n")
		var rows [][]string
		for _, cl := range bill.CaseLines {
			detail := cl.Detail
			if detail == "" {
				detail = "-"
			}
			rows = append(rows, []string{
				cl.Label,
				truncate(detail, 30),
				fmt.Sprintf("%d", cl.Steps),
				fmtTokens(cl.In),
				fmtTokens(cl.Out),
				fmtTokens(cl.Total),
				fmt.Sprintf("$%.4f", cl.CostUSD),
				fmt.Sprintf("%.1fs", float64(cl.WallMs)/1000.0),
			})
		}
		b.WriteString(fmtMDTable(
			[]string{"Case", "Detail", "Steps", "In", "Out", "Total", "Cost", "Time"},
			rows,
			nil,
		))
		b.WriteString("\n\n")
	}

	// Per-step.
	if len(bill.StepLines) > 0 {
		b.WriteString("## Per-step costs\n\n")
		var rows [][]string
		for _, sl := range bill.StepLines {
			rows = append(rows, []string{
				sl.DisplayName,
				fmt.Sprintf("%d", sl.Invocations),
				fmtTokens(sl.In),
				fmtTokens(sl.Out),
				fmtTokens(sl.Total),
				fmt.Sprintf("$%.4f", sl.CostUSD),
			})
		}
		footer := []string{
			"**TOTAL**",
			fmt.Sprintf("**%d**", bill.TotalSteps),
			fmt.Sprintf("**%s**", fmtTokens(bill.TotalIn)),
			fmt.Sprintf("**%s**", fmtTokens(bill.TotalOut)),
			fmt.Sprintf("**%s**", fmtTokens(bill.TotalTokens)),
			fmt.Sprintf("**$%.4f**", bill.TotalCost),
		}
		b.WriteString(fmtMDTable(
			[]string{"Step", "Calls", "In", "Out", "Total", "Cost"},
			rows,
			footer,
		))
		b.WriteString("\n\n")
	}

	return b.String()
}

func summaryRows(bill *CostBill) [][]string {
	rows := [][]string{
		{"Cases", fmt.Sprintf("%d", bill.CaseCount)},
		{"Steps", fmt.Sprintf("%d", bill.TotalSteps)},
		{"Input tokens", fmtTokens(bill.TotalIn)},
		{"Output tokens", fmtTokens(bill.TotalOut)},
		{"**Total tokens**", fmt.Sprintf("**%s**", fmtTokens(bill.TotalTokens))},
		{"**Total cost**", fmt.Sprintf("**$%.4f**", bill.TotalCost)},
		{"Wall clock", fmtDuration(bill.WallClock)},
	}
	if bill.CaseCount > 0 {
		rows = append(rows, []string{
			"Avg per case",
			fmt.Sprintf("%s ($%.4f)",
				fmtTokens(bill.TotalTokens/bill.CaseCount),
				bill.TotalCost/float64(bill.CaseCount)),
		})
	}
	return rows
}
