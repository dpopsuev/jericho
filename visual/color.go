// Package symbol provides visual identity primitives for agents.
// Color, Element, Persona — the cosmetic layer, always rendered.
// Absorbs palette/, element/, identity/ from pre-v0.2.0.
package visual

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"

	"github.com/dpopsuev/tangle/world"
)

// ColorType is the ComponentType for Color (visual identity ECS component).
const ColorType world.ComponentType = "color"

// Shade is a color family grouping for agent collectives.
type Shade struct {
	Name   string
	Colors []PaletteColor
}

// PaletteColor is a specific color within a shade family.
type PaletteColor struct {
	Name string
	Hex  string
}

// Color is the visual identity ECS component for agents.
// Format: "Denim Writer of Indigo Refactor" (Color Role of Shade Collective).
type Color struct {
	Family     string `json:"family"`     // color family: "red", "blue"
	Shade      string `json:"shade"`      // shade within family: "crimson", "azure"
	Name       string `json:"name"`       // display name (= shade): "crimson"
	Role       string `json:"role"`       // function: "worker", "reviewer"
	Collective string `json:"collective"` // formation: "rca", "triage"
	Hex        string `json:"hex"`        // CSS hex: "#DC143C"
}

// ComponentType implements world.Component.
func (Color) ComponentType() world.ComponentType { return ColorType }

// FQDN returns the fully qualified agent address: shade.family.director.broker.
func (c Color) FQDN(director, broker string) string { //nolint:gocritic // value receiver for ECS
	return fmt.Sprintf("%s.%s.%s.%s", c.Shade, c.Family, director, broker)
}

// Title returns the heraldic name: "Denim Writer of Indigo Refactor".
func (c Color) Title() string { //nolint:gocritic // value receiver needed for ECS Get[T]
	return fmt.Sprintf("%s %s of %s %s", c.Name, c.Role, c.Shade, c.Collective)
}

// Label returns the compact log format: "[Indigo·Denim|Writer]".
func (c Color) Label() string { //nolint:gocritic // value receiver needed for ECS Get[T]
	return fmt.Sprintf("[%s·%s|%s]", c.Shade, c.Name, c.Role)
}

// Short returns just the color name: "Denim".
func (c Color) Short() string { return c.Name } //nolint:gocritic // value receiver

// ContrastMode indicates whether the terminal uses light or dark background.
type ContrastMode string

const (
	ContrastAuto  ContrastMode = "auto" // detect from terminal
	ContrastDark  ContrastMode = "dark"
	ContrastLight ContrastMode = "light"
)

// ANSI returns a 24-bit true color ANSI escape sequence for foreground text.
func (c Color) ANSI() string { //nolint:gocritic // value receiver for ECS
	if len(c.Hex) != 7 {
		return ""
	}
	var r, g, b uint8
	fmt.Sscanf(c.Hex, "#%02x%02x%02x", &r, &g, &b) //nolint:errcheck // hex format guaranteed by palette
	return fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b)
}

// Reservation is a color preference, not an assignment.
// Used by persona templates to express preferred shade/color without
// locking in a specific assignment (the Registry handles collisions).
type Reservation struct {
	Shade string // preferred shade family (empty = any)
	Color string // preferred color (empty = any in shade)
}

// Palette defines 12 color families × 12 shades = 144 unique agent addresses.
// All shade names sourced from real color vocabulary.
var Palette = DefaultPalette()

// LookupShade finds a shade by name. Returns nil if not found.
func LookupShade(name string) *Shade {
	for i := range Palette {
		if Palette[i].Name == name {
			return &Palette[i]
		}
	}
	return nil
}

// LookupColor finds a color by name across all shades.
// Returns the color and its parent shade name.
func LookupColor(name string) (PaletteColor, string, bool) {
	for _, shade := range Palette {
		for _, c := range shade.Colors {
			if c.Name == name {
				return c, shade.Name, true
			}
		}
	}
	return PaletteColor{}, "", false
}

// Sentinel errors for color registry operations.
var (
	ErrAllSlotsAssigned = errors.New("symbol: all 56 color slots are assigned")
	ErrUnknownShade     = errors.New("symbol: unknown shade")
	ErrShadeExhausted   = errors.New("symbol: all colors in shade are assigned")
	ErrUnknownColor     = errors.New("symbol: unknown color")
	ErrColorWrongShade  = errors.New("symbol: color belongs to different shade")
	ErrAlreadyAssigned  = errors.New("symbol: color already assigned")
)

// Registry manages color identity assignment with collision prevention.
type Registry struct {
	mu       sync.Mutex
	assigned map[string]bool // "shade·color" → true
}

// NewRegistry creates an empty color registry.
func NewRegistry() *Registry {
	return &Registry{
		assigned: make(map[string]bool),
	}
}

func registryKey(shade, color string) string {
	return shade + "·" + color
}

// Assign returns a Color with a random available color.
func (r *Registry) Assign(role, collective string) (Color, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	shades := make([]int, len(Palette))
	for i := range shades {
		shades[i] = i
	}
	rand.Shuffle(len(shades), func(i, j int) { shades[i], shades[j] = shades[j], shades[i] })

	for _, si := range shades {
		shade := Palette[si]
		for _, color := range shade.Colors {
			key := registryKey(shade.Name, color.Name)
			if !r.assigned[key] {
				r.assigned[key] = true
				return Color{
					Family:     shade.Name,
					Shade:      color.Name,
					Name:       color.Name,
					Role:       role,
					Collective: collective,
					Hex:        color.Hex,
				}, nil
			}
		}
	}
	return Color{}, ErrAllSlotsAssigned
}

// AssignInGroup returns a Color from a specific shade family.
func (r *Registry) AssignInGroup(shade, role, collective string) (Color, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	s := LookupShade(shade)
	if s == nil {
		return Color{}, fmt.Errorf("%w: %q", ErrUnknownShade, shade)
	}

	for _, color := range s.Colors {
		key := registryKey(s.Name, color.Name)
		if !r.assigned[key] {
			r.assigned[key] = true
			return Color{
				Family:     s.Name,
				Shade:      color.Name,
				Name:       color.Name,
				Role:       role,
				Collective: collective,
				Hex:        color.Hex,
			}, nil
		}
	}
	return Color{}, fmt.Errorf("%w: %q", ErrShadeExhausted, shade)
}

// AssignWithPreference tries the preferred shade+color, falls back if taken.
func (r *Registry) AssignWithPreference(res Reservation, role, collective string) (Color, error) {
	if res.Shade != "" && res.Color != "" {
		c, err := r.Set(res.Shade, res.Color, role, collective)
		if err == nil {
			return c, nil
		}
	}
	if res.Shade != "" {
		return r.AssignInGroup(res.Shade, role, collective)
	}
	return r.Assign(role, collective)
}

// Set explicitly assigns a specific shade+color combination.
func (r *Registry) Set(shade, color, role, collective string) (Color, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	c, foundShade, ok := LookupColor(color)
	if !ok {
		return Color{}, fmt.Errorf("%w: %q", ErrUnknownColor, color)
	}
	if foundShade != shade {
		return Color{}, fmt.Errorf("%w: %q belongs to shade %q, not %q", ErrColorWrongShade, color, foundShade, shade)
	}

	key := registryKey(shade, color)
	if r.assigned[key] {
		return Color{}, fmt.Errorf("%w: %s·%s", ErrAlreadyAssigned, shade, color)
	}

	r.assigned[key] = true
	return Color{
		Family:     foundShade,
		Shade:      c.Name,
		Name:       c.Name,
		Role:       role,
		Collective: collective,
		Hex:        c.Hex,
	}, nil
}

// Release returns a color to the available pool.
func (r *Registry) Release(c Color) { //nolint:gocritic // value param for API simplicity
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.assigned, registryKey(c.Shade, c.Name))
}

// Active returns the count of currently assigned colors.
func (r *Registry) Active() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.assigned)
}
