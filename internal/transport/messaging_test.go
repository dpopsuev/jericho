package transport

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestAsk_RespondsCorrectly(t *testing.T) {
	tr := NewLocalTransport()
	defer tr.Close()

	_ = tr.Register(AgentID("echo"), func(_ context.Context, msg Message) (Message, error) {
		return Message{
			From:    "echo",
			To:      msg.From,
			Content: "echo: " + msg.Content,
		}, nil
	})

	resp, err := tr.Ask(context.Background(), "echo", Message{
		From:    "caller",
		To:      "echo",
		Content: "hello",
	})
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if resp.Content != "echo: hello" {
		t.Errorf("Content = %q, want %q", resp.Content, "echo: hello")
	}
	if resp.From != "echo" {
		t.Errorf("From = %q, want %q", resp.From, "echo")
	}
}

func TestAsk_Timeout(t *testing.T) {
	tr := NewLocalTransport()
	defer tr.Close()

	_ = tr.Register(AgentID("slow"), func(_ context.Context, _ Message) (Message, error) {
		time.Sleep(200 * time.Millisecond)
		return Message{Content: "late"}, nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := tr.Ask(ctx, AgentID("slow"), Message{From: "caller"})
	if err == nil {
		t.Fatal("expected error from timeout")
	}
	if ctx.Err() == nil {
		t.Error("expected context to be done")
	}
}

func TestAsk_NotRegistered(t *testing.T) {
	tr := NewLocalTransport()
	defer tr.Close()

	_, err := tr.Ask(context.Background(), "ghost", Message{From: "caller"})
	if err == nil {
		t.Fatal("expected error for unregistered agent")
	}
}

func TestSendToRole_RoundRobin(t *testing.T) {
	tr := NewLocalTransport()
	defer tr.Close()

	received := make(map[string]int)
	var mu sync.Mutex

	for i := range 3 {
		aid := AgentID(fmt.Sprintf("exec-%d", i))
		_ = tr.Register(aid, func(_ context.Context, _ Message) (Message, error) {
			return Message{Content: "done"}, nil
		})
		tr.Roles().Register(string(aid), "executor")
	}

	// Send 6 messages — expect 2 per agent (round-robin).
	for range 6 {
		task, err := tr.SendToRole(context.Background(), "executor", Message{From: "boss"})
		if err != nil {
			t.Fatalf("SendToRole: %v", err)
		}

		ch, err := tr.Subscribe(context.Background(), task.ID)
		if err != nil {
			t.Fatalf("Subscribe: %v", err)
		}
		for ev := range ch {
			if ev.State == TaskCompleted && ev.Data != nil {
				mu.Lock()
				// Track which task completed — we can infer target from task ID
				// but a simpler approach: check total distribution at the end.
				received[task.ID]++
				mu.Unlock()
			}
		}
	}

	// We should have 6 distinct completed tasks.
	if len(received) != 6 {
		t.Errorf("completed tasks = %d, want 6", len(received))
	}

	// Verify round-robin by checking the internal counter advanced.
	tr.mu.RLock()
	counter := tr.roleCounter["executor"]
	tr.mu.RUnlock()
	if counter != 6 {
		t.Errorf("roleCounter[executor] = %d, want 6", counter)
	}
}

func TestSendToRole_RoundRobin_Distribution(t *testing.T) {
	tr := NewLocalTransport()
	defer tr.Close()

	hits := make(map[string]int)
	var mu sync.Mutex

	for i := range 3 {
		aid := AgentID(fmt.Sprintf("w-%d", i))
		id := aid // capture
		_ = tr.Register(id, func(_ context.Context, _ Message) (Message, error) {
			mu.Lock()
			hits[string(id)]++
			mu.Unlock()
			return Message{Content: "ok"}, nil
		})
		tr.Roles().Register(string(id), "executor")
	}

	// Send 6 messages — expect each agent gets exactly 2.
	for range 6 {
		task, err := tr.SendToRole(context.Background(), "executor", Message{From: "boss"})
		if err != nil {
			t.Fatalf("SendToRole: %v", err)
		}
		ch, _ := tr.Subscribe(context.Background(), task.ID)
		for range ch {
		}
	}

	// Wait briefly for handlers to run.
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	for i := range 3 {
		key := fmt.Sprintf("w-%d", i)
		if hits[key] != 2 {
			t.Errorf("hits[%s] = %d, want 2", key, hits[key])
		}
	}
}

func TestAskRole_BlocksForResponse(t *testing.T) {
	tr := NewLocalTransport()
	defer tr.Close()

	_ = tr.Register(AgentID("worker-0"), func(_ context.Context, msg Message) (Message, error) {
		return Message{
			From:    "worker-0",
			Content: "reply: " + msg.Content,
		}, nil
	})
	tr.Roles().Register("worker-0", "processor")

	resp, err := tr.AskRole(context.Background(), "processor", Message{
		From:    "caller",
		Content: "process this",
	})
	if err != nil {
		t.Fatalf("AskRole: %v", err)
	}
	if resp.Content != "reply: process this" {
		t.Errorf("Content = %q, want %q", resp.Content, "reply: process this")
	}
}

func TestBroadcast_AllReceive(t *testing.T) {
	tr := NewLocalTransport()
	defer tr.Close()

	var received sync.Map

	for i := range 3 {
		aid := AgentID(fmt.Sprintf("agent-%d", i))
		id := aid // capture
		_ = tr.Register(id, func(_ context.Context, _ Message) (Message, error) {
			received.Store(id, true)
			return Message{From: id, Content: "ack"}, nil
		})
		tr.Roles().Register(string(id), "executor")
	}

	tasks, err := tr.Broadcast(context.Background(), "executor", Message{
		From:    "leader",
		Content: "do the thing",
	})
	if err != nil {
		t.Fatalf("Broadcast: %v", err)
	}
	if len(tasks) != 3 {
		t.Fatalf("Broadcast returned %d tasks, want 3", len(tasks))
	}

	// Wait for all tasks to complete.
	for _, task := range tasks {
		ch, err := tr.Subscribe(context.Background(), task.ID)
		if err != nil {
			t.Fatalf("Subscribe: %v", err)
		}
		for ev := range ch {
			if ev.State == TaskCompleted {
				if ev.Data == nil {
					t.Error("completed event should have data")
				}
			}
		}
	}

	// Verify all 3 agents received the message.
	count := 0
	received.Range(func(_, _ any) bool {
		count++
		return true
	})
	if count != 3 {
		t.Errorf("received count = %d, want 3", count)
	}
}

func TestBroadcast_EmptyRole(t *testing.T) {
	tr := NewLocalTransport()
	defer tr.Close()

	_, err := tr.Broadcast(context.Background(), "nonexistent", Message{From: "leader"})
	if err == nil {
		t.Fatal("expected error for empty role")
	}
}
