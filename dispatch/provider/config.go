package provider

import (
	"errors"
	"fmt"
	"os"
	"time"

	bd "github.com/dpopsuev/jericho/dispatch"

	"gopkg.in/yaml.v3"
)

// Sentinel errors for config operations.
var (
	ErrNoProviders      = errors.New("dispatch/config: no providers defined")
	ErrMissingName      = errors.New("dispatch/config: provider missing name")
	ErrMissingType      = errors.New("dispatch/config: provider missing type")
	ErrUnknownType      = errors.New("dispatch/config: unknown provider type")
	ErrHTTPBaseRequired = errors.New("http provider requires config.base_url")
	ErrCLICmdRequired   = errors.New("cli provider requires config.command")
)

// Config is the YAML-loadable specification for a dispatch topology:
// named providers with their type and configuration, plus fallback chains.
type Config struct {
	Providers []Def               `yaml:"providers"`
	Fallbacks map[string][]string `yaml:"fallbacks,omitempty"`
}

// Def describes a single named provider.
// Type selects the dispatcher factory; Config carries type-specific parameters.
type Def struct {
	Name   string         `yaml:"name"`
	Type   string         `yaml:"type"`
	Config map[string]any `yaml:"config,omitempty"`
}

// LoadConfig parses a YAML file into a Config.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("dispatch/config: read %s: %w", path, err)
	}
	return ParseConfig(data)
}

// ParseConfig parses raw YAML bytes into a Config.
func ParseConfig(data []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("dispatch/config: parse YAML: %w", err)
	}
	if len(cfg.Providers) == 0 {
		return nil, ErrNoProviders
	}
	for i, p := range cfg.Providers {
		if p.Name == "" {
			return nil, fmt.Errorf("%w: provider[%d]", ErrMissingName, i)
		}
		if p.Type == "" {
			return nil, fmt.Errorf("%w: provider %q", ErrMissingType, p.Name)
		}
	}
	return &cfg, nil
}

// DispatcherFactory creates a bd.Dispatcher from a Def's config map.
type DispatcherFactory func(config map[string]any) (bd.Dispatcher, error)

// BuildRouter constructs a Router from a Config.
//
// Built-in types ("http", "cli", "stdin", "static") are resolved automatically.
// For types that require runtime wiring (e.g. "mux", "file"), register a factory
// via the extraFactories parameter or replace the route entry after construction.
//
// The first provider in the list becomes the default dispatcher.
func BuildRouter(cfg *Config, extraFactories map[string]DispatcherFactory) (*Router, error) {
	factories := builtinFactories()
	for k, v := range extraFactories {
		factories[k] = v
	}

	routes := make(map[string]bd.Dispatcher, len(cfg.Providers))
	var defaultDisp bd.Dispatcher

	for i, pdef := range cfg.Providers {
		factory, ok := factories[pdef.Type]
		if !ok {
			return nil, fmt.Errorf("%w: provider %q type %q", ErrUnknownType, pdef.Name, pdef.Type)
		}
		d, err := factory(pdef.Config)
		if err != nil {
			return nil, fmt.Errorf("dispatch/config: provider %q: %w", pdef.Name, err)
		}
		routes[pdef.Name] = d
		if i == 0 {
			defaultDisp = d
		}
	}

	router := NewRouter(defaultDisp, routes, WithFallbacks(cfg.Fallbacks))
	return router, nil
}

func builtinFactories() map[string]DispatcherFactory {
	return map[string]DispatcherFactory{
		"http":   httpFactory,
		"cli":    cliFactory,
		"stdin":  stdinFactory,
		"static": staticFactory,
	}
}

func staticFactory(config map[string]any) (bd.Dispatcher, error) {
	dir, _ := config["dir"].(string)
	return bd.NewStaticDispatcher(dir), nil
}

func httpFactory(config map[string]any) (bd.Dispatcher, error) {
	baseURL, _ := config["base_url"].(string)
	if baseURL == "" {
		return nil, ErrHTTPBaseRequired
	}

	var opts []bd.HTTPOption
	if model, ok := config["model"].(string); ok && model != "" {
		opts = append(opts, bd.WithModel(model))
	}
	if keyEnv, ok := config["api_key_env"].(string); ok && keyEnv != "" {
		opts = append(opts, bd.WithAPIKeyEnv(keyEnv))
	}

	return bd.NewHTTPDispatcher(baseURL, opts...)
}

func cliFactory(config map[string]any) (bd.Dispatcher, error) {
	command, _ := config["command"].(string)
	if command == "" {
		return nil, ErrCLICmdRequired
	}

	var opts []bd.CLIOption
	if args, ok := config["args"].([]any); ok {
		strArgs := make([]string, 0, len(args))
		for _, a := range args {
			strArgs = append(strArgs, fmt.Sprintf("%v", a))
		}
		opts = append(opts, bd.WithCLIArgs(strArgs...))
	}
	if timeoutStr, ok := config["timeout"].(string); ok {
		dur, err := time.ParseDuration(timeoutStr)
		if err != nil {
			return nil, fmt.Errorf("cli provider: invalid timeout %q: %w", timeoutStr, err)
		}
		opts = append(opts, bd.WithCLITimeout(dur))
	}

	return bd.NewCLIDispatcher(command, opts...)
}

func stdinFactory(_ map[string]any) (bd.Dispatcher, error) {
	return bd.NewStdinDispatcher(), nil
}
