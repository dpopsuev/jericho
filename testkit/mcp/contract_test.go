package mcp

import (
	"testing"

	"github.com/dpopsuev/jericho/internal/protocol"
)

func TestServerContract_MockServer(t *testing.T) {
	RunServerContract(t, func() protocol.Server {
		return NewMockServer()
	})
}
