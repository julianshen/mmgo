package xychart

// Theme mirrors every `themeVariables.xyChart.*` key documented at
// https://mermaid.js.org/syntax/xyChart.html. The aggregate fields
// (LabelFill, AxisStroke) rebroadcast to every per-surface field they
// cover when set, so callers that only know the aggregate keep the
// same behaviour they'd get if they set every narrow field by hand.
// Narrow-field overrides always win.
type Theme struct {
	SeriesColors []string

	LabelFill    string
	AxisStroke   string
	GridStroke   string
	MarkerStroke string
	Background   string

	TitleColor      string
	DataLabelColor  string
	XAxisLabelColor string
	XAxisTitleColor string
	XAxisTickColor  string
	XAxisLineColor  string
	YAxisLabelColor string
	YAxisTitleColor string
	YAxisTickColor  string
	YAxisLineColor  string
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

		TitleColor:      "#333",
		DataLabelColor:  "#333",
		XAxisLabelColor: "#333",
		XAxisTitleColor: "#333",
		XAxisTickColor:  "#999",
		XAxisLineColor:  "#999",
		YAxisLabelColor: "#333",
		YAxisTitleColor: "#333",
		YAxisTickColor:  "#999",
		YAxisLineColor:  "#999",
	}
}

func resolveTheme(opts *Options) Theme {
	th := DefaultTheme()
	if opts == nil {
		return th
	}
	merge := func(dst *string, src string) {
		if src != "" {
			*dst = src
		}
	}
	t := opts.Theme
	if len(t.SeriesColors) > 0 {
		th.SeriesColors = t.SeriesColors
	}
	merge(&th.LabelFill, t.LabelFill)
	merge(&th.AxisStroke, t.AxisStroke)
	merge(&th.GridStroke, t.GridStroke)
	merge(&th.MarkerStroke, t.MarkerStroke)
	merge(&th.Background, t.Background)

	if t.LabelFill != "" {
		th.TitleColor = t.LabelFill
		th.DataLabelColor = t.LabelFill
		th.XAxisLabelColor = t.LabelFill
		th.XAxisTitleColor = t.LabelFill
		th.YAxisLabelColor = t.LabelFill
		th.YAxisTitleColor = t.LabelFill
	}
	if t.AxisStroke != "" {
		th.XAxisTickColor = t.AxisStroke
		th.XAxisLineColor = t.AxisStroke
		th.YAxisTickColor = t.AxisStroke
		th.YAxisLineColor = t.AxisStroke
	}

	merge(&th.TitleColor, t.TitleColor)
	merge(&th.DataLabelColor, t.DataLabelColor)
	merge(&th.XAxisLabelColor, t.XAxisLabelColor)
	merge(&th.XAxisTitleColor, t.XAxisTitleColor)
	merge(&th.XAxisTickColor, t.XAxisTickColor)
	merge(&th.XAxisLineColor, t.XAxisLineColor)
	merge(&th.YAxisLabelColor, t.YAxisLabelColor)
	merge(&th.YAxisTitleColor, t.YAxisTitleColor)
	merge(&th.YAxisTickColor, t.YAxisTickColor)
	merge(&th.YAxisLineColor, t.YAxisLineColor)
	return th
}
