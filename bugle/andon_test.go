package bugle

import "testing"

func TestWorse(t *testing.T) {
	tests := []struct {
		a, b uint8
		want bool
	}{
		{PriorityNominal, PriorityNominal, false},
		{PriorityDegraded, PriorityNominal, true},
		{PriorityNominal, PriorityDegraded, false},
		{PriorityFailure, PriorityDegraded, true},
		{PriorityDead, PriorityFailure, true},
		{PriorityDead, PriorityDead, false},
		{PriorityBlocked, PriorityFailure, true},
		{32, PriorityNominal, true},   // custom level
		{32, PriorityDegraded, false}, // custom below degraded
	}
	for _, tt := range tests {
		if got := Worse(tt.a, tt.b); got != tt.want {
			t.Errorf("Worse(%d, %d) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestWorstPriority(t *testing.T) {
	tests := []struct {
		name       string
		priorities []uint8
		want       uint8
	}{
		{"empty", nil, 0},
		{"all nominal", []uint8{0, 0}, 0},
		{"one degraded", []uint8{0, 64, 0}, 64},
		{"mixed", []uint8{0, 128, 64}, 128},
		{"dead wins", []uint8{64, 255, 128}, 255},
		{"custom mixed", []uint8{32, 64, 16}, 64},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := WorstPriority(tt.priorities...); got != tt.want {
				t.Errorf("WorstPriority() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestPriorityOf(t *testing.T) {
	for level, want := range ReservedLevels {
		got, ok := PriorityOf(level)
		if !ok {
			t.Errorf("PriorityOf(%q) not found", level)
		}
		if got != want {
			t.Errorf("PriorityOf(%q) = %d, want %d", level, got, want)
		}
	}
	_, ok := PriorityOf("custom")
	if ok {
		t.Error("PriorityOf(custom) should return false")
	}
}

func TestDefaultVocabulary(t *testing.T) {
	vocab := DefaultVocabulary()
	if len(vocab) != 5 {
		t.Errorf("DefaultVocabulary() has %d entries, want 5", len(vocab))
	}
	// Verify ordering
	for i := 1; i < len(vocab); i++ {
		if vocab[i].Priority < vocab[i-1].Priority {
			t.Errorf("vocabulary not sorted: %s(%d) < %s(%d)",
				vocab[i].Name, vocab[i].Priority, vocab[i-1].Name, vocab[i-1].Priority)
		}
	}
}

func TestProtocolError(t *testing.T) {
	err := &ProtocolError{Code: ErrCodeNoActiveSession, Message: "session not found"}
	want := "no_active_session: session not found"
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestValidStatuses(t *testing.T) {
	for _, s := range []SubmitStatus{StatusOk, StatusBlocked, StatusResolved, StatusCanceled, StatusError} {
		if !ValidStatuses[s] {
			t.Errorf("ValidStatuses[%q] = false, want true", s)
		}
	}
	if ValidStatuses["bogus"] {
		t.Error("ValidStatuses[bogus] = true, want false")
	}
}
