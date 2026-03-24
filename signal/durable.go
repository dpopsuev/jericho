package signal

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// DurableBus wraps a MemBus with persistent tee-write to a JSON-Lines
// file. On crash recovery, Replay() re-reads the file and populates
// the in-memory bus with historical signals.
type DurableBus struct {
	inner *MemBus
	mu    sync.Mutex
	path  string
	file  *os.File
	enc   *json.Encoder
}

// NewDurableBus creates a durable bus that persists signals to the
// given file path. The file is created if it does not exist, or
// opened for append if it does.
func NewDurableBus(path string) (*DurableBus, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open signal log %s: %w", path, err)
	}
	return &DurableBus{
		inner: NewMemBus(),
		path:  path,
		file:  f,
		enc:   json.NewEncoder(f),
	}, nil
}

// Emit appends a signal to both the in-memory bus and the persistent log.
// Returns the index of the signal in the in-memory bus.
func (d *DurableBus) Emit(s *Signal) int {
	idx := d.inner.Emit(s)

	d.mu.Lock()
	defer d.mu.Unlock()

	if d.enc != nil {
		signals := d.inner.Since(idx)
		if len(signals) > 0 {
			_ = d.enc.Encode(signals[0])
		}
	}
	return idx
}

// Since returns a copy of signals from index onward.
func (d *DurableBus) Since(idx int) []Signal {
	return d.inner.Since(idx)
}

// Len returns the number of signals in the bus.
func (d *DurableBus) Len() int {
	return d.inner.Len()
}

// OnEmit registers a callback that fires on every Emit.
func (d *DurableBus) OnEmit(fn func(Signal)) {
	d.inner.OnEmit(fn)
}

// Replay reads persisted signals from the file and populates the
// in-memory bus. Call this once on startup before any new Emit calls
// to restore state after a crash.
func (d *DurableBus) Replay() (int, error) {
	f, err := os.Open(d.path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("replay signal log: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	count := 0
	for scanner.Scan() {
		var sig Signal
		if err := json.Unmarshal(scanner.Bytes(), &sig); err != nil {
			continue
		}
		// Re-emit with the original timestamp preserved.
		d.inner.Emit(&sig)
		count++
	}
	return count, scanner.Err()
}

// Close flushes and closes the persistent log file.
func (d *DurableBus) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.file != nil {
		err := d.file.Close()
		d.file = nil
		d.enc = nil
		return err
	}
	return nil
}

// Path returns the file path of the persistent log.
func (d *DurableBus) Path() string {
	return d.path
}
