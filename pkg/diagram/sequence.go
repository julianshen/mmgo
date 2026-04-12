package diagram

// ParticipantKind distinguishes between a box participant and a stick-figure actor.
type ParticipantKind int8

const (
	ParticipantKindUnknown ParticipantKind = iota
	ParticipantKindParticipant
	ParticipantKindActor
)

var participantKindNames = []string{"unknown", "participant", "actor"}

func (p ParticipantKind) String() string { return enumString(p, participantKindNames) }

// Participant is a column in a sequence diagram.
type Participant struct {
	ID    string
	Alias string // display name; falls back to ID if empty
	Kind  ParticipantKind
}

// ArrowType describes the visual style of a sequence message arrow.
//
// Unlike flowchart edges (which decompose cleanly into orthogonal LineStyle
// and ArrowHead enums), sequence diagrams have a fixed set of 8 variants in
// Mermaid's spec. Modeling them as a single enum keeps parser logic simple.
type ArrowType int8

const (
	ArrowTypeUnknown      ArrowType = iota
	ArrowTypeSolid                  // ->>  solid line with filled arrowhead
	ArrowTypeSolidNoHead            // ->   solid line, no arrowhead
	ArrowTypeDashed                 // -->> dashed line with filled arrowhead
	ArrowTypeDashedNoHead           // -->  dashed line, no arrowhead
	ArrowTypeSolidCross             // -x   solid line with cross
	ArrowTypeDashedCross            // --x  dashed line with cross
	ArrowTypeSolidOpen              // -)   solid line with open (async) arrow
	ArrowTypeDashedOpen             // --)  dashed line with open (async) arrow
)

var arrowTypeNames = []string{
	"unknown",
	"solid",
	"solid-no-head",
	"dashed",
	"dashed-no-head",
	"solid-cross",
	"dashed-cross",
	"solid-open",
	"dashed-open",
}

func (a ArrowType) String() string { return enumString(a, arrowTypeNames) }

// Message is a single arrow/interaction in a sequence diagram.
type Message struct {
	From       string
	To         string
	Label      string
	ArrowType  ArrowType
	Activate   bool // + suffix: activate receiver lifeline
	Deactivate bool // - suffix: deactivate sender lifeline
}

// BlockKind identifies the structural block type.
type BlockKind int8

const (
	BlockKindUnknown  BlockKind = iota
	BlockKindAlt                // alt/else alternative branches
	BlockKindOpt                // opt conditional
	BlockKindLoop               // loop iteration
	BlockKindPar                // par parallel branches
	BlockKindCritical           // critical section
	BlockKindBreak              // break out of loop
	BlockKindRect               // rectangular visual grouping
)

var blockKindNames = []string{"unknown", "alt", "opt", "loop", "par", "critical", "break", "rect"}

func (b BlockKind) String() string { return enumString(b, blockKindNames) }

// Block is a nested structural element in a sequence diagram.
// Else holds alternative branches for alt/par/critical blocks.
type Block struct {
	Label    string
	Messages []Message
	Blocks   []Block // nested blocks
	Else     []Block // else/and/option branches
	Kind     BlockKind
}

// NotePosition describes where a note is drawn relative to participants.
type NotePosition int8

const (
	NotePositionUnknown NotePosition = iota
	NotePositionLeft
	NotePositionRight
	NotePositionOver
)

var notePositionNames = []string{"unknown", "left", "right", "over"}

func (n NotePosition) String() string { return enumString(n, notePositionNames) }

// Note is an annotation on one or more participants.
type Note struct {
	Participants []string // participant IDs this note is attached to
	Text         string
	Position     NotePosition
}

// SequenceDiagram is the AST for a Mermaid sequence diagram.
type SequenceDiagram struct {
	Participants []Participant
	Messages     []Message
	Blocks       []Block
	Notes        []Note
	AutoNumber   bool
}

// Type implements Diagram.
func (*SequenceDiagram) Type() DiagramType { return Sequence }

var _ Diagram = (*SequenceDiagram)(nil)
