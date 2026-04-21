package er

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
	if opts.Theme.EntityFill != "" {
		th.EntityFill = opts.Theme.EntityFill
	}
	if opts.Theme.EntityStroke != "" {
		th.EntityStroke = opts.Theme.EntityStroke
	}
	if opts.Theme.EntityText != "" {
		th.EntityText = opts.Theme.EntityText
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
