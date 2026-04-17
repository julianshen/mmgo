package diagram

type MindmapNodeShape int8

const (
	MindmapShapeDefault MindmapNodeShape = iota
	MindmapShapeRound                    // (text)
	MindmapShapeSquare                   // [text]
	MindmapShapeCloud                    // ))text((
	MindmapShapeBang                     // {{text}}
	MindmapShapeHexagon                  // {{text}}
)

var mindmapShapeNames = []string{"default", "round", "square", "cloud", "bang", "hexagon"}

func (s MindmapNodeShape) String() string { return enumString(s, mindmapShapeNames) }

type MindmapNode struct {
	Text     string
	Shape    MindmapNodeShape
	Children []*MindmapNode
}

type MindmapDiagram struct {
	Root *MindmapNode
}

func (*MindmapDiagram) Type() DiagramType { return Mindmap }

var _ Diagram = (*MindmapDiagram)(nil)
