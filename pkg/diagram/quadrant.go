package diagram

// QuadrantPoint is a single plotted entry. X and Y are normalized to
// the [0, 1] square — 0 is the low end of the axis, 1 is the high end.
type QuadrantPoint struct {
	Label string
	X, Y  float64
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
	XAxisLow   string
	XAxisHigh  string
	YAxisLow   string
	YAxisHigh  string
	Quadrant1  string
	Quadrant2  string
	Quadrant3  string
	Quadrant4  string
	Points     []QuadrantPoint
}

func (*QuadrantChartDiagram) Type() DiagramType { return Quadrant }

var _ Diagram = (*QuadrantChartDiagram)(nil)
