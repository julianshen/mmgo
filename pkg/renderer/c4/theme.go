package c4

import "github.com/julianshen/mmgo/pkg/diagram"

// RolePalette is the fill/stroke/text triple for a single C4 element
// role (Person, System, Container, Component, plus their *Ext/DB
// variants). Exported as a nested type so callers can override
// individual roles without hand-rebuilding the full map.
type RolePalette struct {
	Fill, Stroke, Text string
}

// Theme holds all C4 color surfaces: a per-role palette map plus
// chrome (background, title text, edge stroke/text, arrow fill).
type Theme struct {
	Roles      map[diagram.C4ElementKind]RolePalette
	TitleText  string
	EdgeStroke string
	EdgeText   string
	Background string
}

// DefaultTheme returns the Mermaid-classic C4 palette — the
// blue-scale hierarchy where people/systems/containers/components
// shade from dark navy down to pale blue, with gray for external
// actors.
func DefaultTheme() Theme {
	return Theme{
		Roles: map[diagram.C4ElementKind]RolePalette{
			diagram.C4ElementPerson:       {"#08427B", "#073B6F", "white"},
			diagram.C4ElementPersonExt:    {"#686868", "#4D4D4D", "white"},
			diagram.C4ElementSystem:       {"#1168BD", "#0B4884", "white"},
			diagram.C4ElementSystemExt:    {"#999999", "#6B6B6B", "white"},
			diagram.C4ElementSystemDB:     {"#1168BD", "#0B4884", "white"},
			diagram.C4ElementContainer:    {"#438DD5", "#3C7FC0", "white"},
			diagram.C4ElementContainerDB:  {"#438DD5", "#3C7FC0", "white"},
			diagram.C4ElementComponent:    {"#85BBF0", "#78A8D8", "#000"},
		},
		TitleText:  "#333",
		EdgeStroke: "#333",
		EdgeText:   "#333",
		Background: "#fff",
	}
}

// roleOf returns the palette for kind, falling back to the System
// palette when the kind is missing (unreachable in practice, but
// keeps the renderer crash-free for future kinds).
func (t Theme) roleOf(kind diagram.C4ElementKind) RolePalette {
	if p, ok := t.Roles[kind]; ok {
		return p
	}
	return t.Roles[diagram.C4ElementSystem]
}

func resolveTheme(opts *Options) Theme {
	th := DefaultTheme()
	if opts == nil {
		return th
	}
	if len(opts.Theme.Roles) > 0 {
		// Merge: caller entries override defaults; missing kinds keep
		// their default so a partial override doesn't blank the rest.
		for k, v := range opts.Theme.Roles {
			th.Roles[k] = v
		}
	}
	if opts.Theme.TitleText != "" {
		th.TitleText = opts.Theme.TitleText
	}
	if opts.Theme.EdgeStroke != "" {
		th.EdgeStroke = opts.Theme.EdgeStroke
	}
	if opts.Theme.EdgeText != "" {
		th.EdgeText = opts.Theme.EdgeText
	}
	if opts.Theme.Background != "" {
		th.Background = opts.Theme.Background
	}
	return th
}
