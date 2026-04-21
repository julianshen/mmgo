package mindmap

// Theme holds the color surfaces the mindmap renderer consumes.
// LevelColors is a cycling palette keyed by tree depth; the node
// fill at depth N is LevelColors[N % len]. NodeText is painted over
// the level color, so it should contrast against the whole palette
// (white works for all Mermaid-classic level colors).
type Theme struct {
	LevelColors []string
	NodeText    string
	EdgeStroke  string
	Background  string
}

func DefaultTheme() Theme {
	return Theme{
		LevelColors: []string{"#4e79a7", "#f28e2b", "#e15759", "#76b7b2", "#59a14f", "#edc948"},
		NodeText:    "#fff",
		EdgeStroke:  "#999",
		Background:  "#fff",
	}
}

func resolveTheme(opts *Options) Theme {
	th := DefaultTheme()
	if opts == nil {
		return th
	}
	if len(opts.Theme.LevelColors) > 0 {
		th.LevelColors = opts.Theme.LevelColors
	}
	if opts.Theme.NodeText != "" {
		th.NodeText = opts.Theme.NodeText
	}
	if opts.Theme.EdgeStroke != "" {
		th.EdgeStroke = opts.Theme.EdgeStroke
	}
	if opts.Theme.Background != "" {
		th.Background = opts.Theme.Background
	}
	return th
}
