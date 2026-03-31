# Claude Code Instructions for Jericho Development

## What is Jericho

Jericho is the agent platform — ECS framework for managing autonomous AI agents at scale. The Bugle Protocol (jericho/bugle/) is the wire format for distributing work to agents.

- Repo: github.com/dpopsuev/jericho (renamed from bugle on 2026-03-31)
- Scribe scope: jericho (legacy artifacts use BGL- prefix)
- Campaigns: BGL-CMP-6 (v0.1.0, complete), BGL-CMP-7 (v0.2.0 Cloud Native, complete)

## Ecosystem Dependency Rules (JRC-SPC-2)

**CRITICAL: Jericho is the bottom of the dependency stack.**

- Jericho NEVER imports origami/ or djinn/ or hegemony/
- Jericho defines interfaces (Responder, Server), consumers implement them
- Consumer-to-consumer communication goes through Bugle Protocol, not Go imports

Dependency direction: `Origami -> Jericho <- Djinn`

## Package Map

```
bugle/        — Bugle Protocol types + AuthServer middleware (LEAF, zero deps)
orchestrate/  — Protocol client loop (RunWorker, MCP helpers)
resilience/   — Circuit breaker, rate limiter, retry (pure algorithms)
acp/          — Agent Context Protocol launcher (safe cmd.Env)
pool/         — Agent process lifecycle (Fork/Kill/Wait, restart, graceful term)
agent/        — Staff, Solo, Agent interface
collective/   — Multi-agent collectives (Dialectic, Arbiter, RoundRobin, Race, Scatter, DialecticPair)
transport/    — A2A messaging (LocalTransport, role-based routing, AgentLookup)
signal/       — Event bus (Bus, DurableBus, WorkerStatus typed enum)
world/        — ECS entity-component store (Alive/Ready probes, ZonedWorld)
symbol/       — Visual identity (Color, Element, Persona, 12-shade palette, Registry)
trait/        — Behavioral traits (8-trait vocabulary, FromVector bridge to Arsenal)
intent/       — Mission purpose (ECS component)
persona/      — Default persona templates (Herald, Seeker, etc.)
arsenal/      — Embedded model catalog (trait-scored selection, Select/Pick, snapshot pinning)
workload/     — Declarative YAML workload types (WorkerPool, DebateTeam, TaskRunner, Controller)
billing/      — Token/cost tracking
worldview/    — Observable agent state (Snapshot, Subscribe)
testkit/      — Test fixtures (QuickWorld, handlers, assertions)
```

## Naming Conventions

- **Bugle Protocol verbs**: pull (get work), push (return results). NOT step/submit.
- **Health signals**: andon (IEC 60073 stack light). NOT horn.
- **Work item field**: `item`. NOT `step`.
- **Tool name**: `bugle`. NOT `circuit`.
- **Project name**: Jericho. NOT Bugle (Bugle = the protocol only).

## Go Conventions

- Go 1.25+
- golangci-lint enforced via pre-commit hook
- American English spelling (canceled, not cancelled)
- Sentinel errors with descriptive names
- slog for structured logging with constant key names
