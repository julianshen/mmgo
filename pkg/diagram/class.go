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
	ID         string
	Label      string
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

// ClassRelation describes one edge between two classes.
//
// RelationType encodes the *kind* of relationship (inheritance, composition,
// dependency, …) independently of how it was written in the source. The
// Reverse and Bidirectional flags preserve the *direction* the source used
// so the renderer can place glyphs on the correct end without losing
// information:
//
//   - "Animal <|-- Dog"   → Inheritance, Reverse=false  (parent on left)
//   - "Dog --|> Animal"   → Inheritance, Reverse=true   (parent on right)
//   - "A <|--|> B"        → Inheritance, Bidirectional=true
//
// Two-way arrows (`<|--|>`, `*--*`, `o--o`, `<-->`, `<..>`, `<|..|>`) set
// Bidirectional. A relation cannot be both Reverse and Bidirectional.
type ClassRelation struct {
	From            string
	To              string
	RelationType    RelationType
	Label           string
	FromCardinality string
	ToCardinality   string
	Reverse         bool
	Bidirectional   bool
}

type ClassDiagram struct {
	Classes   []ClassDef
	Relations []ClassRelation
	// Direction is the layout flow. DirectionUnknown means "use the
	// renderer's default" (currently top-to-bottom).
	Direction Direction
}

func (*ClassDiagram) Type() DiagramType { return Class }

var _ Diagram = (*ClassDiagram)(nil)
