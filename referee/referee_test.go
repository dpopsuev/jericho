package referee_test

import (
	"testing"

	"github.com/dpopsuev/tangle/referee"
	"github.com/dpopsuev/tangle/signal"
)

func TestParseScorecard_Valid(t *testing.T) {
	yaml := []byte(`
name: test
threshold: 10
default_weight: 1
rules:
  - on: success
    weight: 5
  - on: error
    weight: -10
`)
	sc, err := referee.ParseScorecard(yaml)
	if err != nil {
		t.Fatalf("ParseScorecard: %v", err)
	}
	if sc.Name != "test" {
		t.Fatalf("name = %q, want test", sc.Name)
	}
	if sc.Threshold != 10 {
		t.Fatalf("threshold = %d, want 10", sc.Threshold)
	}
	if len(sc.Rules) != 2 {
		t.Fatalf("rules = %d, want 2", len(sc.Rules))
	}
}

func TestParseScorecard_MissingName(t *testing.T) {
	yaml := []byte(`threshold: 10`)
	_, err := referee.ParseScorecard(yaml)
	if err == nil {
		t.Fatal("should error on missing name")
	}
}

func TestParseScorecard_InvalidYAML(t *testing.T) {
	_, err := referee.ParseScorecard([]byte(`{{{`))
	if err == nil {
		t.Fatal("should error on invalid YAML")
	}
}

func testScorecard() referee.Scorecard {
	return referee.Scorecard{
		Name:               "unit-test",
		Threshold:          10,
		DefaultWeight:      1,
		UnknownEventWeight: 0,
		Rules: []referee.ScorecardRule{
			{On: "success", Weight: 5},
			{On: "error", Weight: -10},
		},
	}
}

func TestReferee_MatchedRule(t *testing.T) {
	log := signal.NewMemLog()
	ref := referee.New(testScorecard())
	ref.Subscribe(log)

	log.Emit(signal.Event{Kind: "success", Source: "test"})

	if ref.Score() != 5 {
		t.Fatalf("score = %d, want 5", ref.Score())
	}
}

func TestReferee_DefaultWeight(t *testing.T) {
	log := signal.NewMemLog()
	ref := referee.New(testScorecard())
	ref.Subscribe(log)

	log.Emit(signal.Event{Kind: "unlisted_event", Source: "test"})

	// No matching rule → DefaultWeight (1) used as fallback.
	if ref.Score() != 1 {
		t.Fatalf("score = %d, want 1 (default weight for unlisted)", ref.Score())
	}
}

func TestReferee_UnknownEventWeight(t *testing.T) {
	sc := referee.Scorecard{
		Name:               "unknown-test",
		Threshold:          0,
		DefaultWeight:      0,
		UnknownEventWeight: -1,
		Rules:              []referee.ScorecardRule{{On: "known", Weight: 5}},
	}
	log := signal.NewMemLog()
	ref := referee.New(sc)
	ref.Subscribe(log)

	log.Emit(signal.Event{Kind: "totally_unknown", Source: "test"})

	if ref.Score() != -1 {
		t.Fatalf("score = %d, want -1 (unknown event weight)", ref.Score())
	}
}

func TestReferee_NegativeScore(t *testing.T) {
	log := signal.NewMemLog()
	ref := referee.New(testScorecard())
	ref.Subscribe(log)

	log.Emit(signal.Event{Kind: "error", Source: "test"})

	if ref.Score() != -10 {
		t.Fatalf("score = %d, want -10", ref.Score())
	}
}

func TestReferee_Result_Pass(t *testing.T) {
	log := signal.NewMemLog()
	ref := referee.New(testScorecard())
	ref.Subscribe(log)

	log.Emit(signal.Event{Kind: "success", Source: "a"})
	log.Emit(signal.Event{Kind: "success", Source: "b"})
	log.Emit(signal.Event{Kind: "success", Source: "c"})

	result := ref.Result()
	if !result.Pass {
		t.Fatalf("should pass: score=%d threshold=%d", result.Score, result.Threshold)
	}
	if result.Score != 15 {
		t.Fatalf("score = %d, want 15", result.Score)
	}
}

func TestReferee_Result_Fail(t *testing.T) {
	log := signal.NewMemLog()
	ref := referee.New(testScorecard())
	ref.Subscribe(log)

	log.Emit(signal.Event{Kind: "error", Source: "test"})

	result := ref.Result()
	if result.Pass {
		t.Fatal("should fail with score -10, threshold 10")
	}
}

func TestReferee_Result_Buckets(t *testing.T) {
	log := signal.NewMemLog()
	ref := referee.New(testScorecard())
	ref.Subscribe(log)

	log.Emit(signal.Event{Kind: "success", Source: "a"})
	log.Emit(signal.Event{Kind: "success", Source: "b"})
	log.Emit(signal.Event{Kind: "error", Source: "c"})

	result := ref.Result()
	if result.Buckets["success"].Count != 2 {
		t.Fatalf("success count = %d, want 2", result.Buckets["success"].Count)
	}
	if result.Buckets["error"].Count != 1 {
		t.Fatalf("error count = %d, want 1", result.Buckets["error"].Count)
	}
	if result.Buckets["success"].TotalWeight != 10 {
		t.Fatalf("success weight = %d, want 10", result.Buckets["success"].TotalWeight)
	}
}

func TestReferee_Reset(t *testing.T) {
	log := signal.NewMemLog()
	ref := referee.New(testScorecard())
	ref.Subscribe(log)

	log.Emit(signal.Event{Kind: "success", Source: "test"})
	if ref.Score() != 5 {
		t.Fatalf("pre-reset score = %d, want 5", ref.Score())
	}

	ref.Reset()

	if ref.Score() != 0 {
		t.Fatalf("post-reset score = %d, want 0", ref.Score())
	}

	result := ref.Result()
	if len(result.Events) != 0 {
		t.Fatalf("post-reset events = %d, want 0", len(result.Events))
	}
}

func TestReferee_MultipleEvents_ChronoOrder(t *testing.T) {
	log := signal.NewMemLog()
	ref := referee.New(testScorecard())
	ref.Subscribe(log)

	log.Emit(signal.Event{Kind: "success", Source: "first"})
	log.Emit(signal.Event{Kind: "error", Source: "second"})
	log.Emit(signal.Event{Kind: "success", Source: "third"})

	result := ref.Result()
	if len(result.Events) != 3 {
		t.Fatalf("events = %d, want 3", len(result.Events))
	}
	if result.Events[0].Source != "first" {
		t.Fatalf("first event source = %q, want first", result.Events[0].Source)
	}
	if result.Events[2].Source != "third" {
		t.Fatalf("third event source = %q, want third", result.Events[2].Source)
	}
}
