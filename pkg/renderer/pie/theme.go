package pie

import "github.com/julianshen/mmgo/pkg/renderer/svgutil"

// Theme holds pie-chart color surfaces. SliceColors cycles by slice
// order (and is mirrored into the legend swatches); InsideText is
// painted over each slice so it should contrast with every palette
// entry. TitleText and OutsideText cover the title and the leader-
// line labels for thin outside slices.
type Theme struct {
	SliceColors  []string
	TitleText    string
	InsideText   string
	OutsideText  string
	LeaderStroke string
	LegendText   string
	Background   string
}

func DefaultTheme() Theme {
	return Theme{
		SliceColors: []string{
			"#4e79a7", "#f28e2b", "#e15759", "#76b7b2",
			"#59a14f", "#edc948", "#b07aa1", "#ff9da7",
			"#9c755f", "#bab0ac",
		},
		TitleText:    "#333",
		InsideText:   "white",
		OutsideText:  "#333",
		LeaderStroke: "#666",
		LegendText:   "#333",
		Background:   "white",
	}
}

func resolveTheme(opts *Options) Theme {
	th := DefaultTheme()
	if opts == nil {
		return th
	}
	if len(opts.Theme.SliceColors) > 0 {
		th.SliceColors = opts.Theme.SliceColors
	}
	svgutil.MergeStr(&th.TitleText, opts.Theme.TitleText)
	svgutil.MergeStr(&th.InsideText, opts.Theme.InsideText)
	svgutil.MergeStr(&th.OutsideText, opts.Theme.OutsideText)
	svgutil.MergeStr(&th.LeaderStroke, opts.Theme.LeaderStroke)
	svgutil.MergeStr(&th.LegendText, opts.Theme.LegendText)
	svgutil.MergeStr(&th.Background, opts.Theme.Background)
	return th
}
