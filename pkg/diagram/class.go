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

type ClassMember struct {
	Name       string
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

type ClassRelation struct {
	From          string
	To            string
	RelationType  RelationType
	Label         string
	FromCardinality string
	ToCardinality   string
}

type ClassDiagram struct {
	Classes   []ClassDef
	Relations []ClassRelation
}

func (*ClassDiagram) Type() DiagramType { return Class }

var _ Diagram = (*ClassDiagram)(nil)
