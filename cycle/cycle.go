// Package cycle defines the generative and destructive interaction rules
// between framework elements (Wuxing cycle).
package cycle

import (
	"github.com/dpopsuev/bugle/element"
)

// Type represents the interaction mode between elements.
type Type string

const (
	Generative  Type = "generative"
	Destructive Type = "destructive"
)

// Rule defines a directed interaction between two elements.
type Rule struct {
	Cycle       Type            `json:"cycle"`
	From        element.Element `json:"from"`
	To          element.Element `json:"to"`
	Interaction string          `json:"interaction"`
}

// GenerativeCycle returns the generative (sheng) cycle rules.
// Four main transitions plus two modifier rules for Lightning and Diamond.
func GenerativeCycle() []Rule {
	return []Rule{
		{Generative, element.ElementFire, element.ElementEarth, "classification provides structure for steady investigation"},
		{Generative, element.ElementEarth, element.ElementWater, "stable repo selection enables deep code investigation"},
		{Generative, element.ElementWater, element.ElementAir, "deep evidence enables holistic synthesis"},
		{Generative, element.ElementAir, element.ElementFire, "synthesis reveals patterns for re-classification"},
		{Generative, element.ElementLightning, element.ElementLightning, "lightning shortcuts any generative step"},
		{Generative, element.ElementDiamond, element.ElementDiamond, "diamond validates any generative step"},
	}
}

// DestructiveCycle returns the destructive (ke) cycle rules.
// Six adversarial pairings where one element challenges another.
func DestructiveCycle() []Rule {
	return []Rule{
		{Destructive, element.ElementFire, element.ElementWater, "aggressive challenge forces deeper evidence"},
		{Destructive, element.ElementWater, element.ElementEarth, "depth destabilizes stable conclusions"},
		{Destructive, element.ElementEarth, element.ElementFire, "methodical evidence extinguishes hasty challenges"},
		{Destructive, element.ElementLightning, element.ElementDiamond, "speed exposes brittleness to ambiguity"},
		{Destructive, element.ElementDiamond, element.ElementAir, "precision grounds vague synthesis"},
		{Destructive, element.ElementAir, element.ElementLightning, "breadth covers narrow shortcut mistakes"},
	}
}

var generativeNext = map[element.Element]element.Element{
	element.ElementFire:  element.ElementEarth,
	element.ElementEarth: element.ElementWater,
	element.ElementWater: element.ElementAir,
	element.ElementAir:   element.ElementFire,
}

var destructiveTarget = map[element.Element]element.Element{
	element.ElementFire:      element.ElementWater,
	element.ElementWater:     element.ElementEarth,
	element.ElementEarth:     element.ElementFire,
	element.ElementLightning: element.ElementDiamond,
	element.ElementDiamond:   element.ElementAir,
	element.ElementAir:       element.ElementLightning,
}

var destructiveSource = map[element.Element]element.Element{
	element.ElementWater:     element.ElementFire,
	element.ElementEarth:     element.ElementWater,
	element.ElementFire:      element.ElementEarth,
	element.ElementDiamond:   element.ElementLightning,
	element.ElementAir:       element.ElementDiamond,
	element.ElementLightning: element.ElementAir,
}

// NextGenerative returns the element that naturally follows the given
// element in the generative cycle. Returns empty if not in the main cycle
// (Lightning and Diamond are modifiers, not part of the main sequence).
func NextGenerative(from element.Element) element.Element {
	return generativeNext[from]
}

// Challenges returns the element that the given element challenges
// in the destructive cycle.
func Challenges(from element.Element) element.Element {
	return destructiveTarget[from]
}

// ChallengedBy returns the element that challenges the given element
// in the destructive cycle.
func ChallengedBy(target element.Element) element.Element {
	return destructiveSource[target]
}
