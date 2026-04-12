package diagram

// Direction is the layout direction of a flowchart.
type Direction int8

const (
	DirectionUnknown Direction = iota
	DirectionTB                // top to bottom
	DirectionBT                // bottom to top
	DirectionLR                // left to right
	DirectionRL                // right to left
)

var directionNames = []string{"unknown", "TB", "BT", "LR", "RL"}

// String returns the Mermaid direction keyword ("TB", "LR", ...).
func (d Direction) String() string { return enumString(d, directionNames) }

// NodeShape describes the visual shape of a flowchart node.
type NodeShape int8

const (
	NodeShapeUnknown NodeShape = iota
	NodeShapeRectangle
	NodeShapeRoundedRectangle
	NodeShapeStadium
	NodeShapeSubroutine
	NodeShapeCylinder
	NodeShapeCircle
	NodeShapeAsymmetric
	NodeShapeDiamond
	NodeShapeHexagon
	NodeShapeParallelogram
	NodeShapeParallelogramAlt
	NodeShapeTrapezoid
	NodeShapeTrapezoidAlt
	NodeShapeDoubleCircle
)

var nodeShapeNames = []string{
	"unknown",
	"rectangle",
	"rounded-rectangle",
	"stadium",
	"subroutine",
	"cylinder",
	"circle",
	"asymmetric",
	"diamond",
	"hexagon",
	"parallelogram",
	"parallelogram-alt",
	"trapezoid",
	"trapezoid-alt",
	"double-circle",
}

// String returns a debug slug for the shape (e.g. "rounded-rectangle").
// Note: this is a slug, not a Mermaid token — Mermaid has no textual
// representation for shapes; they are expressed by delimiters like [], (), {}.
func (s NodeShape) String() string { return enumString(s, nodeShapeNames) }

// LineStyle describes the stroke style of a flowchart edge.
type LineStyle int8

const (
	LineStyleUnknown LineStyle = iota
	LineStyleSolid
	LineStyleDotted
	LineStyleThick
)

var lineStyleNames = []string{"unknown", "solid", "dotted", "thick"}

func (ls LineStyle) String() string { return enumString(ls, lineStyleNames) }

// ArrowHead describes the head marker on a flowchart edge.
type ArrowHead int8

const (
	ArrowHeadUnknown ArrowHead = iota
	ArrowHeadNone
	ArrowHeadArrow
	ArrowHeadOpen
	ArrowHeadCross
	ArrowHeadCircle
)

var arrowHeadNames = []string{"unknown", "none", "arrow", "open", "cross", "circle"}

func (a ArrowHead) String() string { return enumString(a, arrowHeadNames) }

// Node is a single element in a flowchart.
type Node struct {
	ID    string
	Label string
	Class string // optional class name for styling
	Shape NodeShape
}

// Edge is a directed connection between two flowchart nodes.
type Edge struct {
	From      string
	To        string
	Label     string
	LineStyle LineStyle
	ArrowHead ArrowHead
}

// Subgraph is a named grouping of nodes. Subgraphs may nest.
type Subgraph struct {
	ID       string
	Label    string
	Nodes    []Node
	Edges    []Edge
	Children []Subgraph
}

// StyleDef is an inline style directive applied to a node.
type StyleDef struct {
	NodeID string
	// raw CSS declarations, e.g. "fill:#f9f,stroke:#333" — opaque and unvalidated.
	CSS string
}

// FlowchartDiagram is the AST for a Mermaid flowchart/graph diagram.
type FlowchartDiagram struct {
	Nodes     []Node
	Edges     []Edge
	Subgraphs []Subgraph
	Styles    []StyleDef
	Classes   map[string]string // class name -> raw CSS
	Direction Direction
}

// Type implements Diagram.
func (*FlowchartDiagram) Type() DiagramType { return Flowchart }

var _ Diagram = (*FlowchartDiagram)(nil)
