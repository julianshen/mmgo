package diagram

// QuadrantPointStyle holds optional per-point visual overrides
// parsed from inline `color: …, radius: …, stroke-width: …,
// stroke-color: …` style lists or from a referenced classDef.
//
// Zero values mean "use the theme default"; renderers should
// resolve a final style by overlaying inline > class > theme.
type QuadrantPointStyle struct {
	Color       string
	StrokeColor string
	Radius      float64
	StrokeWidth float64
}

// QuadrantPoint is a single plotted entry. X and Y are normalized to
// the [0, 1] square — 0 is the low end of the axis, 1 is the high end.
//
// Style holds inline visual overrides (parsed from
// `color:`/`radius:`/`stroke-width:`/`stroke-color:` after the
// coordinate list). Class is the optional `:::name` reference;
// when both are present, inline values win on collisions.
type QuadrantPoint struct {
	Label string
	X, Y  float64
	Style QuadrantPointStyle
	Class string
}

// QuadrantChartDiagram is the AST for a Mermaid quadrantChart.
//
// The quadrant strings follow Mermaid's math-convention numbering
// (not reading order): Q1 = top-right (high-x, high-y), Q2 = top-left
// (low-x, high-y), Q3 = bottom-left (low-x, low-y), Q4 = bottom-right
// (high-x, low-y).
//
// XAxisLow/XAxisHigh and YAxisLow/YAxisHigh are the text labels shown
// at the low and high ends of each axis. Mermaid uses `low --> high`
// syntax to declare them.
type QuadrantChartDiagram struct {
	Title      string
	AccTitle   string
	AccDescr   string
	XAxisLow   string
	XAxisHigh  string
	YAxisLow   string
	YAxisHigh  string
	Quadrant1  string
	Quadrant2  string
	Quadrant3  string
	Quadrant4  string
	Points     []QuadrantPoint
	// Classes is keyed by class name (declared via
	// `classDef name color: …`). Each entry holds the same
	// visual fields a point can override inline.
	Classes map[string]QuadrantPointStyle
}

func (*QuadrantChartDiagram) Type() DiagramType { return Quadrant }

var _ Diagram = (*QuadrantChartDiagram)(nil)
