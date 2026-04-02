// Package jericho is an AI Agent Broker — it provides the primitives
// that make multi-agent orchestration possible without vendor lock-in.
//
// Jericho does not orchestrate. Directors (Origami, Djinn, custom) bring
// orchestration strategies. Jericho provides the actors.
//
// Public API — 3 interfaces, 6 methods:
//
//	broker := jericho.NewBroker(...)
//	actors, _ := broker.Pick(ctx, prefs)    // what's available?
//	actor, _ := broker.Spawn(ctx, config)   // hire an actor
//	response, _ := actor.Perform(ctx, prompt) // do work
//	actor.Ready()                           // health check
//	actor.Kill(ctx)                         // stop
//
// Directors implement the orchestration strategy:
//
//	events, _ := director.Direct(ctx, broker)
//	for ev := range events {
//	    fmt.Println(ev.Kind, ev.Step, ev.Agent)
//	}
package jericho
