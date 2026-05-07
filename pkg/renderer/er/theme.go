package er

import "github.com/julianshen/mmgo/pkg/renderer/svgutil"

// Theme holds the color surfaces the ER renderer consumes. Shape
// mirrors pkg/renderer/class/theme.go so the SVG dispatcher can map
// config.ThemeColors uniformly across box-and-edge renderers. Unset
// fields fall back to DefaultTheme values via resolveTheme.
type Theme struct {
	EntityFill   string
	EntityStroke string
	EntityText   string
	EdgeStroke   string
	EdgeText     string
	Background   string
}

// DefaultTheme returns the Mermaid-classic ER palette (light purple
// boxes, dark purple borders, black text). Used when no explicit
// theme is wired through Options.
func DefaultTheme() Theme {
	return Theme{
		EntityFill:   "#ECECFF",
		EntityStroke: "#9370DB",
		EntityText:   "#333",
		EdgeStroke:   "#333",
		EdgeText:     "#333",
		Background:   "#fff",
	}
}

func resolveTheme(opts *Options) Theme {
	th := DefaultTheme()
	if opts == nil {
		return th
	}
	svgutil.MergeStr(&th.EntityFill, opts.Theme.EntityFill)
	svgutil.MergeStr(&th.EntityStroke, opts.Theme.EntityStroke)
	svgutil.MergeStr(&th.EntityText, opts.Theme.EntityText)
	svgutil.MergeStr(&th.EdgeStroke, opts.Theme.EdgeStroke)
	svgutil.MergeStr(&th.EdgeText, opts.Theme.EdgeText)
	svgutil.MergeStr(&th.Background, opts.Theme.Background)
	return th
}
