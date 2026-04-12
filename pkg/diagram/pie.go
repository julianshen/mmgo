package diagram

// Slice is a single wedge in a pie chart.
type Slice struct {
	Label string
	Value float64
}

// PieDiagram is the AST for a Mermaid pie chart. Pie is deliberately minimal —
// it has no enums or nested structure to model.
type PieDiagram struct {
	Title    string
	Slices   []Slice
	ShowData bool // whether to render raw values alongside labels
}

// Type implements Diagram.
func (*PieDiagram) Type() DiagramType { return Pie }

var _ Diagram = (*PieDiagram)(nil)
