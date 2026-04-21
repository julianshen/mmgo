package block

// Theme holds the color surfaces the block renderer consumes.
// Mirrors class/er/state in shape for dispatcher uniformity.
type Theme struct {
	NodeFill   string
	NodeStroke string
	NodeText   string
	EdgeStroke string
	EdgeText   string
	Background string
}

func DefaultTheme() Theme {
	return Theme{
		NodeFill:   "#ECECFF",
		NodeStroke: "#9370DB",
		NodeText:   "#333",
		EdgeStroke: "#333",
		EdgeText:   "#333",
		Background: "#fff",
	}
}

func resolveTheme(opts *Options) Theme {
	th := DefaultTheme()
	if opts == nil {
		return th
	}
	if opts.Theme.NodeFill != "" {
		th.NodeFill = opts.Theme.NodeFill
	}
	if opts.Theme.NodeStroke != "" {
		th.NodeStroke = opts.Theme.NodeStroke
	}
	if opts.Theme.NodeText != "" {
		th.NodeText = opts.Theme.NodeText
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
