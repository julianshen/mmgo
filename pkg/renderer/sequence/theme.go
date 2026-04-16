package sequence

const (
	DefaultFontSize       = 14.0
	defaultPadding        = 20.0
	defaultParticipantGap = 150.0
	defaultRowHeight      = 50.0
	defaultBoxPadX        = 15.0
	defaultBoxPadY        = 10.0
	defaultBoxHeight      = 35.0
	defaultStrokeWidth    = 1.5
	defaultActorHeadR     = 12.0
	defaultActorBodyH     = 28.0
)

type Options struct {
	FontSize float64
	Padding  float64
	Theme    Theme
}

type Theme struct {
	Background        string
	ParticipantFill   string
	ParticipantStroke string
	ParticipantText   string
	LifelineStroke    string
	MessageStroke     string
	MessageText       string
	NoteFill          string
}

func DefaultTheme() Theme {
	return Theme{
		Background:        "white",
		ParticipantFill:   "#ECECFF",
		ParticipantStroke: "#9370DB",
		ParticipantText:   "#333",
		LifelineStroke:    "#999",
		MessageStroke:     "#333",
		MessageText:       "#333",
		NoteFill:          "#ffffcc",
	}
}

func resolveTheme(opts *Options) Theme {
	th := DefaultTheme()
	if opts == nil {
		return th
	}
	t := opts.Theme
	if t.Background != "" {
		th.Background = t.Background
	}
	if t.ParticipantFill != "" {
		th.ParticipantFill = t.ParticipantFill
	}
	if t.ParticipantStroke != "" {
		th.ParticipantStroke = t.ParticipantStroke
	}
	if t.ParticipantText != "" {
		th.ParticipantText = t.ParticipantText
	}
	if t.LifelineStroke != "" {
		th.LifelineStroke = t.LifelineStroke
	}
	if t.MessageStroke != "" {
		th.MessageStroke = t.MessageStroke
	}
	if t.MessageText != "" {
		th.MessageText = t.MessageText
	}
	if t.NoteFill != "" {
		th.NoteFill = t.NoteFill
	}
	return th
}

func resolveFontSize(opts *Options) float64 {
	if opts != nil && opts.FontSize > 0 {
		return opts.FontSize
	}
	return DefaultFontSize
}

func resolvePadding(opts *Options) float64 {
	if opts != nil && opts.Padding > 0 {
		return opts.Padding
	}
	return defaultPadding
}
