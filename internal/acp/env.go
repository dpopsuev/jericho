package acp

import (
	"os"
	"strings"
)

// safeEnvKeys is the allowlist of env vars inherited by child processes.
var safeEnvKeys = []string{
	"PATH",
	"HOME",
	"USER",
	"TERM",
	"LANG",
	"SHELL",
	// Provider API keys — required for agent authentication.
	"ANTHROPIC_API_KEY",
	"OPENAI_API_KEY",
	"GOOGLE_API_KEY",
	"GEMINI_API_KEY",
	"CODEX_API_KEY",
	// Claude Vertex AI routing.
	"CLAUDE_CODE_USE_VERTEX",
}

// safeEnv builds a minimal environment for child processes.
// Only allowlisted keys + explicit extras are included.
func safeEnv(extra map[string]string) []string {
	env := make([]string, 0, len(safeEnvKeys)+len(extra))

	for _, key := range safeEnvKeys {
		if val, ok := os.LookupEnv(key); ok {
			env = append(env, key+"="+val)
		}
	}

	// Also inherit any ANTHROPIC_VERTEX_*, CLOUD_ML_* vars.
	for _, e := range os.Environ() {
		parts := strings.SplitN(e, "=", 2) //nolint:mnd // key=value split
		if len(parts) != 2 {               //nolint:mnd // key=value split
			continue
		}
		key := parts[0]
		if strings.HasPrefix(key, "ANTHROPIC_VERTEX_") || strings.HasPrefix(key, "CLOUD_ML_") {
			env = append(env, e)
		}
	}

	for k, v := range extra {
		env = append(env, k+"="+v)
	}

	return env
}
