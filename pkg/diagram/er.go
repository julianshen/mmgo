package diagram

type ERAttributeKey int8

const (
	ERKeyNone ERAttributeKey = iota
	ERKeyPK
	ERKeyFK
	ERKeyUK
)

var erKeyNames = []string{"", "PK", "FK", "UK"}

func (k ERAttributeKey) String() string { return enumString(k, erKeyNames) }

type ERAttribute struct {
	Type string
	Name string
	// Key mirrors Keys[0] when Keys is non-empty; ERKeyNone when
	// the attribute has no constraints. Multi-key consumers should
	// walk Keys directly.
	Key     ERAttributeKey
	Keys    []ERAttributeKey
	Comment string
}

type EREntity struct {
	Name string
	// Label is the optional display string parsed from
	// `EntityID["Display Label"]`. Renderers use Label when
	// non-empty, falling back to Name. Relations and bindings
	// continue to reference the entity by Name (the bare ID).
	Label      string
	Attributes []ERAttribute
	// CSSClasses are user-defined CSS class names attached via
	// `class A,B foo` or the inline `A:::foo` shorthand.
	CSSClasses []string
}

type ERCardinality int8

const (
	ERCardUnknown      ERCardinality = iota
	ERCardZeroOrOne                  // |o or o|
	ERCardExactlyOne                 // ||
	ERCardZeroOrMore                 // }o or o{
	ERCardOneOrMore                  // }| or |{
)

var erCardNames = []string{"unknown", "zero-or-one", "exactly-one", "zero-or-more", "one-or-more"}

func (c ERCardinality) String() string { return enumString(c, erCardNames) }

type ERRelationship struct {
	From     string
	To       string
	FromCard ERCardinality
	ToCard   ERCardinality
	Label    string
}

// ERStyleDef is a per-entity inline style override parsed from
// `style EntityID fill:#f9f,stroke:#333`.
type ERStyleDef struct {
	EntityID string
	CSS      string
}

// ERClickDef binds a click action to an entity. Either URL or
// Callback is set; the renderer emits a hyperlink for URL forms.
type ERClickDef struct {
	EntityID string
	URL      string
	Tooltip  string
	Target   string
	Callback string
}

type ERDiagram struct {
	Entities      []EREntity
	Relationships []ERRelationship
	// CSSClasses maps a user-defined class name (from `classDef foo …`)
	// to its semicolon-separated CSS declarations.
	CSSClasses map[string]string
	// Styles are per-entity inline overrides from `style ID …` lines.
	Styles []ERStyleDef
	// Clicks are click / link / callback bindings.
	Clicks []ERClickDef
	// Direction is the layout flow. DirectionUnknown means "use
	// the renderer's default" (currently top-to-bottom).
	Direction Direction
	// Title and accessibility metadata.
	Title    string
	AccTitle string
	AccDescr string
}

func (*ERDiagram) Type() DiagramType { return ER }

var _ Diagram = (*ERDiagram)(nil)
