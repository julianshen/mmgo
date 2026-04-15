package flowchart

import "github.com/julianshen/mmgo/pkg/textmeasure"

const (
	defaultFontSize = 16.0
	defaultPadding  = 20.0
)

type Options struct {
	FontSize   float64
	Padding    float64
	Theme      Theme
	ExtraCSS   string
	Background string
	// Ruler lets the caller share a pre-built text measurer so Render
	// doesn't re-parse the bundled TTF on every call. When nil, Render
	// creates one and closes it internally.
	Ruler *textmeasure.Ruler
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

func rulerFromOpts(opts *Options) *textmeasure.Ruler {
	if opts == nil {
		return nil
	}
	return opts.Ruler
}

func resolvePadding(opts *Options) float64 {
	if opts != nil && opts.Padding > 0 {
		return opts.Padding
	}
	return defaultPadding
}
