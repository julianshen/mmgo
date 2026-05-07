package diagram

// XYSeriesType distinguishes how a data series is rendered on the plot.
type XYSeriesType int8

const (
	XYSeriesBar XYSeriesType = iota
	XYSeriesLine
)

var xySeriesTypeNames = []string{"bar", "line"}

func (t XYSeriesType) String() string { return enumString(t, xySeriesTypeNames) }

// XYAxis describes an axis. Categories is non-nil for a discrete axis
// (typical for x in most charts); otherwise the axis is continuous
// bounded by [Min, Max]. If neither Categories nor explicit bounds are
// set the renderer derives Min/Max from the data.
type XYAxis struct {
	Title      string
	Categories []string
	Min, Max   float64
	HasRange   bool
}

// XYSeries is one data series plotted on the chart. Title is shown in
// a legend (when the renderer supports one); Data is the plotted values
// in X order.
type XYSeries struct {
	Type  XYSeriesType
	Title string
	Data  []float64
}

// XYChartDiagram is the AST for a Mermaid xychart-beta diagram.
//
// Horizontal is set by `xychart-beta horizontal`; the renderer swaps
// the axes' roles (categorical y-axis, value x-axis) when true.
type XYChartDiagram struct {
	Title      string
	AccTitle   string
	AccDescr   string
	Horizontal bool
	XAxis      XYAxis
	YAxis      XYAxis
	Series     []XYSeries
}

func (*XYChartDiagram) Type() DiagramType { return XYChart }

var _ Diagram = (*XYChartDiagram)(nil)
