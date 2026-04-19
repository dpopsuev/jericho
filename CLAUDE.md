# Claude Code Instructions for Troupe Development

## What is Troupe

Troupe is a dual-face AI agent platform:
1. **Server**: Authoritative ECS World for multi-agent orchestration — lifecycle, A2A, admission, buses, scheduling
2. **Library**: Exportable harness components for agent builders — LLM drivers, billing, resilience, model selection, scoring, predicates

Three core interfaces (Broker, Actor, Director) compose agents, strategies, and drivers into collectives.

- Repo: github.com/dpopsuev/troupe
- Scribe scope: troupe

## Platform Boundaries

Troupe is part of a three-project platform. Each project owns specific boundaries:

| Boundary | Owner | What |
|---|---|---|
| World ↔ Agent | **Troupe** | A2A, Admission, lifecycle, ECS, buses, resilience, billing, scoring |
| Agent ↔ Tools | **Origami** | Workbench instruments, normalized tool I/O, strict mode |
| Agent Orchestration | **Origami** | Circuits, graph-walking, Virtuoso harness |
| Human ↔ Agent | **Djinn** | Operator frontend, REPL, HITL, Cortex, context engineering |

Dependency direction: `Djinn -> Origami -> Troupe`

## Ecosystem Dependency Rules

**CRITICAL: Troupe is the bottom of the dependency stack.**

- Troupe NEVER imports origami/ or djinn/ or hegemony/
- Troupe defines interfaces (Actor, Broker, Director, Driver, Meter), consumers implement them
- Consumer-to-consumer communication goes through A2A protocol, not Go imports

## Architecture: Server + Library

### Server packages (authoritative multi-agent world)

```
Root package   — Broker, Actor, Director, Driver, Meter, Gate, Pick, Threshold, Admission, AgentCard interfaces
broker/        — Broker implementation, Lobby (admission), multi-driver adapter, hooked actors
signal/        — Three-bus architecture (ControlLog, WorkLog, StatusLog), Andon health, EventStore
world/         — ECS entity-component store (Alive/Ready, ComponentType, hierarchy edges)
collective/    — Multi-agent primitives (Race, RoundRobin, Scatter, Scale, Dialectic, Arbiter, Fallback)
```

### Library packages (exportable harness components)

```
providers/     — LLM provider abstraction (any-llm-go: Anthropic, OpenAI, Gemini, Vertex, OpenRouter)
billing/       — Token/cost tracking (CostBill, period management, BudgetEnforcer) — dual-homed: server + agent
referee/       — Event-driven scoring engine (YAML Scorecards, weighted rules) — dual-homed: server + agent
arsenal/       — Embedded model catalog (trait-scored selection, TraitVector, snapshot pinning)
resilience/    — Circuit breaker, retry, timeout, rate limiter (pure algorithms)
visual/        — Cosmetic identity (Color, Palette, Element, View)
testkit/       — Test fixtures (MockActor, MockBroker, LinearDirector, FanOutDirector, BusSet helpers)
```

### Internal packages

```
internal/acp/       — Agent Context Protocol launcher (JSON-RPC over stdio, process spawning)
internal/agent/     — Solo agent implementation (Actor wrapper)
auth/               — Authentication abstraction (Bearer, Identity, Authenticator)
internal/transport/ — A2A messaging (LocalTransport, HTTPTransport, A2A server/proxy) — core types to be promoted
internal/warden/    — Agent process supervision (Fork/Kill/Wait, restart, zombie reaping)
```

## Protocol Decisions

- **Uniform A2A**: Server uses A2A (HTTP JSON-RPC) for ALL agent communication — local or remote. No bifurcated transport.
- **ACP**: Remains as process launcher. Agents register back via A2A through Admission/Lobby.
- **Heartbeat vs Andon**: Heartbeat = control plane liveness (transport-level lastSeen). Andon = data plane readiness (StatusLog: Nominal/Degraded/Failure/Blocked/Dead).
- **Three buses**: ControlLog (routing), WorkLog (task lifecycle), StatusLog (health/observability). All durable.

## Naming Conventions

- **Core interfaces**: Actor (Perform/Ready/Kill), Broker, Director, Driver, Meter
- **Predicates**: Gate (allow/deny), Pick[T] (selection), Threshold (numeric condition)
- **Health signals**: Andon (IEC 60073 stack light). NOT horn.
- **Events**: EventKind (Started, Completed, Failed, Transition, Done)
- **Identity**: AgentCard (public interface), ActorConfig (input/job spec), Actor (running instance)
- **Visual**: Color, Palette, Element — cosmetic only, in visual/ package
- **Project name**: Troupe. NOT Jericho or Bugle.

## Go Conventions

- Go 1.25+
- golangci-lint enforced via pre-commit hook
- American English spelling (canceled, not cancelled)
- Sentinel errors with descriptive names
- slog for structured logging with constant key names
