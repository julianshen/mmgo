package gitgraph

// Theme holds gitgraph color surfaces. BranchColors cycles by branch
// declaration order; lane 0 (main) picks the first entry.
type Theme struct {
	BranchColors    []string
	Text            string
	BranchLabelText string // text color inside the colored branch pill
	DotStrokeFill   string // outer ring fill for highlight dots
	LaneGuide       string // dashed swimlane baseline
	TagFill         string // rounded tag callout background
	TagText         string // tag callout text color
	TagStroke       string // tag callout border
	Background      string
}

func DefaultTheme() Theme {
	return Theme{
		BranchColors: []string{
			"#0f62fe", "#24a148", "#f1c21b",
			"#8a3ffc", "#ff7eb6", "#6fdc8c",
		},
		Text:            "#333",
		BranchLabelText: "#fff",
		DotStrokeFill:   "#fff",
		LaneGuide:       "#bbb",
		TagFill:         "#eeeeee",
		TagText:         "#333",
		TagStroke:       "#bbb",
		Background:      "#fff",
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
	if opts.Theme.BranchLabelText != "" {
		th.BranchLabelText = opts.Theme.BranchLabelText
	}
	if opts.Theme.DotStrokeFill != "" {
		th.DotStrokeFill = opts.Theme.DotStrokeFill
	}
	if opts.Theme.LaneGuide != "" {
		th.LaneGuide = opts.Theme.LaneGuide
	}
	if opts.Theme.TagFill != "" {
		th.TagFill = opts.Theme.TagFill
	}
	if opts.Theme.TagText != "" {
		th.TagText = opts.Theme.TagText
	}
	if opts.Theme.TagStroke != "" {
		th.TagStroke = opts.Theme.TagStroke
	}
	if opts.Theme.Background != "" {
		th.Background = opts.Theme.Background
	}
	return th
}
