package bugle

// ProtocolVersion is the current Bugle Protocol version.
const ProtocolVersion = "bugle/v1"

// Capabilities declares which protocol layers a server supports.
// Returned on start response.
type Capabilities struct {
	Protocol    string          `json:"protocol"`
	Layers      LayerSupport    `json:"layers"`
	AndonLevels []AndonLevelDef `json:"andon_levels,omitempty"`
}

// LayerSupport indicates which optional layers the server implements.
type LayerSupport struct {
	Andon  bool `json:"andon"`
	Budget bool `json:"budget"`
	HITL   bool `json:"hitl"`
	Status bool `json:"status"`
}

// DefaultCapabilities returns Core-only capabilities (Level 0)
// with the reserved andon vocabulary.
func DefaultCapabilities() Capabilities {
	return Capabilities{
		Protocol:    ProtocolVersion,
		AndonLevels: DefaultVocabulary(),
	}
}

// FullCapabilities returns all layers enabled with the reserved andon vocabulary.
func FullCapabilities() Capabilities {
	return Capabilities{
		Protocol: ProtocolVersion,
		Layers: LayerSupport{
			Andon:  true,
			Budget: true,
			HITL:   true,
			Status: true,
		},
		AndonLevels: DefaultVocabulary(),
	}
}
