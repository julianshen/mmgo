package mindmap

type Theme struct {
	SectionColors []string
	RootColor     string
	NodeText      string
	RootText      string
	EdgeStroke    string
	Background    string
}

func DefaultTheme() Theme {
	return Theme{
		SectionColors: []string{"#f28e2b", "#e15759", "#76b7b2", "#59a14f", "#edc948", "#b07aa1"},
		RootColor:     "#4e79a7",
		NodeText:      "#fff",
		RootText:      "#fff",
		EdgeStroke:    "#999",
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
	if opts.Theme.RootColor != "" {
		th.RootColor = opts.Theme.RootColor
	}
	if opts.Theme.NodeText != "" {
		th.NodeText = opts.Theme.NodeText
	}
	if opts.Theme.RootText != "" {
		th.RootText = opts.Theme.RootText
	}
	if opts.Theme.EdgeStroke != "" {
		th.EdgeStroke = opts.Theme.EdgeStroke
	}
	if opts.Theme.Background != "" {
		th.Background = opts.Theme.Background
	}
	return th
}
