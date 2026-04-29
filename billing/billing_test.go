package billing_test

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dpopsuev/tangle/billing"
)

// ---------------------------------------------------------------------------
// InMemoryTracker
// ---------------------------------------------------------------------------

func TestInMemoryTracker_RecordAndSummary(t *testing.T) {
	tracker := billing.NewTracker()

	records := []billing.TokenRecord{
		{CaseID: "C1", Step: "F0", PromptBytes: 4000, ArtifactBytes: 400, PromptTokens: 1000, ArtifactTokens: 100, WallClockMs: 500},
		{CaseID: "C1", Step: "F1", PromptBytes: 8000, ArtifactBytes: 800, PromptTokens: 2000, ArtifactTokens: 200, WallClockMs: 600},
		{CaseID: "C2", Step: "F0", PromptBytes: 4000, ArtifactBytes: 400, PromptTokens: 1000, ArtifactTokens: 100, WallClockMs: 450},
		{CaseID: "C2", Step: "F1", PromptBytes: 6000, ArtifactBytes: 600, PromptTokens: 1500, ArtifactTokens: 150, WallClockMs: 550},
		{CaseID: "C1", Step: "F2", PromptBytes: 2000, ArtifactBytes: 200, PromptTokens: 500, ArtifactTokens: 50, WallClockMs: 300},
		{CaseID: "C2", Step: "F2", PromptBytes: 2400, ArtifactBytes: 240, PromptTokens: 600, ArtifactTokens: 60, WallClockMs: 320},
		{CaseID: "C1", Step: "F3", PromptBytes: 12000, ArtifactBytes: 1200, PromptTokens: 3000, ArtifactTokens: 300, WallClockMs: 800},
		{CaseID: "C2", Step: "F3", PromptBytes: 10000, ArtifactBytes: 1000, PromptTokens: 2500, ArtifactTokens: 250, WallClockMs: 700},
		{CaseID: "C1", Step: "F5", PromptBytes: 4000, ArtifactBytes: 400, PromptTokens: 1000, ArtifactTokens: 100, WallClockMs: 400},
		{CaseID: "C2", Step: "F5", PromptBytes: 3600, ArtifactBytes: 360, PromptTokens: 900, ArtifactTokens: 90, WallClockMs: 380},
	}

	for i := range records {
		tracker.Record(&records[i])
	}

	s := tracker.Summary()

	// Total prompt tokens: 1000+2000+1000+1500+500+600+3000+2500+1000+900 = 14000
	if s.TotalPromptTokens != 14000 {
		t.Errorf("TotalPromptTokens: got %d, want 14000", s.TotalPromptTokens)
	}
	// Total artifact tokens: 100+200+100+150+50+60+300+250+100+90 = 1400
	if s.TotalArtifactTokens != 1400 {
		t.Errorf("TotalArtifactTokens: got %d, want 1400", s.TotalArtifactTokens)
	}
	if s.TotalTokens != 15400 {
		t.Errorf("TotalTokens: got %d, want 15400", s.TotalTokens)
	}
	if s.TotalSteps != 10 {
		t.Errorf("TotalSteps: got %d, want 10", s.TotalSteps)
	}

	// Per-case: C1 has 5 records, C2 has 5 records.
	if len(s.PerCase) != 2 {
		t.Fatalf("PerCase count: got %d, want 2", len(s.PerCase))
	}
	c1 := s.PerCase["C1"]
	if c1.Steps != 5 {
		t.Errorf("C1 steps: got %d, want 5", c1.Steps)
	}
	if c1.PromptTokens != 7500 {
		t.Errorf("C1 PromptTokens: got %d, want 7500", c1.PromptTokens)
	}

	c2 := s.PerCase["C2"]
	if c2.Steps != 5 {
		t.Errorf("C2 steps: got %d, want 5", c2.Steps)
	}
	if c2.PromptTokens != 6500 {
		t.Errorf("C2 PromptTokens: got %d, want 6500", c2.PromptTokens)
	}

	// Per-step: F0 has 2 invocations, F1 has 2, etc.
	if len(s.PerStep) != 5 {
		t.Fatalf("PerStep count: got %d, want 5", len(s.PerStep))
	}
	f0 := s.PerStep["F0"]
	if f0.Invocations != 2 {
		t.Errorf("F0 invocations: got %d, want 2", f0.Invocations)
	}
	if f0.PromptTokens != 2000 {
		t.Errorf("F0 PromptTokens: got %d, want 2000", f0.PromptTokens)
	}

	// Cost: 14000 prompt / 1M * $3 = $0.042, 1400 artifact / 1M * $15 = $0.021 -> $0.063.
	expectedCost := 0.042 + 0.021
	if s.TotalCostUSD < expectedCost-0.001 || s.TotalCostUSD > expectedCost+0.001 {
		t.Errorf("TotalCostUSD: got %.6f, want ~%.6f", s.TotalCostUSD, expectedCost)
	}
}

func TestInMemoryTracker_EmptySummary(t *testing.T) {
	tracker := billing.NewTracker()
	s := tracker.Summary()

	if s.TotalTokens != 0 {
		t.Errorf("empty tracker TotalTokens: got %d, want 0", s.TotalTokens)
	}
	if s.TotalSteps != 0 {
		t.Errorf("empty tracker TotalSteps: got %d, want 0", s.TotalSteps)
	}
	if len(s.PerCase) != 0 {
		t.Errorf("empty tracker PerCase: got %d entries, want 0", len(s.PerCase))
	}
}

func TestInMemoryTracker_ConcurrentAccess(t *testing.T) {
	tracker := billing.NewTracker()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tracker.Record(&billing.TokenRecord{
				CaseID:         "C1",
				Step:           "F0",
				PromptBytes:    400,
				ArtifactBytes:  40,
				PromptTokens:   100,
				ArtifactTokens: 10,
				Timestamp:      time.Now(),
				WallClockMs:    50,
			})
		}()
	}
	wg.Wait()

	s := tracker.Summary()
	if s.TotalSteps != 100 {
		t.Errorf("concurrent TotalSteps: got %d, want 100", s.TotalSteps)
	}
	if s.TotalPromptTokens != 10000 {
		t.Errorf("concurrent TotalPromptTokens: got %d, want 10000", s.TotalPromptTokens)
	}
}

func TestInMemoryTracker_CustomCost(t *testing.T) {
	tracker := billing.NewTrackerWithCost(billing.CostConfig{
		InputPricePerMToken:  10.0,
		OutputPricePerMToken: 30.0,
	})
	tracker.Record(&billing.TokenRecord{
		CaseID:         "C1",
		Step:           "F0",
		PromptTokens:   1_000_000,
		ArtifactTokens: 500_000,
	})

	s := tracker.Summary()
	// 1M * $10/M = $10, 0.5M * $30/M = $15 -> $25.
	if s.TotalCostUSD < 24.99 || s.TotalCostUSD > 25.01 {
		t.Errorf("custom cost: got $%.2f, want $25.00", s.TotalCostUSD)
	}
}

func TestInMemoryTracker_OnRecordHook(t *testing.T) {
	tracker := billing.NewTracker()

	type hookCall struct {
		step    string
		costUSD float64
	}
	var hookCalls []hookCall
	tracker.OnRecord(func(r billing.TokenRecord, costUSD float64) {
		hookCalls = append(hookCalls, hookCall{r.Step, costUSD})
	})

	tracker.Record(&billing.TokenRecord{
		CaseID:         "C1",
		Step:           "recall",
		PromptTokens:   1000,
		ArtifactTokens: 200,
	})
	tracker.Record(&billing.TokenRecord{
		CaseID:         "C1",
		Step:           "triage",
		PromptTokens:   500,
		ArtifactTokens: 100,
	})

	if len(hookCalls) != 2 {
		t.Fatalf("hook calls = %d, want 2", len(hookCalls))
	}
	if hookCalls[0].step != "recall" {
		t.Errorf("hook[0].step = %q, want %q", hookCalls[0].step, "recall")
	}
	if hookCalls[1].step != "triage" {
		t.Errorf("hook[1].step = %q, want %q", hookCalls[1].step, "triage")
	}
	if hookCalls[0].costUSD <= 0 {
		t.Error("hook[0].costUSD should be positive")
	}
}

func TestInMemoryTracker_PerNodeAggregation(t *testing.T) {
	tracker := billing.NewTracker()

	tracker.Record(&billing.TokenRecord{
		CaseID:         "C1",
		Step:           "F0",
		Node:           "nodeA",
		PromptTokens:   500,
		ArtifactTokens: 50,
	})
	tracker.Record(&billing.TokenRecord{
		CaseID:         "C1",
		Step:           "F1",
		Node:           "nodeA",
		PromptTokens:   300,
		ArtifactTokens: 30,
	})
	tracker.Record(&billing.TokenRecord{
		CaseID:         "C1",
		Step:           "F2",
		Node:           "nodeB",
		PromptTokens:   200,
		ArtifactTokens: 20,
	})

	s := tracker.Summary()
	if len(s.PerNode) != 2 {
		t.Fatalf("PerNode count: got %d, want 2", len(s.PerNode))
	}
	nA := s.PerNode["nodeA"]
	if nA.Invocations != 2 {
		t.Errorf("nodeA invocations: got %d, want 2", nA.Invocations)
	}
	if nA.PromptTokens != 800 {
		t.Errorf("nodeA PromptTokens: got %d, want 800", nA.PromptTokens)
	}
}

func TestInMemoryTracker_ImplementsTrackerInterface(t *testing.T) {
	var _ billing.Tracker = billing.NewTracker()
}

// ---------------------------------------------------------------------------
// EstimateTokens
// ---------------------------------------------------------------------------

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		bytes int
		want  int
	}{
		{0, 0},
		{-1, 0},
		{4, 1},
		{100, 25},
		{4000, 1000},
	}
	for _, tt := range tests {
		got := billing.EstimateTokens(tt.bytes)
		if got != tt.want {
			t.Errorf("EstimateTokens(%d): got %d, want %d", tt.bytes, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// CostBill
// ---------------------------------------------------------------------------

func sampleTokenSummary() *billing.TokenSummary {
	return &billing.TokenSummary{
		TotalPromptTokens:   100_000,
		TotalArtifactTokens: 5_000,
		TotalTokens:         105_000,
		TotalCostUSD:        0.375,
		TotalSteps:          12,
		TotalWallClockMs:    60_000,
		PerCase: map[string]billing.CaseTokenSummary{
			"C1": {PromptTokens: 60000, ArtifactTokens: 3000, TotalTokens: 63000, Steps: 7, WallClockMs: 35000},
			"C2": {PromptTokens: 40000, ArtifactTokens: 2000, TotalTokens: 42000, Steps: 5, WallClockMs: 25000},
		},
		PerStep: map[string]billing.StepTokenSummary{
			"STEP_A": {PromptTokens: 20000, ArtifactTokens: 1000, TotalTokens: 21000, Invocations: 2},
			"STEP_B": {PromptTokens: 30000, ArtifactTokens: 2000, TotalTokens: 32000, Invocations: 2},
			"STEP_C": {PromptTokens: 50000, ArtifactTokens: 2000, TotalTokens: 52000, Invocations: 8},
		},
	}
}

func TestBuildCostBill_Nil(t *testing.T) {
	bill := billing.BuildCostBill(nil)
	if bill != nil {
		t.Error("nil TokenSummary should produce nil bill")
	}
}

func TestBuildCostBill_Basic(t *testing.T) {
	bill := billing.BuildCostBill(sampleTokenSummary())
	if bill == nil {
		t.Fatal("expected non-nil bill")
	}
	if bill.CaseCount != 2 {
		t.Errorf("CaseCount: want 2, got %d", bill.CaseCount)
	}
	if len(bill.CaseLines) != 2 {
		t.Errorf("CaseLines: want 2, got %d", len(bill.CaseLines))
	}
	if len(bill.StepLines) != 3 {
		t.Errorf("StepLines: want 3, got %d", len(bill.StepLines))
	}
	if bill.Title != "Cost Bill" {
		t.Errorf("Title: want 'Cost Bill', got %q", bill.Title)
	}
}

func TestBuildCostBill_WithOptions(t *testing.T) {
	bill := billing.BuildCostBill(sampleTokenSummary(),
		billing.WithTitle("TokiMeter"),
		billing.WithSubtitle("scenario: test | backend: llm"),
		billing.WithStepOrder([]string{"STEP_C", "STEP_A", "STEP_B"}),
		billing.WithStepNames(func(s string) string {
			names := map[string]string{"STEP_A": "Alpha", "STEP_B": "Beta", "STEP_C": "Gamma"}
			return names[s]
		}),
		billing.WithCaseLabels(func(id string) string { return "case-" + id }),
		billing.WithCaseDetails(func(id string) string { return "detail for " + id }),
	)

	if bill.Title != "TokiMeter" {
		t.Errorf("Title: got %q", bill.Title)
	}
	if bill.StepLines[0].Step != "STEP_C" {
		t.Errorf("step order: first should be STEP_C, got %s", bill.StepLines[0].Step)
	}
	if bill.StepLines[0].DisplayName != "Gamma" {
		t.Errorf("display name: want Gamma, got %s", bill.StepLines[0].DisplayName)
	}
	if bill.CaseLines[0].Label != "case-C1" {
		t.Errorf("case label: want case-C1, got %s", bill.CaseLines[0].Label)
	}
	if bill.CaseLines[0].Detail != "detail for C1" {
		t.Errorf("case detail: want 'detail for C1', got %s", bill.CaseLines[0].Detail)
	}
}

func TestBuildCostBill_StepOrderPartial(t *testing.T) {
	bill := billing.BuildCostBill(sampleTokenSummary(),
		billing.WithStepOrder([]string{"STEP_B"}),
	)
	if bill.StepLines[0].Step != "STEP_B" {
		t.Errorf("first step should be STEP_B, got %s", bill.StepLines[0].Step)
	}
	// Remaining steps appear after in alphabetical order.
	if len(bill.StepLines) != 3 {
		t.Fatalf("want 3 steps, got %d", len(bill.StepLines))
	}
	if bill.StepLines[1].Step != "STEP_A" {
		t.Errorf("second step should be STEP_A, got %s", bill.StepLines[1].Step)
	}
}

func TestBuildCostBill_CustomCostConfig(t *testing.T) {
	bill := billing.BuildCostBill(sampleTokenSummary(),
		billing.WithCostConfig(billing.CostConfig{InputPricePerMToken: 1.0, OutputPricePerMToken: 2.0}),
	)
	// C1: 60000 in * 1/M + 3000 out * 2/M = 0.06 + 0.006 = 0.066.
	for _, cl := range bill.CaseLines {
		if cl.CaseID == "C1" {
			expected := float64(60000)/1e6*1.0 + float64(3000)/1e6*2.0
			if cl.CostUSD != expected {
				t.Errorf("C1 cost: want %f, got %f", expected, cl.CostUSD)
			}
		}
	}
}

func TestFormatCostBill_Nil(t *testing.T) {
	if billing.FormatCostBill(nil) != "" {
		t.Error("nil bill should produce empty string")
	}
}

func TestFormatCostBill_Markdown(t *testing.T) {
	bill := billing.BuildCostBill(sampleTokenSummary(),
		billing.WithTitle("TokiMeter"),
		billing.WithSubtitle("test scenario"),
	)
	md := billing.FormatCostBill(bill)

	checks := []string{
		"# TokiMeter",
		"## Summary",
		"## Per-case costs",
		"## Per-step costs",
		"| Case |",
		"| Step |",
		"| **TOTAL**",
		"test scenario",
		"105.0K",
		"C1",
		"C2",
	}
	for _, check := range checks {
		if !strings.Contains(md, check) {
			t.Errorf("markdown missing: %q", check)
		}
	}
}

func TestFormatCostBill_NoCases(t *testing.T) {
	ts := &billing.TokenSummary{
		TotalPromptTokens:   1000,
		TotalArtifactTokens: 500,
		TotalTokens:         1500,
		TotalSteps:          1,
	}
	bill := billing.BuildCostBill(ts)
	md := billing.FormatCostBill(bill)
	if strings.Contains(md, "Per-case costs") {
		t.Error("should not show per-case section when no cases")
	}
}
