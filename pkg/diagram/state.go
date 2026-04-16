package diagram

type StateKind int8

const (
	StateKindNormal  StateKind = iota
	StateKindStart             // [*] as source
	StateKindEnd               // [*] as target
	StateKindFork              // <<fork>>
	StateKindJoin              // <<join>>
	StateKindChoice            // <<choice>>
)

var stateKindNames = []string{"normal", "start", "end", "fork", "join", "choice"}

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

type StateNote struct {
	StateID  string
	Text     string
	Position string
}

type StateDiagram struct {
	States      []StateDef
	Transitions []StateTransition
	Notes       []StateNote
}

func (*StateDiagram) Type() DiagramType { return State }

var _ Diagram = (*StateDiagram)(nil)
