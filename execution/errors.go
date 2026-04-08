package execution

import "errors"

// Sentinel errors for LLM provider failures. Consumers use errors.Is()
// to match and provide actionable hints.
var (
	// ErrProviderNotSet is returned when the provider env var is empty.
	ErrProviderNotSet = errors.New("provider not configured")

	// ErrCredentialsMissing is returned when required API keys or ADC are not available.
	ErrCredentialsMissing = errors.New("credentials missing")

	// ErrModelNotFound is returned on HTTP 404 — model ID doesn't exist on the provider.
	ErrModelNotFound = errors.New("model not found")

	// ErrQuotaExceeded is returned on HTTP 429 — rate limit or quota exhausted.
	ErrQuotaExceeded = errors.New("quota exceeded")

	// ErrAuthFailed is returned on HTTP 401/403 — credentials invalid or insufficient permissions.
	ErrAuthFailed = errors.New("authentication failed")

	// ErrStreamingNotSupported is returned when streaming is called on a non-streaming provider.
	ErrStreamingNotSupported = errors.New("streaming not supported")

	// ErrModelRequired is returned when the model parameter is empty.
	ErrModelRequired = errors.New("model is required")

	// ErrProviderUnknown is returned for an unrecognized provider name.
	ErrProviderUnknown = errors.New("unknown provider")

	// ErrNoChoices is returned when the LLM response contains no choices.
	ErrNoChoices = errors.New("no choices returned")

	// ErrContentNotText is returned when the LLM response content is not a string.
	ErrContentNotText = errors.New("content is not a string")
)
