package flowchart

import (
	"github.com/julianshen/mmgo/pkg/renderer/svgutil"
	"github.com/julianshen/mmgo/pkg/textmeasure"
)

const (
	DefaultFontSize = 16.0
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

// DefaultTheme returns the Mermaid-classic flowchart palette: light
// lavender fill with dark-purple stroke for nodes, matching mermaid-
// cli's "default" theme. The config-driven mapping in
// pkg/output/svg overrides these when an explicit theme is selected.
func DefaultTheme() Theme {
	return Theme{
		NodeFill:       "#ECECFF",
		NodeStroke:     "#9370DB",
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
	svgutil.MergeStr(&th.NodeFill, opts.Theme.NodeFill)
	svgutil.MergeStr(&th.NodeStroke, opts.Theme.NodeStroke)
	svgutil.MergeStr(&th.NodeText, opts.Theme.NodeText)
	svgutil.MergeStr(&th.EdgeStroke, opts.Theme.EdgeStroke)
	svgutil.MergeStr(&th.EdgeText, opts.Theme.EdgeText)
	svgutil.MergeStr(&th.SubgraphFill, opts.Theme.SubgraphFill)
	svgutil.MergeStr(&th.SubgraphStroke, opts.Theme.SubgraphStroke)
	svgutil.MergeStr(&th.SubgraphText, opts.Theme.SubgraphText)
	svgutil.MergeStr(&th.Background, opts.Theme.Background)
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
	return DefaultFontSize
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
