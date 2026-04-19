package visual

import (
	"strings"
	"testing"
)

func TestColor_FQDN_Format(t *testing.T) {
	c := Color{Family: "red", Shade: "crimson"}
	got := c.FQDN("origami", "local")
	want := "crimson.red.origami.local"
	if got != want {
		t.Errorf("FQDN = %q, want %q", got, want)
	}
}

func TestColor_FQDN_AllSegments(t *testing.T) {
	c := Color{Family: "blue", Shade: "azure"}
	fqdn := c.FQDN("djinn", "prod")
	parts := strings.Split(fqdn, ".")
	if len(parts) != 4 {
		t.Fatalf("FQDN has %d segments, want 4: %q", len(parts), fqdn)
	}
	if parts[0] != "azure" {
		t.Errorf("shade = %q, want azure", parts[0])
	}
	if parts[1] != "blue" {
		t.Errorf("family = %q, want blue", parts[1])
	}
	if parts[2] != "djinn" {
		t.Errorf("director = %q, want djinn", parts[2])
	}
	if parts[3] != "prod" {
		t.Errorf("broker = %q, want prod", parts[3])
	}
}

func TestDefaultPalette_12Families(t *testing.T) {
	palette := DefaultPalette()
	if len(palette) != 12 {
		t.Errorf("palette has %d families, want 12", len(palette))
	}
}

func TestDefaultPalette_12ShadesPerFamily(t *testing.T) {
	for _, family := range DefaultPalette() {
		if len(family.Colors) != 12 {
			t.Errorf("family %s has %d shades, want 12", family.Name, len(family.Colors))
		}
	}
}

func TestDefaultPalette_144UniqueShades(t *testing.T) {
	seen := make(map[string]bool)
	for _, family := range DefaultPalette() {
		for _, shade := range family.Colors {
			key := shade.Name
			if seen[key] {
				t.Errorf("duplicate shade name: %q", key)
			}
			seen[key] = true
		}
	}
	if len(seen) != 144 {
		t.Errorf("unique shades = %d, want 144", len(seen))
	}
}

func TestRegistry_Assign_PopulatesFamily(t *testing.T) {
	reg := NewRegistry()
	c, err := reg.Assign("worker", "test")
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}
	if c.Family == "" {
		t.Error("Family is empty after Assign")
	}
	if c.Shade == "" {
		t.Error("Shade is empty after Assign")
	}
	// FQDN should work
	fqdn := c.FQDN("origami", "local")
	if !strings.Contains(fqdn, c.Family) {
		t.Errorf("FQDN %q missing family %q", fqdn, c.Family)
	}
}

func TestRegistry_Assign_144UniqueColors(t *testing.T) {
	reg := NewRegistry()
	seen := make(map[string]bool)
	for i := range 144 {
		c, err := reg.Assign("worker", "test")
		if err != nil {
			t.Fatalf("Assign #%d: %v", i+1, err)
		}
		fqdn := c.FQDN("test", "local")
		if seen[fqdn] {
			t.Fatalf("duplicate FQDN at #%d: %s", i+1, fqdn)
		}
		seen[fqdn] = true
	}
}

func TestRegistry_Assign_145th_Exhausted(t *testing.T) {
	reg := NewRegistry()
	for i := range 144 {
		_, err := reg.Assign("worker", "test")
		if err != nil {
			t.Fatalf("Assign #%d: %v", i+1, err)
		}
	}
	_, err := reg.Assign("worker", "test")
	if err == nil {
		t.Fatal("expected error on 145th assign")
	}
}

func TestDefaultPalette_AllHaveHex(t *testing.T) {
	for _, family := range DefaultPalette() {
		for _, shade := range family.Colors {
			if shade.Hex == "" {
				t.Errorf("shade %s.%s has empty hex", family.Name, shade.Name)
			}
			if shade.Hex[0] != '#' {
				t.Errorf("shade %s.%s hex %q doesn't start with #", family.Name, shade.Name, shade.Hex)
			}
		}
	}
}

func TestDefaultPalette_FamilyNames(t *testing.T) {
	expected := []string{"red", "orange", "yellow", "green", "cyan", "blue", "violet", "pink", "brown", "gray", "white", "black"}
	palette := DefaultPalette()
	for i, want := range expected {
		if palette[i].Name != want {
			t.Errorf("family[%d] = %q, want %q", i, palette[i].Name, want)
		}
	}
}
