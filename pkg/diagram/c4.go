package diagram

type C4ElementKind int8

const (
	C4ElementUnknown C4ElementKind = iota
	C4ElementPerson
	C4ElementPersonExt
	C4ElementSystem
	C4ElementSystemExt
	C4ElementSystemDB
	C4ElementSystemDBExt
	C4ElementSystemQueue
	C4ElementSystemQueueExt
	C4ElementContainer
	C4ElementContainerExt
	C4ElementContainerDB
	C4ElementContainerDBExt
	C4ElementContainerQueue
	C4ElementContainerQueueExt
	C4ElementComponent
	C4ElementComponentExt
	C4ElementComponentDB
	C4ElementComponentDBExt
	C4ElementComponentQueue
	C4ElementComponentQueueExt
	C4ElementBoundary
	C4ElementDeploymentNode
)

var c4ElementNames = []string{
	"unknown",
	"person", "person_ext",
	"system", "system_ext", "system_db", "system_db_ext", "system_queue", "system_queue_ext",
	"container", "container_ext", "container_db", "container_db_ext", "container_queue", "container_queue_ext",
	"component", "component_ext", "component_db", "component_db_ext", "component_queue", "component_queue_ext",
	"boundary", "deployment_node",
}

func (k C4ElementKind) String() string { return enumString(k, c4ElementNames) }

type C4Variant int8

const (
	C4VariantContext C4Variant = iota
	C4VariantContainer
	C4VariantComponent
	C4VariantDynamic
	C4VariantDeployment
)

var c4VariantNames = []string{"context", "container", "component", "dynamic", "deployment"}

func (v C4Variant) String() string { return enumString(v, c4VariantNames) }

type C4Element struct {
	ID          string
	Kind        C4ElementKind
	Label       string
	Technology  string
	Description string
	// Tags is a comma-separated list of stereotype names from the
	// `$tags=` named arg. Captured for downstream consumers; Mermaid
	// itself does not paint anything from it.
	Tags string
	// Link is the URL from the `$link=` named arg.
	Link string
	// Sprite is the icon name from the `$sprite=` named arg.
	// Captured for downstream consumers; not painted today.
	Sprite string
}

type C4RelDirection int8

const (
	C4RelDefault C4RelDirection = iota
	C4RelUp
	C4RelDown
	C4RelLeft
	C4RelRight
	C4RelBack
	C4RelBi
)

var c4RelDirNames = []string{"default", "up", "down", "left", "right", "back", "bi"}

func (d C4RelDirection) String() string { return enumString(d, c4RelDirNames) }

type C4Relation struct {
	From       string
	To         string
	Label      string
	Technology string
	Direction  C4RelDirection
	// Tags / Link / Sprite mirror the named-arg surface on elements.
	Tags   string
	Link   string
	Sprite string
	// OffsetX / OffsetY shift the label from the default mid-curve
	// anchor. Useful for nudging crowded labels off neighbouring
	// lines.
	OffsetX float64
	OffsetY float64
	// Index is the sequence number from `RelIndex(N, from, to, …)`.
	// Zero means "no index" — relations parsed via plain Rel(...)
	// don't get one. Renderers draw it as a small circled number
	// near the source endpoint of the curve.
	Index int
}

// C4BoundaryKind discriminates among the documented boundary
// container keywords. Renderers consume it to pick the
// stereotype label and any kind-specific styling.
type C4BoundaryKind int8

const (
	C4BoundaryGeneric C4BoundaryKind = iota
	C4BoundarySystem
	C4BoundaryEnterprise
	C4BoundaryContainer
)

var c4BoundaryKindNames = []string{"boundary", "system_boundary", "enterprise_boundary", "container_boundary"}

func (k C4BoundaryKind) String() string { return enumString(k, c4BoundaryKindNames) }

// C4Boundary is a `Boundary(...) { ... }` container. Elements
// indexes into the parent diagram's flat Elements slice; nested
// groups live in Boundaries.
//
// TypeHint stores the optional positional 3rd arg
// (`Boundary(b, "Label", "system")`) — Mermaid uses it to
// override the rendered stereotype on a generic Boundary.
type C4Boundary struct {
	ID         string
	Label      string
	TypeHint   string
	Kind       C4BoundaryKind
	Elements   []int // indexes into C4Diagram.Elements
	Boundaries []*C4Boundary
	// Tags / Link / Sprite mirror the named-arg surface. The
	// renderer wraps the boundary frame in `<a href>` when Link is
	// set; Tags and Sprite are captured but unrendered.
	Tags   string
	Link   string
	Sprite string
}

// C4LayoutDirection mirrors the LAYOUT_TOP_DOWN /
// LAYOUT_LEFT_RIGHT directives that override the variant's default
// flow. Empty value means "inherit the variant default".
type C4LayoutDirection string

const (
	C4LayoutTopDown   C4LayoutDirection = "TB"
	C4LayoutLeftRight C4LayoutDirection = "LR"
)

// C4ElementStyleOverride captures the per-kind theme override from
// `UpdateElementStyle(kind, $bgColor=…, $fontColor=…, $borderColor=…)`.
// Empty fields fall through to the resolved theme palette.
type C4ElementStyleOverride struct {
	BgColor     string
	FontColor   string
	BorderColor string
}

// C4RelStyleOverride captures `UpdateRelStyle(from, to, $textColor=…,
// $lineColor=…, $offsetX=…, $offsetY=…)`. Empty / zero fields fall
// through.
type C4RelStyleOverride struct {
	TextColor string
	LineColor string
	OffsetX   float64
	OffsetY   float64
}

// C4LayoutConfig captures `UpdateLayoutConfig($c4ShapeInRow=N,
// $c4BoundaryInRow=M)`. Mermaid uses these to lay out elements in a
// grid of the specified row width. Zero means "no override".
//
// The parser populates the values; the layout engine doesn't yet
// translate them into a row-bounded grid (still uses dagre's
// natural rank assignment). Captured for downstream consumers and
// to keep the directive recognised at parse time.
type C4LayoutConfig struct {
	ShapesInRow     int
	BoundariesInRow int
}

type C4Diagram struct {
	Variant   C4Variant
	Title     string
	AccTitle  string
	AccDescr  string
	Direction C4LayoutDirection
	Elements  []C4Element
	Relations []C4Relation
	// Boundaries are top-level boundary blocks parsed from the
	// source. Nested boundaries live in each parent's Boundaries
	// slice; an element appearing inside a boundary is added to
	// the flat Elements list (so existing renderer-side ID
	// lookups keep working) AND has its index appended to the
	// surrounding boundary's Elements slice.
	Boundaries []*C4Boundary
	// ElementStyles maps a C4ElementKind to its per-kind override.
	// Indexed by Kind.String() so source authors don't have to
	// match a typed enum.
	ElementStyles map[string]C4ElementStyleOverride
	// RelStyles maps "from->to" to a per-edge override. Multiple
	// `UpdateRelStyle` calls on the same pair last-wins.
	RelStyles map[string]C4RelStyleOverride
	// LayoutConfig captures UpdateLayoutConfig knobs. Renderers
	// honour them when sizing and arranging the element grid.
	LayoutConfig C4LayoutConfig
	// ShowLegend, set by the SHOW_LEGEND() directive, asks the
	// renderer to emit a key listing every distinct element kind
	// appearing in the diagram with its theme color.
	ShowLegend bool
}

func (*C4Diagram) Type() DiagramType { return C4 }

var _ Diagram = (*C4Diagram)(nil)
