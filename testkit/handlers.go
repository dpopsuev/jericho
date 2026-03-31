package testkit

import (
	"context"

	"github.com/dpopsuev/jericho/signal"
	"github.com/dpopsuev/jericho/transport"
)

// EchoHandler returns a handler that echoes the message back with Confirm performative.
func EchoHandler() transport.Handler {
	return func(_ context.Context, msg transport.Message) (transport.Message, error) {
		return transport.Message{
			From:         msg.To,
			To:           msg.From,
			Performative: signal.Confirm,
			Content:      msg.Content,
		}, nil
	}
}

// StubHandler returns a handler that replies with a fixed performative.
func StubHandler(reply signal.Performative) transport.Handler {
	return func(_ context.Context, msg transport.Message) (transport.Message, error) {
		return transport.Message{
			From:         msg.To,
			To:           msg.From,
			Performative: reply,
			Content:      msg.Content,
		}, nil
	}
}

// ErrorHandler returns a handler that always fails with the given error.
func ErrorHandler(err error) transport.Handler {
	return func(_ context.Context, _ transport.Message) (transport.Message, error) {
		return transport.Message{}, err
	}
}
