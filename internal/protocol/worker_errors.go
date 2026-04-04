package protocol

import "errors"

// Sentinel errors.
var (
	ErrStepFailed = errors.New("pull failed")
)
