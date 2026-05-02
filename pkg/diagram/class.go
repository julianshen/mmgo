package diagram

type Visibility int8

const (
	VisibilityNone    Visibility = iota
	VisibilityPublic              // +
	VisibilityPrivate             // -
	VisibilityProtected           // #
	VisibilityPackage             // ~
)

var visibilityNames = []string{"none", "public", "private", "protected", "package"}

func (v Visibility) String() string { return enumString(v, visibilityNames) }

// ClassMember holds one entry inside a class body.
//
// For methods (IsMethod=true), Name/Args/ReturnType are parsed apart so
// the renderer can emit the canonical `name(args) : returnType` form
// regardless of which Mermaid syntax variant the source used.
//
// For fields, Name carries the *raw* post-visibility text (e.g. the full
// "String name" or "name: String"), Args is unused, ReturnType is empty.
// The renderer prints fields verbatim — splitting on whitespace would
// silently invert "type name" vs "name type" orderings, and splitting
// on `:` would mangle TypeScript-style declarations.
type ClassMember struct {
	Name       string
	Args       string
	ReturnType string
	Visibility Visibility
	IsMethod   bool
	// IsStatic / IsAbstract are UML modifiers parsed from trailing
	// `$` and `*` markers respectively. Renderers conventionally
	// underline static members and italicize abstract ones.
	IsStatic   bool
	IsAbstract bool
}

type ClassAnnotation int8

const (
	AnnotationNone      ClassAnnotation = iota
	AnnotationInterface
	AnnotationAbstract
	AnnotationService
	AnnotationEnum
)

var classAnnotationNames = []string{"none", "interface", "abstract", "service", "enum"}

func (a ClassAnnotation) String() string { return enumString(a, classAnnotationNames) }

type ClassDef struct {
	ID    string
	Label string
	// Generic carries the parametric-type list parsed from
	// `class Name~T~` or `class Map~K, V~`. Empty when the class
	// has no generic. The angle brackets are not stored.
	Generic    string
	Members    []ClassMember
	Annotation ClassAnnotation
}

type RelationType int8

const (
	RelationTypeUnknown     RelationType = iota
	RelationTypeInheritance               // <|--
	RelationTypeComposition               // *--
	RelationTypeAggregation               // o--
	RelationTypeAssociation               // -->
	RelationTypeDependency                // ..>
	RelationTypeRealization               // ..|>
	RelationTypeLink                      // --
	RelationTypeDashedLink                // ..
)

var relationTypeNames = []string{
	"unknown", "inheritance", "composition", "aggregation",
	"association", "dependency", "realization", "link", "dashed-link",
}

func (r RelationType) String() string { return enumString(r, relationTypeNames) }

// RelationDirection captures whether the source wrote the arrow in
// canonical direction, reversed, or as a two-way (bidirectional) form.
// Encoded as a single enum so the illegal combination "reverse +
// bidirectional" is unrepresentable.
type RelationDirection int8

const (
	// RelationForward matches Mermaid's canonical literal — `<|--`,
	// `*--`, `o--`, `-->`, `..>`, `..|>`, `--`, `..`.
	RelationForward RelationDirection = iota
	// RelationReverse mirrors the canonical literal — `--|>`, `--*`,
	// `--o`, `<--`, `<..`, `<|..`. The relation kind is unchanged;
	// only the glyph-bearing end swaps.
	RelationReverse
	// RelationBidirectional is the two-way form — `<|--|>`, `*--*`,
	// `o--o`, `<-->`, `<..>`, `<|..|>`. Same glyph at both ends.
	RelationBidirectional
)

// ClassRelation describes one edge between two classes. RelationType
// encodes the kind of relationship; Direction encodes how the arrow
// was written so the renderer can place glyphs on the correct end(s).
type ClassRelation struct {
	From            string
	To              string
	RelationType    RelationType
	Label           string
	FromCardinality string
	ToCardinality   string
	Direction       RelationDirection
}

// ClassNote is a free-floating annotation on the diagram. When For
// names a class ID, the renderer anchors the note next to that class
// with a thin connector. When For is empty, the note is general and
// floats beside the diagram.
//
// Mermaid supports line breaks inside note text via the literal `\n`
// sequence in the source; the parser stores the text with `\n`
// converted to a real newline so renderers can split on it directly.
type ClassNote struct {
	Text string
	For  string // class ID, or "" for a general note
}

type ClassDiagram struct {
	Classes   []ClassDef
	Relations []ClassRelation
	Notes     []ClassNote
	// Direction is the layout flow. DirectionUnknown means "use the
	// renderer's default" (currently top-to-bottom).
	Direction Direction
}

func (*ClassDiagram) Type() DiagramType { return Class }

var _ Diagram = (*ClassDiagram)(nil)
