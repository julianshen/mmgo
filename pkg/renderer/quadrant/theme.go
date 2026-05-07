package quadrant

import "github.com/julianshen/mmgo/pkg/renderer/svgutil"

// QuadrantPalette holds the per-quadrant colour pair (body fill +
// label text fill). Empty fields fall back to Theme.PlotFill /
// Theme.QuadrantTitleFill respectively.
type QuadrantPalette struct {
	Fill     string
	TextFill string
}

// Theme mirrors the `themeVariables.quadrantChart.*` surface
// Mermaid documents at https://mermaid.js.org/syntax/quadrantChart.html.
// Every field maps 1:1 to a documented variable. The four quadrant
// palettes live in a fixed-size array indexed in math-convention
// order (0=Q1 top-right, 1=Q2 top-left, 2=Q3 bottom-left,
// 3=Q4 bottom-right) so the JSON `quadrant1Fill` / `quadrant2Fill`
// keys decode into Quadrants[0].Fill / Quadrants[1].Fill.
type Theme struct {
	BackgroundColor string
	TitleColor      string

	Quadrants [4]QuadrantPalette

	QuadrantTitleFill string
	PlotFill          string

	QuadrantInternalBorderStrokeFill string
	QuadrantExternalBorderStrokeFill string

	XAxisLabelColor string
	XAxisTitleColor string
	YAxisLabelColor string
	YAxisTitleColor string

	QuadrantPointFill     string
	QuadrantPointTextFill string
	QuadrantPointStroke   string
}

// DefaultTheme returns the Mermaid-default palette. Every field
// is non-empty so a caller can override a subset and inherit the
// rest via resolveTheme.
func DefaultTheme() Theme {
	return Theme{
		BackgroundColor:                  "#fff",
		TitleColor:                       "#333",
		QuadrantTitleFill:                "#555",
		PlotFill:                         "#f7f7fa",
		QuadrantInternalBorderStrokeFill: "#bbb",
		QuadrantExternalBorderStrokeFill: "#888",
		XAxisLabelColor:                  "#333",
		XAxisTitleColor:                  "#333",
		YAxisLabelColor:                  "#333",
		YAxisTitleColor:                  "#333",
		QuadrantPointFill:                "#5470c6",
		QuadrantPointTextFill:            "#222",
		QuadrantPointStroke:              "#fff",
	}
}

// resolveTheme overlays opts.Theme on top of DefaultTheme(),
// keeping any zero (empty) field at the default. Mirrors the
// "merge non-empty fields" pattern other diagram types use.
func resolveTheme(opts *Options) Theme {
	th := DefaultTheme()
	if opts == nil {
		return th
	}
	t := opts.Theme
	svgutil.MergeStr(&th.BackgroundColor, t.BackgroundColor)
	svgutil.MergeStr(&th.TitleColor, t.TitleColor)
	for i := range th.Quadrants {
		svgutil.MergeStr(&th.Quadrants[i].Fill, t.Quadrants[i].Fill)
		svgutil.MergeStr(&th.Quadrants[i].TextFill, t.Quadrants[i].TextFill)
	}
	svgutil.MergeStr(&th.QuadrantTitleFill, t.QuadrantTitleFill)
	svgutil.MergeStr(&th.PlotFill, t.PlotFill)
	svgutil.MergeStr(&th.QuadrantInternalBorderStrokeFill, t.QuadrantInternalBorderStrokeFill)
	svgutil.MergeStr(&th.QuadrantExternalBorderStrokeFill, t.QuadrantExternalBorderStrokeFill)
	svgutil.MergeStr(&th.XAxisLabelColor, t.XAxisLabelColor)
	svgutil.MergeStr(&th.XAxisTitleColor, t.XAxisTitleColor)
	svgutil.MergeStr(&th.YAxisLabelColor, t.YAxisLabelColor)
	svgutil.MergeStr(&th.YAxisTitleColor, t.YAxisTitleColor)
	svgutil.MergeStr(&th.QuadrantPointFill, t.QuadrantPointFill)
	svgutil.MergeStr(&th.QuadrantPointTextFill, t.QuadrantPointTextFill)
	svgutil.MergeStr(&th.QuadrantPointStroke, t.QuadrantPointStroke)
	return th
}
