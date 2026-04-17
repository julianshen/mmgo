package diagram

type StateKind int8

const (
	StateKindNormal StateKind = iota
	StateKindFork             // <<fork>>
	StateKindJoin             // <<join>>
	StateKindChoice           // <<choice>>
)

var stateKindNames = []string{"normal", "fork", "join", "choice"}

func (s StateKind) String() string { return enumString(s, stateKindNames) }

type StateDef struct {
	ID       string
	Label    string
	Kind     StateKind
	Children []StateDef
}

type StateTransition struct {
	From  string
	To    string
	Label string
}

type StateDiagram struct {
	States      []StateDef
	Transitions []StateTransition
}

func (*StateDiagram) Type() DiagramType { return State }

var _ Diagram = (*StateDiagram)(nil)
