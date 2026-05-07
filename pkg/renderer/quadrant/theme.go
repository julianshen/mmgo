package quadrant

// Theme mirrors the `themeVariables.quadrantChart.*` surface
// Mermaid documents at https://mermaid.js.org/syntax/quadrantChart.html.
// Every field maps 1:1 to a documented variable so callers can
// translate `%%{init: {themeVariables: {...}}}%%` blocks into a
// Theme without an intermediate lookup table.
type Theme struct {
	// Background and chrome colors.
	BackgroundColor string
	TitleColor      string

	// Per-quadrant fills. Empty values fall back to PlotFill so
	// authors can paint just one quadrant without rebuilding the
	// whole palette.
	Quadrant1Fill string
	Quadrant2Fill string
	Quadrant3Fill string
	Quadrant4Fill string

	// Per-quadrant text fills. Empty falls back to QuadrantTitleFill.
	Quadrant1TextFill string
	Quadrant2TextFill string
	Quadrant3TextFill string
	Quadrant4TextFill string

	// Quadrant label color (the text shown inside each quarter).
	QuadrantTitleFill string

	// Plot fill (the body color when no per-quadrant fill is
	// supplied). Distinct from BackgroundColor so the plot can
	// read against a different page background.
	PlotFill string

	// Internal divider stroke (the dashed midline) and external
	// border stroke (the plot's outer rect).
	QuadrantInternalBorderStrokeFill string
	QuadrantExternalBorderStrokeFill string

	// Axis label / tick fills.
	XAxisLabelColor string
	XAxisTitleColor string
	YAxisLabelColor string
	YAxisTitleColor string

	// Default point color when a point has no inline / class style.
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
		Quadrant1Fill:                    "",
		Quadrant2Fill:                    "",
		Quadrant3Fill:                    "",
		Quadrant4Fill:                    "",
		Quadrant1TextFill:                "",
		Quadrant2TextFill:                "",
		Quadrant3TextFill:                "",
		Quadrant4TextFill:                "",
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
	merge(&th.Quadrant1Fill, t.Quadrant1Fill)
	merge(&th.Quadrant2Fill, t.Quadrant2Fill)
	merge(&th.Quadrant3Fill, t.Quadrant3Fill)
	merge(&th.Quadrant4Fill, t.Quadrant4Fill)
	merge(&th.Quadrant1TextFill, t.Quadrant1TextFill)
	merge(&th.Quadrant2TextFill, t.Quadrant2TextFill)
	merge(&th.Quadrant3TextFill, t.Quadrant3TextFill)
	merge(&th.Quadrant4TextFill, t.Quadrant4TextFill)
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
