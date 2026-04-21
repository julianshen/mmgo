package gantt

import "github.com/julianshen/mmgo/pkg/diagram"

// Theme holds gantt color surfaces. TaskColors maps task status to
// its bar fill; a missing entry falls back to the Default entry.
type Theme struct {
	TaskColors     map[diagram.TaskStatus]string
	TitleText      string
	SectionText    string
	AxisStroke     string
	AxisLabel      string
	InsideBarText  string
	OutsideBarText string
	Background     string
}

func DefaultTheme() Theme {
	return Theme{
		TaskColors: map[diagram.TaskStatus]string{
			diagram.TaskStatusDone:   "#9370DB",
			diagram.TaskStatusActive: "#4e79a7",
			diagram.TaskStatusCrit:   "#e15759",
			diagram.TaskStatusNone:   "#76b7b2",
		},
		TitleText:      "#333",
		SectionText:    "#333",
		AxisStroke:     "#ccc",
		AxisLabel:      "#666",
		InsideBarText:  "white",
		OutsideBarText: "#333",
		Background:     "#fff",
	}
}

// taskColor returns the fill color for s, falling back to the
// TaskStatusNone entry when s is missing from the map.
func (t Theme) taskColor(s diagram.TaskStatus) string {
	if c, ok := t.TaskColors[s]; ok && c != "" {
		return c
	}
	return t.TaskColors[diagram.TaskStatusNone]
}

func resolveTheme(opts *Options) Theme {
	th := DefaultTheme()
	if opts == nil {
		return th
	}
	if len(opts.Theme.TaskColors) > 0 {
		for k, v := range opts.Theme.TaskColors {
			if v != "" {
				th.TaskColors[k] = v
			}
		}
	}
	if opts.Theme.TitleText != "" {
		th.TitleText = opts.Theme.TitleText
	}
	if opts.Theme.SectionText != "" {
		th.SectionText = opts.Theme.SectionText
	}
	if opts.Theme.AxisStroke != "" {
		th.AxisStroke = opts.Theme.AxisStroke
	}
	if opts.Theme.AxisLabel != "" {
		th.AxisLabel = opts.Theme.AxisLabel
	}
	if opts.Theme.InsideBarText != "" {
		th.InsideBarText = opts.Theme.InsideBarText
	}
	if opts.Theme.OutsideBarText != "" {
		th.OutsideBarText = opts.Theme.OutsideBarText
	}
	if opts.Theme.Background != "" {
		th.Background = opts.Theme.Background
	}
	return th
}
