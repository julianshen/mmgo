package diagram

type BlockShape int8

const (
	BlockShapeRect BlockShape = iota
	BlockShapeRound
	BlockShapeDiamond
	BlockShapeStadium
	BlockShapeCircle
	BlockShapeHexagon
	BlockShapeSubroutine
	BlockShapeDoubleCircle
	BlockShapeCylinder
)

var blockShapeNames = []string{"rect", "round", "diamond", "stadium", "circle", "hexagon", "subroutine", "doubleCircle", "cylinder"}

func (s BlockShape) String() string { return enumString(s, blockShapeNames) }

type BlockNode struct {
	ID    string
	Label string
	Shape BlockShape
}

type BlockEdge struct {
	From  string
	To    string
	Label string
}

type BlockDiagram struct {
	Columns  int
	AccTitle string
	AccDescr string
	Nodes    []BlockNode
	Edges    []BlockEdge
}

func (*BlockDiagram) Type() DiagramType { return BlockDiag }

var _ Diagram = (*BlockDiagram)(nil)
