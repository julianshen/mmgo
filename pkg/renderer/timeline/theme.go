package timeline

import "github.com/julianshen/mmgo/pkg/renderer/svgutil"

// Theme holds color surfaces for the timeline renderer. SectionColors
// cycles per section (or per event when no sections are declared);
// it needs at least one entry. EventText is painted over a section
// color so it should contrast with every palette entry — white is
// safe for the classic palette.
type Theme struct {
	SectionColors []string
	TitleText     string
	SectionText   string
	EventText     string
	AxisStroke    string
	Background    string
}

func DefaultTheme() Theme {
	return Theme{
		SectionColors: []string{"#4e79a7", "#f28e2b", "#59a14f", "#e15759", "#76b7b2", "#edc948"},
		TitleText:     "#333",
		SectionText:   "#333",
		EventText:     "#fff",
		AxisStroke:    "#999",
		Background:    "#fff",
	}
}

func resolveTheme(opts *Options) Theme {
	th := DefaultTheme()
	if opts == nil {
		return th
	}
	if len(opts.Theme.SectionColors) > 0 {
		th.SectionColors = opts.Theme.SectionColors
	}
	svgutil.MergeStr(&th.TitleText, opts.Theme.TitleText)
	svgutil.MergeStr(&th.SectionText, opts.Theme.SectionText)
	svgutil.MergeStr(&th.EventText, opts.Theme.EventText)
	svgutil.MergeStr(&th.AxisStroke, opts.Theme.AxisStroke)
	svgutil.MergeStr(&th.Background, opts.Theme.Background)
	return th
}
