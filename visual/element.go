package visual

// Element represents a visual identity archetype — cosmetic only.
// Does not drive model selection or behavioral config.
type Element string

const (
	ElementFire      Element = "fire"
	ElementLightning Element = "lightning"
	ElementEarth     Element = "earth"
	ElementDiamond   Element = "diamond"
	ElementWater     Element = "water"
	ElementAir       Element = "air"
)

var coreElements = []Element{
	ElementFire, ElementLightning, ElementEarth,
	ElementDiamond, ElementWater, ElementAir,
}

// AllElements returns the six core elements.
func AllElements() []Element {
	out := make([]Element, len(coreElements))
	copy(out, coreElements)
	return out
}
