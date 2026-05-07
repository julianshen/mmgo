package gitgraph

import "github.com/julianshen/mmgo/pkg/renderer/svgutil"

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
	svgutil.MergeStr(&th.Text, opts.Theme.Text)
	svgutil.MergeStr(&th.BranchLabelText, opts.Theme.BranchLabelText)
	svgutil.MergeStr(&th.DotStrokeFill, opts.Theme.DotStrokeFill)
	svgutil.MergeStr(&th.LaneGuide, opts.Theme.LaneGuide)
	svgutil.MergeStr(&th.TagFill, opts.Theme.TagFill)
	svgutil.MergeStr(&th.TagText, opts.Theme.TagText)
	svgutil.MergeStr(&th.TagStroke, opts.Theme.TagStroke)
	svgutil.MergeStr(&th.Background, opts.Theme.Background)
	return th
}
