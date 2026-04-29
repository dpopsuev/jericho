package testkit

import (
	"context"

	"github.com/dpopsuev/tangle/internal/transport"
)

// EchoHandler returns a handler that echoes the message back.
func EchoHandler() transport.MsgHandler {
	return func(_ context.Context, msg transport.Message) (transport.Message, error) {
		return transport.Message{
			From:    msg.To,
			To:      msg.From,
			Role:    "agent",
			Content: msg.Content,
		}, nil
	}
}

// ErrorHandler returns a handler that always fails with the given error.
func ErrorHandler(err error) transport.MsgHandler {
	return func(_ context.Context, _ transport.Message) (transport.Message, error) {
		return transport.Message{}, err
	}
}
