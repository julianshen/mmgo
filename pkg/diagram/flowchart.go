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

// String returns a stable debug slug for the shape (e.g. "rounded-rectangle").
// It is intentionally not a Mermaid token: Mermaid expresses shapes through
// multiple surface forms (delimiters like [], (), {} and the @{shape: ...}
// extended syntax), and this enum is deliberately decoupled from any one form.
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
	// Classes is the list of class names referenced by this node (e.g. via
	// Mermaid's `class nodeId classA,classB`). Each name should be a key
	// in FlowchartDiagram.Classes.
	Classes []string
	Shape   NodeShape
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
	// CSS holds raw CSS declarations (e.g. "fill:#f9f,stroke:#333").
	// The AST stores this opaquely; validation is the renderer's concern.
	CSS string
}

// FlowchartDiagram is the AST for a Mermaid flowchart/graph diagram.
//
// Node ownership: a node appearing inside a Subgraph is stored only in that
// Subgraph's Nodes slice, not also at the top level. Top-level Nodes is for
// nodes outside any subgraph. To iterate all nodes, walk subgraphs recursively.
// Edges may cross subgraph boundaries; an edge lives in the scope where it
// is declared in the source.
type FlowchartDiagram struct {
	Nodes     []Node
	Edges     []Edge
	Subgraphs []Subgraph
	Styles    []StyleDef
	// Classes maps class name -> raw CSS declarations, referenced by
	// Node.Classes. May be nil if no classDef directives were parsed.
	Classes   map[string]string
	Direction Direction
}

// Type implements Diagram.
func (*FlowchartDiagram) Type() DiagramType { return Flowchart }

var _ Diagram = (*FlowchartDiagram)(nil)
