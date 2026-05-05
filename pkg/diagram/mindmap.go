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
	ID         string
	Text       string
	Shape      MindmapNodeShape
	Icon       string
	CSSClasses []string
	Children   []*MindmapNode
}

// MindmapStyleDef is a per-node inline style override parsed from a
// `style ID css` line. Mirrors the StateStyleDef / ERStyleDef shape
// used by the other diagram types.
type MindmapStyleDef struct {
	NodeID string
	CSS    string
}

type MindmapDiagram struct {
	Root       *MindmapNode
	AccTitle   string
	AccDescr   string
	CSSClasses map[string]string  // class name → CSS string from `classDef`
	Styles     []MindmapStyleDef  // per-node `style` override lines
}

func (*MindmapDiagram) Type() DiagramType { return Mindmap }

var _ Diagram = (*MindmapDiagram)(nil)
