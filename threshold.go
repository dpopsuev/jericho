package troupe

// Threshold is a predicate over a numeric condition.
// Returns true when the condition is met (value exceeds limit).
type Threshold func() bool

// IntThreshold returns a Threshold that fires when value() >= limit.
func IntThreshold(value func() int, limit int) Threshold {
	return func() bool {
		return value() >= limit
	}
}

// DurationThreshold returns a Threshold that fires when elapsed() >= limit.
// Both sides use the same unit (caller decides seconds, millis, etc.).
func DurationThreshold(elapsed func() int64, limitNanos int64) Threshold {
	return func() bool {
		return elapsed() >= limitNanos
	}
}

// FloatThreshold returns a Threshold that fires when value() >= limit.
func FloatThreshold(value func() float64, limit float64) Threshold {
	return func() bool {
		return value() >= limit
	}
}
