package broker_test

import (
	"testing"

	"github.com/dpopsuev/tangle/broker"
	"github.com/dpopsuev/tangle/internal/transport"
	"github.com/dpopsuev/tangle/testkit"
	"github.com/dpopsuev/tangle/world"
)

func TestLobby_AdmissionContract(t *testing.T) {
	w := world.NewWorld()
	tr := transport.NewLocalTransport()
	log := testkit.NewStubEventLog()

	lobby := broker.NewLobby(broker.LobbyConfig{
		World:      w,
		Transport:  tr,
		ControlLog: log,
	})

	testkit.RunAdmissionContract(t, testkit.AdmissionTestDeps{
		Admission:  lobby,
		ControlLog: log,
		WorldCount: w.Count,
	})
}
