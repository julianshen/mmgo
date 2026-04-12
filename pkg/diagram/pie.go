package diagram

// Slice is a single wedge in a pie chart.
type Slice struct {
	Label string
	Value float64
}

// PieDiagram is the AST for a Mermaid pie chart.
type PieDiagram struct {
	Title    string
	Slices   []Slice
	ShowData bool // whether to render raw values alongside labels
}

// Type implements the Diagram interface.
func (*PieDiagram) Type() DiagramType { return Pie }
