package visual

// palette defines the 12×12 color matrix for agent FQDN addressing.
// 12 color families × 12 shades = 144 unique agent addresses.
// All shade names sourced from real color vocabulary.

// DefaultPalette returns the 12 color families with 12 shades each.
func DefaultPalette() []Shade {
	return []Shade{
		{Name: "red", Colors: []PaletteColor{
			{Name: "crimson", Hex: "#DC143C"}, {Name: "scarlet", Hex: "#FF2400"}, {Name: "ruby", Hex: "#E0115F"},
			{Name: "cardinal", Hex: "#C41E3A"}, {Name: "vermillion", Hex: "#E34234"}, {Name: "garnet", Hex: "#733635"},
			{Name: "rosewood", Hex: "#65000B"}, {Name: "carmine", Hex: "#960018"}, {Name: "cinnabar", Hex: "#E44D2E"},
			{Name: "claret", Hex: "#7F1734"}, {Name: "cordovan", Hex: "#893F45"}, {Name: "jasper", Hex: "#D73B3E"},
		}},
		{Name: "orange", Colors: []PaletteColor{
			{Name: "coral", Hex: "#FF7F50"}, {Name: "tangerine", Hex: "#FF9966"}, {Name: "amber", Hex: "#FFBF00"},
			{Name: "apricot", Hex: "#FBCEB1"}, {Name: "flame", Hex: "#E25822"}, {Name: "persimmon", Hex: "#EC5800"},
			{Name: "saffron", Hex: "#F4C430"}, {Name: "tawny", Hex: "#CD5700"}, {Name: "papaya", Hex: "#FFEFD5"},
			{Name: "melon", Hex: "#FEBAAD"}, {Name: "marigold", Hex: "#EAA221"}, {Name: "pumpkin", Hex: "#FF7518"},
		}},
		{Name: "yellow", Colors: []PaletteColor{
			{Name: "gold", Hex: "#FFD700"}, {Name: "canary", Hex: "#FFEF00"}, {Name: "citrine", Hex: "#E4D00A"},
			{Name: "ivory", Hex: "#FFFFF0"}, {Name: "mustard", Hex: "#FFDB58"}, {Name: "flax", Hex: "#EEDC82"},
			{Name: "jasmine", Hex: "#F8DE7E"}, {Name: "champagne", Hex: "#F7E7CE"}, {Name: "cream", Hex: "#FFFDD0"},
			{Name: "honey", Hex: "#EB9605"}, {Name: "straw", Hex: "#E4D96F"}, {Name: "buttercup", Hex: "#F9E154"},
		}},
		{Name: "green", Colors: []PaletteColor{
			{Name: "emerald", Hex: "#50C878"}, {Name: "jade", Hex: "#00A86B"}, {Name: "sage", Hex: "#BCB88A"},
			{Name: "olive", Hex: "#808000"}, {Name: "fern", Hex: "#4F7942"}, {Name: "malachite", Hex: "#0BDA51"},
			{Name: "pistachio", Hex: "#93C572"}, {Name: "viridian", Hex: "#40826D"}, {Name: "moss", Hex: "#8A9A5B"},
			{Name: "mint", Hex: "#3EB489"}, {Name: "juniper", Hex: "#3A5311"}, {Name: "laurel", Hex: "#A9BA9D"},
		}},
		{Name: "cyan", Colors: []PaletteColor{
			{Name: "teal", Hex: "#008080"}, {Name: "turquoise", Hex: "#40E0D0"}, {Name: "aqua", Hex: "#00FFFF"},
			{Name: "cerulean", Hex: "#007BA7"}, {Name: "capri", Hex: "#00BFFF"}, {Name: "celeste", Hex: "#B2FFFF"},
			{Name: "keppel", Hex: "#3AB09E"}, {Name: "moonstone", Hex: "#3AA8C1"}, {Name: "lagoon", Hex: "#017987"},
			{Name: "opal", Hex: "#A8C3BC"}, {Name: "seafoam", Hex: "#93E9BE"}, {Name: "frost", Hex: "#A1CAF1"},
		}},
		{Name: "blue", Colors: []PaletteColor{
			{Name: "azure", Hex: "#007FFF"}, {Name: "cobalt", Hex: "#0047AB"}, {Name: "sapphire", Hex: "#0F52BA"},
			{Name: "navy", Hex: "#000080"}, {Name: "denim", Hex: "#1560BD"}, {Name: "indigo", Hex: "#4B0082"},
			{Name: "lapis", Hex: "#26619C"}, {Name: "periwinkle", Hex: "#CCCCFF"}, {Name: "midnight", Hex: "#191970"},
			{Name: "aegean", Hex: "#1F456E"}, {Name: "horizon", Hex: "#5A86AD"}, {Name: "glacier", Hex: "#80B3C4"},
		}},
		{Name: "violet", Colors: []PaletteColor{
			{Name: "amethyst", Hex: "#9966CC"}, {Name: "iris", Hex: "#5A4FCF"}, {Name: "lilac", Hex: "#C8A2C8"},
			{Name: "orchid", Hex: "#DA70D6"}, {Name: "wisteria", Hex: "#C9A0DC"}, {Name: "heliotrope", Hex: "#DF73FF"},
			{Name: "mauve", Hex: "#E0B0FF"}, {Name: "plum", Hex: "#8E4585"}, {Name: "thistle", Hex: "#D8BFD8"},
			{Name: "lavender", Hex: "#E6E6FA"}, {Name: "mulberry", Hex: "#C54B8C"}, {Name: "byzantium", Hex: "#702963"},
		}},
		{Name: "pink", Colors: []PaletteColor{
			{Name: "rose", Hex: "#FF007F"}, {Name: "blush", Hex: "#DE5D83"}, {Name: "fuchsia", Hex: "#FF00FF"},
			{Name: "salmon", Hex: "#FA8072"}, {Name: "peach", Hex: "#FFCBA4"}, {Name: "carnation", Hex: "#FFA6C9"},
			{Name: "cerise", Hex: "#DE3163"}, {Name: "magenta", Hex: "#FF0090"}, {Name: "flamingo", Hex: "#FC8EAC"},
			{Name: "petal", Hex: "#F4C2C2"}, {Name: "begonia", Hex: "#FA6775"}, {Name: "primrose", Hex: "#F6E3CE"},
		}},
		{Name: "brown", Colors: []PaletteColor{
			{Name: "umber", Hex: "#635147"}, {Name: "sienna", Hex: "#A0522D"}, {Name: "chestnut", Hex: "#954535"},
			{Name: "bronze", Hex: "#CD7F32"}, {Name: "copper", Hex: "#B87333"}, {Name: "walnut", Hex: "#773F1A"},
			{Name: "caramel", Hex: "#FFD59A"}, {Name: "mahogany", Hex: "#C04000"}, {Name: "cocoa", Hex: "#D2691E"},
			{Name: "toffee", Hex: "#755139"}, {Name: "russet", Hex: "#80461B"}, {Name: "hickory", Hex: "#674C47"},
		}},
		{Name: "gray", Colors: []PaletteColor{
			{Name: "slate", Hex: "#708090"}, {Name: "ash", Hex: "#B2BEB5"}, {Name: "charcoal", Hex: "#36454F"},
			{Name: "silver", Hex: "#C0C0C0"}, {Name: "gunmetal", Hex: "#2A3439"}, {Name: "iron", Hex: "#48494B"},
			{Name: "steel", Hex: "#71797E"}, {Name: "pewter", Hex: "#8E8E8E"}, {Name: "flint", Hex: "#6D6D6D"},
			{Name: "smoke", Hex: "#738276"}, {Name: "graphite", Hex: "#383838"}, {Name: "fossil", Hex: "#787276"},
		}},
		{Name: "white", Colors: []PaletteColor{
			{Name: "snow", Hex: "#FFFAFA"}, {Name: "pearl", Hex: "#EAE0C8"}, {Name: "alabaster", Hex: "#F2F0E6"},
			{Name: "bone", Hex: "#E3DAC9"}, {Name: "linen", Hex: "#FAF0E6"}, {Name: "porcelain", Hex: "#F0EAD6"},
			{Name: "eggshell", Hex: "#F0EAD6"}, {Name: "chalk", Hex: "#FDFDFD"}, {Name: "cloud", Hex: "#F2F3F4"},
			{Name: "milk", Hex: "#FDFFF5"}, {Name: "cotton", Hex: "#FBFBF9"}, {Name: "marble", Hex: "#F2F0EB"},
		}},
		{Name: "black", Colors: []PaletteColor{
			{Name: "onyx", Hex: "#353839"}, {Name: "jet", Hex: "#343434"}, {Name: "ebony", Hex: "#555D50"},
			{Name: "obsidian", Hex: "#3C3C3C"}, {Name: "raven", Hex: "#0E0E10"}, {Name: "coal", Hex: "#3B3C36"},
			{Name: "sable", Hex: "#3B3B3B"}, {Name: "shadow", Hex: "#292929"}, {Name: "ink", Hex: "#1B1B1B"},
			{Name: "pitch", Hex: "#1C1C1C"}, {Name: "void", Hex: "#0A0A0A"}, {Name: "phantom", Hex: "#1A1A2E"},
		}},
	}
}
