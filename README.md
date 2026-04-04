<p align="center">
  <img src="assets/troupe.png" alt="Troupe" width="200"/>
</p>

# Troupe

**AI Agent Broker** — the contract library that makes multi-agent orchestration possible without vendor lock-in.

Troupe does not orchestrate. Directors bring orchestration strategies. Troupe provides the actors.

## Install

```bash
go get github.com/dpopsuev/troupe@latest
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"

    "github.com/dpopsuev/troupe"
)

func main() {
    ctx := context.Background()
    broker := troupe.NewBroker("")

    // Spawn an actor
    actor, _ := broker.Spawn(ctx, troupe.ActorConfig{
        Model: "sonnet",
        Role:  "analyst",
    })

    // Perform work
    response, _ := actor.Perform(ctx, "Analyze this codebase")
    fmt.Println(response)

    // Cleanup
    actor.Kill(ctx)
}
```

## Core API

3 interfaces, 6 methods:

```go
// Broker casts and hires actors for Directors.
type Broker interface {
    Pick(ctx context.Context, prefs Preferences) ([]ActorConfig, error)
    Spawn(ctx context.Context, config ActorConfig) (Actor, error)
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
Consumer (Origami, Djinn, custom)
    |
    v
 Director  ──>  Broker  ──>  Actor
                   |
                   v
                Driver (ACP, HTTP, custom)
```

- **Broker** — casts actors, manages lifecycle
- **Actor** — performs work, reports readiness
- **Director** — orchestrates actors via Broker (consumer-defined)
- **Driver** — provisions agents (ACP subprocess, HTTP API, custom)

## Packages

| Package | Purpose |
|---------|---------|
| `troupe` | Core interfaces: Broker, Actor, Director, Driver, Event |
| `identity/` | Agent identity: Color, Archetype, TraitVector, Mission |
| `arsenal/` | Model catalog and selection |
| `signal/` | Event bus for agent lifecycle signals |
| `collective/` | N agents behind one Actor interface (strategies: Dialectic, Race, RoundRobin, Scatter) |
| `world/` | ECS (Entity Component System) — Broker's internal state |
| `worldview/` | Read-only World projections |
| `resilience/` | CircuitBreaker, Retry, RateLimiter |
| `billing/` | Token tracking and budget enforcement |
| `testkit/` | MockBroker, MockActor, LinearDirector, FanOutDirector |

## Consumers

| Consumer | Role | Import Path |
|----------|------|-------------|
| [Origami](https://github.com/dpopsuev/origami) | Circuit Director — YAML DAG orchestration | `agentport/` adapter |
| Djinn | Local Director — task orchestration | `jerichoport/` adapter |
| Olympiad | Agent Mesh — distributed broker service | shared World |

## Driver

Troupe ships with ACP (Agent Communication Protocol) as the default driver — subprocess agents over stdio JSON-RPC. Custom drivers implement two methods:

```go
type Driver interface {
    Start(ctx context.Context, id world.EntityID, config ActorConfig) error
    Stop(ctx context.Context, id world.EntityID) error
}

broker := troupe.NewBroker("", troupe.WithDriver(myDriver))
```

## Remote Broker

```go
// Local: in-process ACP
broker := troupe.NewBroker("")

// Remote: HTTP proxy to a running Troupe service
broker := troupe.NewBroker("https://cluster:8080")
```

## License

See [LICENSE](LICENSE).
