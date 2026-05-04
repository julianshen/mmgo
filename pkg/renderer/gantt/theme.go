package gantt

import "github.com/julianshen/mmgo/pkg/diagram"

// Theme holds gantt color surfaces. TaskColors maps task status to
// its bar fill; a missing entry falls back to the Default entry.
// SectionBands cycles per section in document order — mmdc tints the
// full row background so the eye can group related bars at a glance.
type Theme struct {
	TaskColors     map[diagram.TaskStatus]string
	TitleText      string
	SectionText    string
	AxisStroke     string
	AxisLabel      string
	GridStroke     string
	CritStroke     string   // outline drawn on top of crit bars
	SectionBands   []string // alternating row tints; len==0 disables banding
	InsideBarText  string
	OutsideBarText string
	Background     string
}

func DefaultTheme() Theme {
	return Theme{
		TaskColors: map[diagram.TaskStatus]string{
			// done → muted gray, matching mmdc's "completed" treatment.
			diagram.TaskStatusDone: "#bfc7d1",
			// active bar reads slightly lighter than the default accent
			// so an in-progress task stands out against plain bars.
			diagram.TaskStatusActive: "#8aa7cc",
			diagram.TaskStatusCrit:   "#e15759",
			diagram.TaskStatusNone:   "#8a8aca",
		},
		TitleText:      "#333",
		SectionText:    "#333",
		AxisStroke:     "#999",
		AxisLabel:      "#333",
		GridStroke:     "#d0d0d0",
		CritStroke:     "#9c2724",
		SectionBands:   []string{"#eaeaff", "#ffffff", "#fffbe6"},
		InsideBarText:  "white",
		OutsideBarText: "#333",
		Background:     "#fff",
	}
}

// taskColor maps a (possibly multi-flag) status bitmask to a fill
// color using a fixed priority order: Crit > Active > Done > None.
// Milestone does not enter the priority lookup — until PR2 adds a
// dedicated diamond glyph, a milestone task picks up whichever
// other flag it carries (commonly Crit). Missing or empty entries
// fall back to the TaskStatusNone color.
func (t Theme) taskColor(s diagram.TaskStatus) string {
	for _, flag := range []diagram.TaskStatus{
		diagram.TaskStatusCrit,
		diagram.TaskStatusActive,
		diagram.TaskStatusDone,
	} {
		if !s.Has(flag) {
			continue
		}
		if c, ok := t.TaskColors[flag]; ok && c != "" {
			return c
		}
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
	if opts.Theme.GridStroke != "" {
		th.GridStroke = opts.Theme.GridStroke
	}
	if opts.Theme.CritStroke != "" {
		th.CritStroke = opts.Theme.CritStroke
	}
	if len(opts.Theme.SectionBands) > 0 {
		th.SectionBands = opts.Theme.SectionBands
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
