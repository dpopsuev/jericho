package testkit

import "errors"

// Sentinel errors for testkit Directors.
var (
	ErrNoActors = errors.New("director: no actors available")
)
