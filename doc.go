// Package bugle is an agent framework for AI systems.
//
// Bugle handles two protocol concerns:
//   - A2A (Agent-to-Agent) — how agents coordinate with each other
//   - ACP (Agent Client Protocol) — how editors and clients talk to agents
//
// MCP (Model Context Protocol) is a consumer concern — Bugle doesn't
// know about tools. Consumers like Djinn and Origami wire MCP themselves.
//
// Core packages:
//
//   - ECS World (world/ — entity registry, component storage, system queries)
//   - Agent identity (identity/ — AgentIdentity, ModelIdentity, Persona)
//   - Heraldic color system (palette/ — ColorIdentity, Registry, Shade/Colour)
//   - Behavioral profiles (element/ — Element, Approach)
//   - Signal bus (signal/ — Bus, DurableBus)
//   - Process supervision (pool/ — Fork, Kill, Wait, orphan reparenting)
//   - A2A transport (transport/ — LocalTransport, Ask, Broadcast, RoleRegistry)
//   - ACP client (acp/ — JSON-RPC over stdio, 12 providers, shape classifier)
//   - Facade (facade/ — Staff, AgentHandle — API for Humans)
//   - Observable state (worldview/ — View, Snapshot, Minimap)
//   - Billing (billing/ — per-agent cost tracking)
//
// Usage:
//
//	staff := facade.NewStaff(myLauncher)
//	agent, _ := staff.Spawn(ctx, "executor", pool.LaunchConfig{})
//	agent.Listen(func(content string) string { return "done: " + content })
//	response, _ := agent.Ask(ctx, "build auth module")
//	staff.KillAll(ctx)
package bugle
