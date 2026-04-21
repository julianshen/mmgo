package pie

// Theme holds pie-chart color surfaces. SliceColors cycles by slice
// order (and is mirrored into the legend swatches); InsideText is
// painted over each slice so it should contrast with every palette
// entry. TitleText and OutsideText cover the title and the leader-
// line labels for thin outside slices.
type Theme struct {
	SliceColors []string
	TitleText   string
	InsideText  string
	OutsideText string
	LeaderStroke string
	LegendText  string
	Background  string
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
	if opts.Theme.TitleText != "" {
		th.TitleText = opts.Theme.TitleText
	}
	if opts.Theme.InsideText != "" {
		th.InsideText = opts.Theme.InsideText
	}
	if opts.Theme.OutsideText != "" {
		th.OutsideText = opts.Theme.OutsideText
	}
	if opts.Theme.LeaderStroke != "" {
		th.LeaderStroke = opts.Theme.LeaderStroke
	}
	if opts.Theme.LegendText != "" {
		th.LegendText = opts.Theme.LegendText
	}
	if opts.Theme.Background != "" {
		th.Background = opts.Theme.Background
	}
	return th
}
