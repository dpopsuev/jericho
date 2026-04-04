package mcp

import (
	"testing"

	"github.com/dpopsuev/troupe/internal/protocol"
)

func TestServerContract_MockServer(t *testing.T) {
	RunServerContract(t, func() protocol.Server {
		return NewMockServer()
	})
}
