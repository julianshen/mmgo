package sequence

import "github.com/julianshen/mmgo/pkg/renderer/svgutil"

const (
	DefaultFontSize       = 14.0
	defaultPadding        = 20.0
	defaultParticipantGap = 150.0
	defaultRowHeight      = 50.0
	defaultBoxPadX        = 15.0
	defaultBoxPadY        = 10.0
	defaultBoxHeight      = 35.0
	defaultStrokeWidth    = 1.5
	defaultLifelineWidth  = 2.0
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
		LifelineStroke:    "#9370DB",
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
	svgutil.MergeStr(&th.Background, t.Background)
	svgutil.MergeStr(&th.ParticipantFill, t.ParticipantFill)
	svgutil.MergeStr(&th.ParticipantStroke, t.ParticipantStroke)
	svgutil.MergeStr(&th.ParticipantText, t.ParticipantText)
	svgutil.MergeStr(&th.LifelineStroke, t.LifelineStroke)
	svgutil.MergeStr(&th.MessageStroke, t.MessageStroke)
	svgutil.MergeStr(&th.MessageText, t.MessageText)
	svgutil.MergeStr(&th.NoteFill, t.NoteFill)
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
