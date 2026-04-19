# Claude Code Instructions for Troupe Development

## What is Troupe

Troupe is a dual-face AI agent platform:
1. **Server**: Authoritative ECS World for multi-agent orchestration — lifecycle, A2A, admission, buses, scheduling
2. **Library**: Exportable harness components for agent builders — LLM drivers, billing, resilience, model selection, scoring, predicates

Core interfaces: Caster (Pick+Spawn), Actor, Director, Admin. Consumer flow: `troupe.Pick → troupe.Spawn → troupe.Perform`.

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
- Troupe defines interfaces (Caster, Actor, Director, Driver, Meter, Admin), consumers implement them
- Consumer-to-consumer communication goes through A2A protocol, not Go imports

## Public API

```go
// Caster is the narrow factory interface. Troupe facade satisfies it.
// Directors and Collectives take Caster, not the full facade.
type Caster interface {
    Pick(ctx, Preferences) ([]ActorConfig, error)
    Spawn(ctx, ActorConfig) (Actor, error)
}

// Broker = Caster + Discover. Internal engine, not consumer-facing.
type Broker interface {
    Caster
    Discover(role string) []AgentCard
}

// Troupe facade: Admit, Kick, Ban, Pick, Spawn, Discover, Perform
// Satisfies Caster — can be passed directly to Directors/Collectives.
```

## Architecture: Server + Library

### Server packages (authoritative multi-agent world)

```
Root package   — Caster, Broker, Actor, Director, Driver, Meter, Gate, Pick, Threshold, Admission, Admin, AgentCard
broker/        — DefaultBroker, Lobby (admission), DefaultAdmin, multi-driver adapter, hooked actors
                 broker.New() = bare, broker.Default() = batteries-included (Arsenal wired)
                 AdmissionHandler() = HTTP endpoint for external agent registration
signal/        — Three-bus architecture (ControlLog, WorkLog, StatusLog), Andon health, EventStore
world/         — ECS entity-component store (Alive/Ready/Budget/Annotation, ComponentType, hierarchy edges)
collective/    — Multi-agent primitives (Race, RoundRobin, Scatter, Scale, Dialectic, Arbiter, Fallback)
                 Add/Remove for per-agent join/leave. SpawnCollective takes Caster.
client/        — Go SDK for external agents (Register via POST /admission)
```

### Library packages (all wired into Broker via With* options)

```
providers/     — LLM provider abstraction (any-llm-go: Anthropic, OpenAI, Gemini, Vertex, OpenRouter)
                 Wired via WithProviderResolver()
billing/       — Token/cost tracking (CostBill, BudgetEnforcer) — wired via WithTracker()
referee/       — Event-driven scoring engine (YAML Scorecards) — wired via WithReferee()
arsenal/       — Embedded model catalog (trait-scored selection, TraitVector) — wired via WithArsenal()
                 OpenRouter discoverer for free model catalog enrichment
resilience/    — Circuit breaker, retry, timeout, rate limiter (pure algorithms)
visual/        — Cosmetic identity (Color, Palette, Element, View)
testkit/       — Test fixtures (MockActor, MockBroker, toy agents, BusSet helpers)
```

### Internal packages

```
internal/agent/     — Solo agent implementation (Actor wrapper)
auth/               — Authentication abstraction (Bearer, Identity, Authenticator)
                      Wired into transport via NewA2ATransportWithAuth()
internal/transport/ — A2A messaging types, LocalTransport, HTTPTransport, A2A server
internal/warden/    — Agent process supervision (Fork/Kill/Wait, restart, zombie reaping)
```

## Protocol Decisions

- **A2A is the protocol**: A2A (HTTP JSON-RPC via a2a-go) is the sole wire protocol. Types live in internal/transport/.
- **A2A roles**: Messages use A2A roles (user/agent) directly. RoleUser/RoleAgent constants.
- **Heartbeat vs Andon**: Heartbeat = control plane liveness (transport-level lastSeen). Andon = data plane readiness (StatusLog).
- **Three buses**: ControlLog (routing), WorkLog (task lifecycle), StatusLog (health/observability). All durable.
- **Vertex env vars**: Use Google standard (GOOGLE_CLOUD_LOCATION, GOOGLE_CLOUD_PROJECT).
- **Arsenal source provider mask**: vertex-ai.yaml `provider` field filters models to what the implementation can reach.

## Admin Control Plane

`Admin` interface (admin.go) — privileged operator API, separate from Caster (agent-facing):
- Query: Agents(), Inspect(), Tree(), Annotations()
- Lifecycle: Kill(), Drain(), Undrain()
- Policy: SetBudget(), SetQuota(), Annotate()
- Emergency: Cordon(), Uncordon(), KillAll()
- Streaming: Watch()

Implemented in `broker/admin.go`. CordonGate() returns a Gate for Lobby admission blocking.

## Admission

- **Admit**: register agent into World (gates, entity, transport, events)
- **Kick**: forceful removal (hooks can block via KickHook)
- **Ban**: Kick + deny list (hooks can block via BanHook)
- **Unban**: remove from deny list
- External agents register via `POST /admission` (broker/admission_endpoint.go)
- Go SDK: `client.New(serverURL).Register(ctx, role, callbackURL)`

## Naming Conventions

- **Core interfaces**: Caster (Pick+Spawn), Actor (Perform/Ready/Kill), Director, Admin
- **Internal**: Broker (Caster+Discover), Driver, Meter
- **Predicates**: Gate (allow/deny), Pick[T] (selection), Threshold (numeric condition)
- **Health signals**: Andon (IEC 60073 stack light). NOT horn.
- **Events**: EventKind (Started, Completed, Failed, Transition, Done)
- **Identity**: AgentCard (public interface), ActorConfig (input/job spec), Actor (running instance)
- **Visual**: Color, Palette, Element — cosmetic only, in visual/ package
- **Admission**: Admit (enter), Kick (forceful removal), Ban (Kick + deny list)
- **Project name**: Troupe. NOT Jericho or Bugle.

## Go Conventions

- Go 1.25+
- golangci-lint enforced via pre-commit hook
- American English spelling (canceled, not cancelled)
- Sentinel errors with descriptive names
- slog for structured logging with constant key names
- broker.New() = bare, broker.Default() = batteries-included
- integrate-early: no package without a production caller
