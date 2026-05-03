package state

// Theme holds color surfaces the state renderer consumes. Mirrors
// pkg/renderer/class/theme.go for dispatcher uniformity. ChoiceFill
// covers the filled diamond for StateKindChoice; PseudoMark covers
// the start/end lollipop dots.
type Theme struct {
	StateFill    string
	StateStroke  string
	StateText    string
	ChoiceFill   string // StateKindChoice filled diamond border
	PseudoMark   string // start/end pseudo-state dot
	EdgeStroke   string
	EdgeText     string
	LabelBackdrop string // backdrop behind edge labels
	Background    string
	NoteFill      string // sticky-note background (yellow by default)
	NoteStroke    string
	NoteText      string
}

// DefaultTheme returns the Mermaid-classic state-diagram palette.
func DefaultTheme() Theme {
	return Theme{
		StateFill:     "#ECECFF",
		StateStroke:   "#9370DB",
		StateText:     "#333",
		ChoiceFill:    "#333",
		PseudoMark:    "#333",
		EdgeStroke:    "#333",
		EdgeText:      "#333",
		LabelBackdrop: "#fff",
		Background:    "#fff",
		NoteFill:      "#fff5ad",
		NoteStroke:    "#aaaa33",
		NoteText:      "#333",
	}
}

func resolveTheme(opts *Options) Theme {
	th := DefaultTheme()
	if opts == nil {
		return th
	}
	if opts.Theme.StateFill != "" {
		th.StateFill = opts.Theme.StateFill
	}
	if opts.Theme.StateStroke != "" {
		th.StateStroke = opts.Theme.StateStroke
	}
	if opts.Theme.StateText != "" {
		th.StateText = opts.Theme.StateText
	}
	if opts.Theme.ChoiceFill != "" {
		th.ChoiceFill = opts.Theme.ChoiceFill
	}
	if opts.Theme.PseudoMark != "" {
		th.PseudoMark = opts.Theme.PseudoMark
	}
	if opts.Theme.EdgeStroke != "" {
		th.EdgeStroke = opts.Theme.EdgeStroke
	}
	if opts.Theme.EdgeText != "" {
		th.EdgeText = opts.Theme.EdgeText
	}
	if opts.Theme.LabelBackdrop != "" {
		th.LabelBackdrop = opts.Theme.LabelBackdrop
	}
	if opts.Theme.Background != "" {
		th.Background = opts.Theme.Background
	}
	if opts.Theme.NoteFill != "" {
		th.NoteFill = opts.Theme.NoteFill
	}
	if opts.Theme.NoteStroke != "" {
		th.NoteStroke = opts.Theme.NoteStroke
	}
	if opts.Theme.NoteText != "" {
		th.NoteText = opts.Theme.NoteText
	}
	return th
}
