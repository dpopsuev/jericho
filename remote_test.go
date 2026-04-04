package troupe_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/dpopsuev/troupe"
	"github.com/dpopsuev/troupe/testkit"
)

func TestRemoteBroker_PickAndSpawnRoundTrip(t *testing.T) {
	server := httptest.NewServer(mockBrokerHandler())
	defer server.Close()

	broker := troupe.NewBroker(server.URL)

	configs, err := broker.Pick(context.Background(), troupe.Preferences{Count: 2})
	if err != nil {
		t.Fatalf("Pick: %v", err)
	}
	if len(configs) != 2 {
		t.Errorf("Pick returned %d configs, want 2", len(configs))
	}

	actor, err := broker.Spawn(context.Background(), troupe.ActorConfig{Role: "worker"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}

	response, err := actor.Perform(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Perform: %v", err)
	}
	if response == "" {
		t.Error("Perform returned empty response")
	}

	if !actor.Ready() {
		t.Error("Ready() = false, want true")
	}

	if err := actor.Kill(context.Background()); err != nil {
		t.Fatalf("Kill: %v", err)
	}
}

func TestSSEDirector_RoundTrip(t *testing.T) {
	director := &testkit.LinearDirector{
		Steps: []testkit.Step{
			{Name: "step-1", Prompt: "do thing 1"},
			{Name: "step-2", Prompt: "do thing 2"},
		},
	}
	mockBroker := testkit.NewMockBroker(1)

	server := httptest.NewServer(troupe.DirectorHandler(director, mockBroker))
	defer server.Close()

	sseDirector := troupe.ConnectDirector(server.URL)
	events, err := sseDirector.Direct(context.Background(), nil)
	if err != nil {
		t.Fatalf("Direct: %v", err)
	}

	kinds := make([]troupe.EventKind, 0, 5) //nolint:mnd // expected event count
	for ev := range events {
		kinds = append(kinds, ev.Kind)
	}

	if len(kinds) != 5 { //nolint:mnd // 2 steps × (Started + Completed) + Done
		t.Fatalf("received %d events, want 5: %v", len(kinds), kinds)
	}
	if kinds[len(kinds)-1] != troupe.Done {
		t.Errorf("last event = %s, want done", kinds[len(kinds)-1])
	}
}

func TestProxyActor_ConcurrentPerform(t *testing.T) {
	server := httptest.NewServer(mockBrokerHandler())
	defer server.Close()

	broker := troupe.NewBroker(server.URL)
	actor, err := broker.Spawn(context.Background(), troupe.ActorConfig{Role: "worker"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}

	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := actor.Perform(context.Background(), "concurrent")
			if err != nil {
				t.Errorf("concurrent Perform: %v", err)
			}
		}()
	}
	wg.Wait()
}

func mockBrokerHandler() http.Handler {
	mux := http.NewServeMux()
	var spawnCount int
	actors := make(map[string]*testkit.MockActor)
	var mu sync.Mutex

	mux.HandleFunc("POST /pick", func(w http.ResponseWriter, r *http.Request) {
		var prefs troupe.Preferences
		json.NewDecoder(r.Body).Decode(&prefs) //nolint:errcheck // test helper
		count := prefs.Count
		if count <= 0 {
			count = 1
		}
		configs := make([]troupe.ActorConfig, count)
		for i := range count {
			configs[i] = troupe.ActorConfig{Role: fmt.Sprintf("actor-%d", i+1)}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(configs) //nolint:errcheck // test helper
	})

	mux.HandleFunc("POST /spawn", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		spawnCount++
		id := fmt.Sprintf("actor-%d", spawnCount)
		actors[id] = &testkit.MockActor{Name: id}
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": id}) //nolint:errcheck // test helper
	})

	mux.HandleFunc("POST /perform", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     string `json:"id"`
			Prompt string `json:"prompt"`
		}
		json.NewDecoder(r.Body).Decode(&req) //nolint:errcheck // test helper
		mu.Lock()
		actor, ok := actors[req.ID]
		mu.Unlock()
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		resp, err := actor.Perform(context.Background(), req.Prompt)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"response": resp}) //nolint:errcheck // test helper
	})

	mux.HandleFunc("GET /ready/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		mu.Lock()
		actor, ok := actors[id]
		mu.Unlock()
		ready := ok && actor.Ready()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"ready": ready}) //nolint:errcheck // test helper
	})

	mux.HandleFunc("POST /kill/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		mu.Lock()
		actor, ok := actors[id]
		mu.Unlock()
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		actor.Kill(context.Background()) //nolint:errcheck // test helper
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"ok": true}) //nolint:errcheck // test helper
	})

	return mux
}
