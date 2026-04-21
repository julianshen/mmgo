package xychart

// Theme holds xychart color surfaces. SeriesColors cycles by series
// index so adjacent series stay visually distinct.
type Theme struct {
	SeriesColors []string
	LabelFill    string
	AxisStroke   string
	GridStroke   string
	MarkerStroke string // outer edge of line-chart dot markers
	Background   string
}

func DefaultTheme() Theme {
	return Theme{
		SeriesColors: []string{
			"#5470c6", "#91cc75", "#fac858", "#ee6666",
			"#73c0de", "#3ba272", "#fc8452", "#9a60b4",
		},
		LabelFill:    "#333",
		AxisStroke:   "#999",
		GridStroke:   "#e5e5e5",
		MarkerStroke: "#fff",
		Background:   "#fff",
	}
}

func resolveTheme(opts *Options) Theme {
	th := DefaultTheme()
	if opts == nil {
		return th
	}
	if len(opts.Theme.SeriesColors) > 0 {
		th.SeriesColors = opts.Theme.SeriesColors
	}
	if opts.Theme.LabelFill != "" {
		th.LabelFill = opts.Theme.LabelFill
	}
	if opts.Theme.AxisStroke != "" {
		th.AxisStroke = opts.Theme.AxisStroke
	}
	if opts.Theme.GridStroke != "" {
		th.GridStroke = opts.Theme.GridStroke
	}
	if opts.Theme.MarkerStroke != "" {
		th.MarkerStroke = opts.Theme.MarkerStroke
	}
	if opts.Theme.Background != "" {
		th.Background = opts.Theme.Background
	}
	return th
}
