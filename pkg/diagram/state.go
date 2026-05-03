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
	ID    string
	Label string
	// Description is optional secondary text parsed from
	// `id : description` syntax. Renderers may show it below the
	// state title in a separate compartment.
	Description string
	Kind        StateKind
	Children    []StateDef
}

type StateTransition struct {
	From  string
	To    string
	Label string
}

// NoteSide is which side of the target state a note is anchored on.
type NoteSide int8

const (
	NoteSideUnspecified NoteSide = iota
	NoteSideLeft
	NoteSideRight
)

// StateNote is a free-floating annotation anchored to a state. The
// Side controls which edge (left or right) of the target the note
// sits beside; the renderer draws a dashed connector from the state
// to the note. Multi-line text uses real `\n` (the parser collapses
// the literal `\n` sequence and the multi-line block form into the
// same representation).
type StateNote struct {
	Text   string
	Side   NoteSide
	Target string // state ID the note attaches to
}

type StateDiagram struct {
	States      []StateDef
	Transitions []StateTransition
	Notes       []StateNote
	// Direction is the layout flow. DirectionUnknown means "use the
	// renderer's default" (currently top-to-bottom).
	Direction Direction
}

func (*StateDiagram) Type() DiagramType { return State }

var _ Diagram = (*StateDiagram)(nil)
