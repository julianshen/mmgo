package diagram

// Direction is the layout direction of a flowchart.
type Direction int

const (
	DirectionUnknown Direction = iota
	DirectionTB                // top to bottom
	DirectionBT                // bottom to top
	DirectionLR                // left to right
	DirectionRL                // right to left
)

// String returns the Mermaid keyword for this direction.
func (d Direction) String() string {
	switch d {
	case DirectionTB:
		return "TB"
	case DirectionBT:
		return "BT"
	case DirectionLR:
		return "LR"
	case DirectionRL:
		return "RL"
	default:
		return "unknown"
	}
}

// NodeShape describes the visual shape of a flowchart node.
type NodeShape int

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

// String returns a human-readable name for this shape.
func (s NodeShape) String() string {
	switch s {
	case NodeShapeRectangle:
		return "rectangle"
	case NodeShapeRoundedRectangle:
		return "rounded-rectangle"
	case NodeShapeStadium:
		return "stadium"
	case NodeShapeSubroutine:
		return "subroutine"
	case NodeShapeCylinder:
		return "cylinder"
	case NodeShapeCircle:
		return "circle"
	case NodeShapeAsymmetric:
		return "asymmetric"
	case NodeShapeDiamond:
		return "diamond"
	case NodeShapeHexagon:
		return "hexagon"
	case NodeShapeParallelogram:
		return "parallelogram"
	case NodeShapeParallelogramAlt:
		return "parallelogram-alt"
	case NodeShapeTrapezoid:
		return "trapezoid"
	case NodeShapeTrapezoidAlt:
		return "trapezoid-alt"
	case NodeShapeDoubleCircle:
		return "double-circle"
	default:
		return "unknown"
	}
}

// LineStyle describes the stroke style of a flowchart edge.
type LineStyle int

const (
	LineStyleUnknown LineStyle = iota
	LineStyleSolid
	LineStyleDotted
	LineStyleThick
)

// String returns a human-readable name.
func (ls LineStyle) String() string {
	switch ls {
	case LineStyleSolid:
		return "solid"
	case LineStyleDotted:
		return "dotted"
	case LineStyleThick:
		return "thick"
	default:
		return "unknown"
	}
}

// ArrowHead describes the head marker on a flowchart edge.
type ArrowHead int

const (
	ArrowHeadUnknown ArrowHead = iota
	ArrowHeadNone
	ArrowHeadArrow
	ArrowHeadOpen
	ArrowHeadCross
	ArrowHeadCircle
)

// String returns a human-readable name.
func (a ArrowHead) String() string {
	switch a {
	case ArrowHeadNone:
		return "none"
	case ArrowHeadArrow:
		return "arrow"
	case ArrowHeadOpen:
		return "open"
	case ArrowHeadCross:
		return "cross"
	case ArrowHeadCircle:
		return "circle"
	default:
		return "unknown"
	}
}

// Node is a single element in a flowchart.
type Node struct {
	ID    string
	Label string
	Shape NodeShape
	Class string // optional class name for styling
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
	CSS    string // raw CSS declarations, e.g. "fill:#f9f,stroke:#333"
}

// FlowchartDiagram is the AST for a Mermaid flowchart/graph diagram.
type FlowchartDiagram struct {
	Direction Direction
	Nodes     []Node
	Edges     []Edge
	Subgraphs []Subgraph
	Styles    []StyleDef
	Classes   map[string]string // class name -> CSS
}

// Type implements the Diagram interface.
func (*FlowchartDiagram) Type() DiagramType { return Flowchart }
