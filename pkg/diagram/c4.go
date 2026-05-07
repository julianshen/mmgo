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
}

// C4BoundaryKind discriminates among the documented boundary
// container keywords. Each draws as a dashed outer rectangle with
// a `<<kind>>` stereotype label; layout-aware renderers can use
// the kind to pick a fill / heading style.
type C4BoundaryKind int8

const (
	C4BoundaryGeneric C4BoundaryKind = iota
	C4BoundarySystem
	C4BoundaryEnterprise
	C4BoundaryContainer
)

var c4BoundaryKindNames = []string{"boundary", "system_boundary", "enterprise_boundary", "container_boundary"}

func (k C4BoundaryKind) String() string { return enumString(k, c4BoundaryKindNames) }

// C4Boundary is a `Boundary(...) { ... }` container. Children
// indexes into the parent diagram's flat Elements slice (or, for
// nested groups, into Boundaries). Tags / Type are reserved for
// Phase 3 named-arg parsing.
type C4Boundary struct {
	ID         string
	Label      string
	Type       string
	Kind       C4BoundaryKind
	Tags       string
	Link       string
	Elements   []int        // indexes into C4Diagram.Elements
	Boundaries []*C4Boundary
}

type C4Diagram struct {
	Variant   C4Variant
	Title     string
	AccTitle  string
	AccDescr  string
	Elements  []C4Element
	Relations []C4Relation
	// Boundaries are top-level boundary blocks parsed from the
	// source. Nested boundaries live in each parent's Boundaries
	// slice; an element appearing inside a boundary is added to
	// the flat Elements list (so existing renderer-side ID
	// lookups keep working) AND has its index appended to the
	// surrounding boundary's Elements slice.
	Boundaries []*C4Boundary
}

func (*C4Diagram) Type() DiagramType { return C4 }

var _ Diagram = (*C4Diagram)(nil)
