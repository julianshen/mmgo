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
	Type    string
	Name    string
	Key     ERAttributeKey
	Comment string
}

type EREntity struct {
	Name       string
	Attributes []ERAttribute
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

type ERDiagram struct {
	Entities      []EREntity
	Relationships []ERRelationship
}

func (*ERDiagram) Type() DiagramType { return ER }

var _ Diagram = (*ERDiagram)(nil)
