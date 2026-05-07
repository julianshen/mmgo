package block

import "github.com/julianshen/mmgo/pkg/renderer/svgutil"

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
	svgutil.MergeStr(&th.NodeFill, opts.Theme.NodeFill)
	svgutil.MergeStr(&th.NodeStroke, opts.Theme.NodeStroke)
	svgutil.MergeStr(&th.NodeText, opts.Theme.NodeText)
	svgutil.MergeStr(&th.EdgeStroke, opts.Theme.EdgeStroke)
	svgutil.MergeStr(&th.EdgeText, opts.Theme.EdgeText)
	svgutil.MergeStr(&th.Background, opts.Theme.Background)
	return th
}
