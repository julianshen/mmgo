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
	// Width is the number of grid columns this node spans, parsed
	// from the trailing `:N` suffix (`id:3`). Zero defaults to 1
	// at layout time; renderers should treat 0 as "single column".
	Width int
}

type BlockEdge struct {
	From  string
	To    string
	Label string
}

// BlockItemKind discriminates the BlockItem union. Items are the
// structural form preserved by the parser — the flat Nodes / Edges
// slices on BlockDiagram remain the data the (current) renderer
// consumes, while Items holds the original layout intent
// (groups, spacers, ordering) for layout-aware renderers.
type BlockItemKind int8

const (
	BlockItemNodeRef BlockItemKind = iota
	BlockItemSpace
	BlockItemGroup
)

// BlockItem is a single child of a row (or of a group). Exactly
// one of NodeID / Cols / Group is meaningful, dictated by Kind.
type BlockItem struct {
	Kind   BlockItemKind
	NodeID string      // Kind == BlockItemNodeRef
	Cols   int         // Kind == BlockItemSpace; columns the spacer occupies (default 1)
	Group  *BlockGroup // Kind == BlockItemGroup
}

// BlockGroup is a `block:ID[:N]["label"] ... end` container. Width
// is the number of grid columns the group itself occupies in its
// parent (zero → 1). Columns is the number of columns the group's
// own internal layout uses (zero → inherit from parent context).
type BlockGroup struct {
	ID      string
	Label   string
	Width   int
	Columns int
	Items   []BlockItem
}

type BlockDiagram struct {
	Columns  int
	AccTitle string
	AccDescr string
	Nodes    []BlockNode
	Edges    []BlockEdge
	// Items preserves the structural ordering parsed from the
	// source: top-level node references, spacers, and groups in
	// source order. The flat Nodes slice retains every node
	// (including those nested inside groups) for renderer-side
	// id lookup; layout-aware renderers should walk Items.
	Items []BlockItem
}

func (*BlockDiagram) Type() DiagramType { return BlockDiag }

var _ Diagram = (*BlockDiagram)(nil)
