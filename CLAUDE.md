# Claude Code Instructions for Tangle Development

## What is Tangle

Tangle is the Federated Agent Runtime (FAR) — infrastructure for agent platforms. Tako is a Factory, Tangle is the Utilities (electricity, water, railway, security, inspectors). The factory doesn't build its own power plant, it plugs in. Tangle serves any factory, doesn't know what's manufactured inside. Tangle can run standalone.

- Repo: github.com/dpopsuev/tangle (renamed from troupe)
- Scribe scope: `tangled`
- Campaign: CMP-44 (Tangle Reform)

## ISA-95 Alignment (5-Level Automation Pyramid)

- L4 Coordination (Tako): Director → Collectives hierarchy. HITL aggregation.
- L3 Services (Tako): Kanban, Andon, Discourse, Sleep. Stigmergic coordination.
- L2 Routing (Tangle): Switchboard (TangleD). Wire routing. Star topology.
- L1 Connectivity (Tangle): Tangle client/server. Embed/Connect. VersionGate.
- L0 Execution (Tangle): AXI. Mirage enclosure. Instrument execution. LLM calls.

Tangle owns L0-L2. Tako owns L3-L4. Clean boundary.

## 6 Interface Families (TNG-DOC-1)

- **AAI** — Agent Auth Interface (DEFINES trust: Identity, Capability, Audit)
- **ARI** — Agent Runtime Interface (exist: Probe, Lifecycle, Caster)
- **ANI** — Agent Network Interface (connect: Admission, Gate, Registry)
- **ASI** — Agent State Interface (persist: Stateful seam, StateStore, Config, Meter)
- **AOI** — Agent Observability Interface (watch: Health events, Admin, OTel shape)
- **AXI** — Agent Execution Interface (execute: Executor, Sandbox, ResourceLimit, Policy)

AAI defines contracts. Other families enforce at their boundary. Defense-in-depth.

## Public API

```go
type Caster interface {
    Pick(ctx, Preferences) ([]ActorConfig, error)
    Spawn(ctx, ActorConfig) (Actor, error)
}

type Broker interface {
    Caster
    Discover(role string) []AgentCard
}
```

## Architecture: Server + Library

### Server packages
```
Root package   — Caster, Broker, Actor, Director, Driver, Meter, Gate, Pick, Threshold, Admission, Admin, AgentCard
broker/        — DefaultBroker, Lobby (admission), DefaultAdmin, multi-driver adapter
signal/        — Three-bus architecture (ControlLog, WorkLog, StatusLog), Andon health
world/         — ECS entity-component store
collective/    — Multi-agent primitives (Race, RoundRobin, Scatter, Scale, Dialectic, Arbiter, Fallback)
client/        — Go SDK for external agents
```

### Library packages
```
providers/     — LLM provider abstraction (any-llm-go)
billing/       — Token/cost tracking
referee/       — Event-driven scoring engine (YAML Scorecards)
arsenal/       — Embedded model catalog (trait-scored selection)
resilience/    — Circuit breaker, retry, timeout, rate limiter
visual/        — Cosmetic identity (Color, Palette, Element, View)
testkit/       — Test fixtures
```

## Key Principles

- Tangle NEVER imports tako/ or mirage/ — bottom of the dependency stack
- Tangle defines interfaces, consumers implement them
- Tangle emits facts (HealthEvents, ResourceBreachEvents), Tako interprets (Andon, OAE)
- OTel: Tangle owns shape (context propagation, metric registry), Tako owns content (span names)
- PDP-PEP: Tangle is PDP (Policy Decision Point, decides), Tako is PEP (enforces)

## Naming Conventions

- **Project name**: Tangle (was Troupe). NOT Jericho or Bugle.
- **Core interfaces**: Caster (Pick+Spawn), Actor, Director, Admin
- **Internal**: Broker (Caster+Discover), Driver, Meter
- **Health**: Andon (IEC 60073 stack light)
- **Identity**: AgentCard (public), ActorConfig (input/job), Actor (running)
- **Admission**: Admit (enter), Kick (remove), Ban (Kick + deny list)

## Go Conventions

- Go 1.25+
- golangci-lint enforced via pre-commit hook
- American English spelling
- Sentinel errors, slog structured logging
- broker.New() = bare, broker.Default() = batteries-included
