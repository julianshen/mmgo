package state

import "github.com/julianshen/mmgo/pkg/renderer/svgutil"

// Theme holds color surfaces the state renderer consumes. Mirrors
// pkg/renderer/class/theme.go for dispatcher uniformity. ChoiceFill
// covers the filled diamond for StateKindChoice; PseudoMark covers
// the start/end lollipop dots.
type Theme struct {
	StateFill       string
	StateStroke     string
	StateText       string
	ChoiceFill      string // StateKindChoice filled diamond border
	PseudoMark      string // start/end pseudo-state dot
	EdgeStroke      string
	EdgeText        string
	LabelBackdrop   string // backdrop behind edge labels
	Background      string
	NoteFill        string // sticky-note background (yellow by default)
	NoteStroke      string
	NoteText        string
	CompositeFill   string // composite-state container background
	CompositeStroke string
	CompositeText   string
}

// DefaultTheme returns the Mermaid-classic state-diagram palette.
func DefaultTheme() Theme {
	return Theme{
		StateFill:       "#ECECFF",
		StateStroke:     "#9370DB",
		StateText:       "#333",
		ChoiceFill:      "#333",
		PseudoMark:      "#333",
		EdgeStroke:      "#333",
		EdgeText:        "#333",
		LabelBackdrop:   "#fff",
		Background:      "#fff",
		NoteFill:        "#fff5ad",
		NoteStroke:      "#aaaa33",
		NoteText:        "#333",
		CompositeFill:   "#f7f7ff",
		CompositeStroke: "#9370DB",
		CompositeText:   "#555",
	}
}

func resolveTheme(opts *Options) Theme {
	th := DefaultTheme()
	if opts == nil {
		return th
	}
	svgutil.MergeStr(&th.StateFill, opts.Theme.StateFill)
	svgutil.MergeStr(&th.StateStroke, opts.Theme.StateStroke)
	svgutil.MergeStr(&th.StateText, opts.Theme.StateText)
	svgutil.MergeStr(&th.ChoiceFill, opts.Theme.ChoiceFill)
	svgutil.MergeStr(&th.PseudoMark, opts.Theme.PseudoMark)
	svgutil.MergeStr(&th.EdgeStroke, opts.Theme.EdgeStroke)
	svgutil.MergeStr(&th.EdgeText, opts.Theme.EdgeText)
	svgutil.MergeStr(&th.LabelBackdrop, opts.Theme.LabelBackdrop)
	svgutil.MergeStr(&th.Background, opts.Theme.Background)
	svgutil.MergeStr(&th.NoteFill, opts.Theme.NoteFill)
	svgutil.MergeStr(&th.NoteStroke, opts.Theme.NoteStroke)
	svgutil.MergeStr(&th.NoteText, opts.Theme.NoteText)
	svgutil.MergeStr(&th.CompositeFill, opts.Theme.CompositeFill)
	svgutil.MergeStr(&th.CompositeStroke, opts.Theme.CompositeStroke)
	svgutil.MergeStr(&th.CompositeText, opts.Theme.CompositeText)
	return th
}
