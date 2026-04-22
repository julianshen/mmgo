package diagram

type MindmapNodeShape int8

const (
	MindmapShapeDefault MindmapNodeShape = iota
	MindmapShapeRound
	MindmapShapeSquare
	MindmapShapeCircle
	MindmapShapeCloud
	MindmapShapeBang
	MindmapShapeHexagon
)

var mindmapShapeNames = []string{"default", "round", "square", "circle", "cloud", "bang", "hexagon"}

func (s MindmapNodeShape) String() string { return enumString(s, mindmapShapeNames) }

type MindmapNode struct {
	ID       string
	Text     string
	Shape    MindmapNodeShape
	Icon     string
	Class    string
	Children []*MindmapNode
}

type MindmapDiagram struct {
	Root *MindmapNode
}

func (*MindmapDiagram) Type() DiagramType { return Mindmap }

var _ Diagram = (*MindmapDiagram)(nil)
