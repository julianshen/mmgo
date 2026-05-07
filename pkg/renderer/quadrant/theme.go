package quadrant

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
	merge := func(dst *string, src string) {
		if src != "" {
			*dst = src
		}
	}
	t := opts.Theme
	merge(&th.BackgroundColor, t.BackgroundColor)
	merge(&th.TitleColor, t.TitleColor)
	for i := range th.Quadrants {
		merge(&th.Quadrants[i].Fill, t.Quadrants[i].Fill)
		merge(&th.Quadrants[i].TextFill, t.Quadrants[i].TextFill)
	}
	merge(&th.QuadrantTitleFill, t.QuadrantTitleFill)
	merge(&th.PlotFill, t.PlotFill)
	merge(&th.QuadrantInternalBorderStrokeFill, t.QuadrantInternalBorderStrokeFill)
	merge(&th.QuadrantExternalBorderStrokeFill, t.QuadrantExternalBorderStrokeFill)
	merge(&th.XAxisLabelColor, t.XAxisLabelColor)
	merge(&th.XAxisTitleColor, t.XAxisTitleColor)
	merge(&th.YAxisLabelColor, t.YAxisLabelColor)
	merge(&th.YAxisTitleColor, t.YAxisTitleColor)
	merge(&th.QuadrantPointFill, t.QuadrantPointFill)
	merge(&th.QuadrantPointTextFill, t.QuadrantPointTextFill)
	merge(&th.QuadrantPointStroke, t.QuadrantPointStroke)
	return th
}
