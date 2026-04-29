package broker_test

import (
	"context"
	"errors"
	"testing"

	"github.com/dpopsuev/tangle"
	"github.com/dpopsuev/tangle/broker"
	"github.com/dpopsuev/tangle/world"
)

// RED: Driver must be a proper public interface, not a type alias.

type testDriver struct {
	started map[world.EntityID]bool
	stopped map[world.EntityID]bool
}

func newTestDriver() *testDriver {
	return &testDriver{
		started: make(map[world.EntityID]bool),
		stopped: make(map[world.EntityID]bool),
	}
}

func (d *testDriver) Start(_ context.Context, id world.EntityID, _ troupe.AgentConfig) error {
	d.started[id] = true
	return nil
}

func (d *testDriver) Stop(_ context.Context, id world.EntityID) error {
	d.stopped[id] = true
	return nil
}

func TestDriver_Interface(t *testing.T) {
	// A custom Driver can be passed to NewBroker via WithDriver
	driver := newTestDriver()
	b := broker.New("", broker.WithDriver(driver))

	actor, err := b.Spawn(context.Background(), troupe.AgentConfig{Role: "test"})
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}

	// Driver.Start should have been called
	if len(driver.started) == 0 {
		t.Error("Driver.Start was not called")
	}

	// Agent should be usable
	if !actor.Ready() {
		t.Error("actor not ready after spawn")
	}
}

func TestDriver_PublicType(t *testing.T) {
	// Driver must be a public interface, not a type alias to internal
	// This test proves consumers can implement Driver without importing internal/
	var _ troupe.Driver = newTestDriver()
}

// --- DriverDescriptor / DriverValidator tests ---

type validatingDriver struct {
	testDriver
	envErr error
	info   troupe.DriverInfo
}

func (d *validatingDriver) ValidateEnvironment(_ context.Context) error { return d.envErr }
func (d *validatingDriver) Describe() troupe.DriverInfo                 { return d.info }

func TestDriverDescriptor_OptionalInterface(t *testing.T) {
	// A bare testDriver should NOT implement DriverDescriptor.
	var d troupe.Driver = newTestDriver()
	if _, ok := d.(troupe.DriverDescriptor); ok {
		t.Error("bare testDriver should not implement DriverDescriptor")
	}

	// A validatingDriver SHOULD implement both optional interfaces.
	vd := &validatingDriver{info: troupe.DriverInfo{Name: "test"}}
	if _, ok := troupe.Driver(vd).(troupe.DriverDescriptor); !ok {
		t.Error("validatingDriver should implement DriverDescriptor")
	}
	if _, ok := troupe.Driver(vd).(troupe.DriverValidator); !ok {
		t.Error("validatingDriver should implement DriverValidator")
	}
}

func TestDriverValidator_RejectsInvalidEnv(t *testing.T) {
	errMissing := errors.New("missing API key")
	driver := &validatingDriver{
		testDriver: *newTestDriver(),
		envErr:     errMissing,
	}
	b := broker.New("", broker.WithDriver(driver))
	_, err := b.Spawn(context.Background(), troupe.AgentConfig{Role: "test"})
	if err == nil {
		t.Fatal("expected env validation error")
	}
	if !errors.Is(err, errMissing) {
		t.Fatalf("expected wrapped env error, got: %v", err)
	}
}

func TestDriverValidator_PassesValidEnv(t *testing.T) {
	driver := &validatingDriver{testDriver: *newTestDriver()}
	b := broker.New("", broker.WithDriver(driver))
	_, err := b.Spawn(context.Background(), troupe.AgentConfig{Role: "test"})
	if err != nil {
		t.Fatalf("Spawn with valid env: %v", err)
	}
}
