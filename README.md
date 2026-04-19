<p align="center">
  <img src="assets/troupe.png" alt="Troupe" width="200"/>
</p>

# Troupe

**AI Agent Platform** — infrastructure + harness components for multi-agent orchestration without vendor lock-in.

Troupe has two faces:
1. **Server** — authoritative ECS World managing agent lifecycle, admission, and communication via A2A
2. **Library** — exportable harness components (LLM drivers, billing, resilience, model selection, scoring)

## Install

```bash
go get github.com/dpopsuev/troupe@latest
```

## Core API

```go
// Broker casts and hires actors for Directors.
type Broker interface {
    Pick(ctx context.Context, prefs Preferences) ([]ActorConfig, error)
    Spawn(ctx context.Context, config ActorConfig) (Actor, error)
    Discover(role string) []AgentCard
}

// Actor is what the Broker gives back — an agent ready to perform.
type Actor interface {
    Perform(ctx context.Context, prompt string) (string, error)
    Ready() bool
    Kill(ctx context.Context) error
}

// Director is the consumer contract for orchestration strategies.
type Director interface {
    Direct(ctx context.Context, broker Broker) (<-chan Event, error)
}
```

## Architecture

```
Djinn (operator frontend)
    |
    v
Origami (orchestration + tools)
    |
    v
 Troupe (infrastructure)
    |
    ├── Server: World, Broker, Admission, Signal, Transport
    └── Library: execution/, billing/, resilience/, arsenal/, referee/
```

## Server Packages

| Package | Purpose |
|---------|---------|
| `broker/` | Broker implementation, Lobby admission, multi-driver adapter |
| `signal/` | Three-bus architecture (ControlLog, WorkLog, StatusLog), Andon health |
| `world/` | ECS entity-component store |
| `collective/` | Multi-agent primitives (Race, RoundRobin, Scatter, Dialectic, Arbiter) |

## Library Packages

| Package | Purpose |
|---------|---------|
| `execution/` | LLM provider abstraction (Anthropic, OpenAI, Gemini, Vertex, OpenRouter) |
| `billing/` | Token tracking and budget enforcement |
| `resilience/` | CircuitBreaker, Retry, RateLimiter, Timeout |
| `arsenal/` | Trait-scored model catalog and selection |
| `referee/` | YAML-defined weighted scoring engine |
| `visual/` | Cosmetic identity (Color, Palette, Element) |
| `testkit/` | MockBroker, MockActor, test directors |

## Consumers

| Consumer | Role |
|----------|------|
| [Origami](https://github.com/dpopsuev/origami) | Circuit orchestration, instruments, Virtuoso agent harness |
| [Djinn](https://github.com/dpopsuev/djinn) | Operator frontend, REPL, HITL, context engineering |

## License

See [LICENSE](LICENSE).
