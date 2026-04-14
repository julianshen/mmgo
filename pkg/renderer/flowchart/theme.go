package flowchart

const (
	defaultFontSize = 16.0
	defaultPadding  = 20.0
)

type Options struct {
	FontSize   float64
	Padding    float64
	Theme      Theme
	CSSFile    string
	Background string
}

type Theme struct {
	NodeFill       string
	NodeStroke     string
	NodeText       string
	EdgeStroke     string
	EdgeText       string
	SubgraphFill   string
	SubgraphStroke string
	SubgraphText   string
	Background     string
}

func DefaultTheme() Theme {
	return Theme{
		NodeFill:       "#fff",
		NodeStroke:     "#333",
		NodeText:       "#333",
		EdgeStroke:     "#333",
		EdgeText:       "#333",
		SubgraphFill:   "#eee",
		SubgraphStroke: "#999",
		SubgraphText:   "#333",
		Background:     "#fff",
	}
}

func resolveTheme(opts *Options) Theme {
	th := DefaultTheme()
	if opts == nil {
		return th
	}
	if opts.Theme.NodeFill != "" {
		th.NodeFill = opts.Theme.NodeFill
	}
	if opts.Theme.NodeStroke != "" {
		th.NodeStroke = opts.Theme.NodeStroke
	}
	if opts.Theme.NodeText != "" {
		th.NodeText = opts.Theme.NodeText
	}
	if opts.Theme.EdgeStroke != "" {
		th.EdgeStroke = opts.Theme.EdgeStroke
	}
	if opts.Theme.EdgeText != "" {
		th.EdgeText = opts.Theme.EdgeText
	}
	if opts.Theme.SubgraphFill != "" {
		th.SubgraphFill = opts.Theme.SubgraphFill
	}
	if opts.Theme.SubgraphStroke != "" {
		th.SubgraphStroke = opts.Theme.SubgraphStroke
	}
	if opts.Theme.SubgraphText != "" {
		th.SubgraphText = opts.Theme.SubgraphText
	}
	if opts.Theme.Background != "" {
		th.Background = opts.Theme.Background
	}
	return th
}

func resolveBackground(opts *Options, th Theme) string {
	if opts != nil && opts.Background != "" {
		return opts.Background
	}
	return th.Background
}

func resolveFontSize(opts *Options) float64 {
	if opts != nil && opts.FontSize > 0 {
		return opts.FontSize
	}
	return defaultFontSize
}

func resolvePadding(opts *Options) float64 {
	if opts != nil && opts.Padding > 0 {
		return opts.Padding
	}
	return defaultPadding
}
