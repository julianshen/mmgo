package sankey

import "github.com/julianshen/mmgo/pkg/renderer/svgutil"

// Theme holds sankey color surfaces. NodeColors cycles by first-
// appearance index; the color of a ribbon matches its source node.
type Theme struct {
	NodeColors []string
	LabelText  string
	Background string
}

func DefaultTheme() Theme {
	return Theme{
		NodeColors: []string{
			"#5470c6", "#91cc75", "#fac858", "#ee6666",
			"#73c0de", "#3ba272", "#fc8452", "#9a60b4",
		},
		LabelText:  "#333",
		Background: "#fff",
	}
}

func resolveTheme(opts *Options) Theme {
	th := DefaultTheme()
	if opts == nil {
		return th
	}
	if len(opts.Theme.NodeColors) > 0 {
		th.NodeColors = opts.Theme.NodeColors
	}
	svgutil.MergeStr(&th.LabelText, opts.Theme.LabelText)
	svgutil.MergeStr(&th.Background, opts.Theme.Background)
	return th
}
