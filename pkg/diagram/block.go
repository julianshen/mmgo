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
	// BlockShapeAsymmetric is the `>label]` "odd" / flag shape.
	BlockShapeAsymmetric
	// BlockShapeParallelogram is `[/label/]`.
	BlockShapeParallelogram
	// BlockShapeParallelogramAlt is `[\label\]` (mirrored slant).
	BlockShapeParallelogramAlt
	// BlockShapeTrapezoid is `[/label\]`.
	BlockShapeTrapezoid
	// BlockShapeTrapezoidAlt is `[\label/]` (inverted trapezoid).
	BlockShapeTrapezoidAlt
)

var blockShapeNames = []string{
	"rect", "round", "diamond", "stadium", "circle",
	"hexagon", "subroutine", "doubleCircle", "cylinder",
	"asymmetric", "parallelogram", "parallelogramAlt", "trapezoid", "trapezoidAlt",
}

func (s BlockShape) String() string { return enumString(s, blockShapeNames) }

type BlockNode struct {
	ID    string
	Label string
	Shape BlockShape
	// Width is the number of grid columns this node spans, parsed
	// from the trailing `:N` suffix (`id:3`). Zero defaults to 1
	// at layout time; renderers should treat 0 as "single column".
	Width int
	// CSSClasses is the list of class names this node has been
	// associated with via `class id name` lines or the `id:::name`
	// shorthand. Each entry should resolve to a key in
	// BlockDiagram.CSSClasses.
	CSSClasses []string
}

// BlockStyleDef is a per-node inline style override parsed from a
// `style ID css` line. Mirrors the shape used by class / state /
// ER / mindmap diagram parsers.
type BlockStyleDef struct {
	NodeID string
	CSS    string
}

type BlockEdge struct {
	From  string
	To    string
	Label string
	// LineStyle covers the stroke pattern: solid (`-->`/`---`),
	// thick (`==>`), dotted (`-.->`), invisible (`~~~`).
	LineStyle LineStyle
	// ArrowHead is the marker at the To end. `---` → None.
	ArrowHead ArrowHead
	// ArrowTail is the marker at the From end (only set for the
	// bidirectional `<-->` form; otherwise None).
	ArrowTail ArrowHead
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
	Title    string
	AccTitle string
	AccDescr string
	Nodes    []BlockNode
	Edges    []BlockEdge
	// CSSClasses maps a class name (declared via `classDef name css`)
	// to its CSS string.
	CSSClasses map[string]string
	// Styles holds per-node `style id css` overrides in declaration
	// order so later rules override earlier ones at render time.
	Styles []BlockStyleDef
	// Items preserves the structural ordering parsed from the
	// source: top-level node references, spacers, and groups in
	// source order. The flat Nodes slice retains every node
	// (including those nested inside groups) for renderer-side
	// id lookup; layout-aware renderers should walk Items.
	Items []BlockItem
}

func (*BlockDiagram) Type() DiagramType { return BlockDiag }

var _ Diagram = (*BlockDiagram)(nil)
