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
// actors. `_Ext` variants share the same gray as their non-ext
// counterpart's external sibling so a queue/db external still
// reads as "external" first.
func DefaultTheme() Theme {
	system := RolePalette{Fill: "#1168BD", Stroke: "#0B4884", Text: "white"}
	systemExt := RolePalette{Fill: "#999999", Stroke: "#6B6B6B", Text: "white"}
	container := RolePalette{Fill: "#438DD5", Stroke: "#3C7FC0", Text: "white"}
	containerExt := RolePalette{Fill: "#B3B3B3", Stroke: "#9A9A9A", Text: "white"}
	component := RolePalette{Fill: "#85BBF0", Stroke: "#78A8D8", Text: "#000"}
	componentExt := RolePalette{Fill: "#CCCCCC", Stroke: "#B0B0B0", Text: "#000"}
	deployment := RolePalette{Fill: "#FFFFFF", Stroke: "#444444", Text: "#222"}
	return Theme{
		Roles: map[diagram.C4ElementKind]RolePalette{
			diagram.C4ElementPerson:    {Fill: "#08427B", Stroke: "#073B6F", Text: "white"},
			diagram.C4ElementPersonExt: {Fill: "#686868", Stroke: "#4D4D4D", Text: "white"},

			diagram.C4ElementSystem:          system,
			diagram.C4ElementSystemExt:       systemExt,
			diagram.C4ElementSystemDB:        system,
			diagram.C4ElementSystemDBExt:     systemExt,
			diagram.C4ElementSystemQueue:     system,
			diagram.C4ElementSystemQueueExt:  systemExt,

			diagram.C4ElementContainer:          container,
			diagram.C4ElementContainerExt:       containerExt,
			diagram.C4ElementContainerDB:        container,
			diagram.C4ElementContainerDBExt:     containerExt,
			diagram.C4ElementContainerQueue:     container,
			diagram.C4ElementContainerQueueExt:  containerExt,

			diagram.C4ElementComponent:          component,
			diagram.C4ElementComponentExt:       componentExt,
			diagram.C4ElementComponentDB:        component,
			diagram.C4ElementComponentDBExt:     componentExt,
			diagram.C4ElementComponentQueue:     component,
			diagram.C4ElementComponentQueueExt:  componentExt,

			diagram.C4ElementDeploymentNode: deployment,
		},
		TitleText:  "#333",
		EdgeStroke: "#333",
		EdgeText:   "#333",
		Background: "#fff",
	}
}

// IsDBKind reports whether the kind should render as a cylinder.
func IsDBKind(k diagram.C4ElementKind) bool {
	switch k {
	case diagram.C4ElementSystemDB, diagram.C4ElementSystemDBExt,
		diagram.C4ElementContainerDB, diagram.C4ElementContainerDBExt,
		diagram.C4ElementComponentDB, diagram.C4ElementComponentDBExt:
		return true
	}
	return false
}

// IsQueueKind reports whether the kind should render as a queue
// (stadium / pill) shape.
func IsQueueKind(k diagram.C4ElementKind) bool {
	switch k {
	case diagram.C4ElementSystemQueue, diagram.C4ElementSystemQueueExt,
		diagram.C4ElementContainerQueue, diagram.C4ElementContainerQueueExt,
		diagram.C4ElementComponentQueue, diagram.C4ElementComponentQueueExt:
		return true
	}
	return false
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
