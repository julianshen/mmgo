package gitgraph

// Theme holds gitgraph color surfaces. BranchColors cycles by branch
// declaration order; lane 0 (main) picks the first entry.
type Theme struct {
	BranchColors  []string
	Text          string
	DotStrokeFill string // outer ring fill for highlight dots
	Background    string
}

func DefaultTheme() Theme {
	return Theme{
		BranchColors: []string{
			"#0f62fe", "#24a148", "#f1c21b",
			"#8a3ffc", "#ff7eb6", "#6fdc8c",
		},
		Text:          "#333",
		DotStrokeFill: "#fff",
		Background:    "#fff",
	}
}

func resolveTheme(opts *Options) Theme {
	th := DefaultTheme()
	if opts == nil {
		return th
	}
	if len(opts.Theme.BranchColors) > 0 {
		th.BranchColors = opts.Theme.BranchColors
	}
	if opts.Theme.Text != "" {
		th.Text = opts.Theme.Text
	}
	if opts.Theme.DotStrokeFill != "" {
		th.DotStrokeFill = opts.Theme.DotStrokeFill
	}
	if opts.Theme.Background != "" {
		th.Background = opts.Theme.Background
	}
	return th
}
