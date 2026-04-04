// Package intent provides the mission/purpose component for agents.
// Intent is a consumer-defined string describing what the agent is for.
// Independent of symbols and traits — changing intent doesn't change
// how the agent looks or behaves.
package identity

import "github.com/dpopsuev/troupe/world"

// MissionType is the ComponentType for Mission.
const MissionType world.ComponentType = "mission"

// Mission describes the agent's purpose. Consumer-defined, per session.
type Mission struct {
	Purpose string `json:"purpose"`
}

// ComponentType implements world.Component.
func (Mission) ComponentType() world.ComponentType { return MissionType }
