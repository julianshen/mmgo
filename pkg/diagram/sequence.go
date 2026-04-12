package diagram

// ParticipantKind distinguishes between a box participant and a stick-figure actor.
type ParticipantKind int

const (
	ParticipantKindUnknown ParticipantKind = iota
	ParticipantKindParticipant
	ParticipantKindActor
)

// String returns a human-readable name.
func (p ParticipantKind) String() string {
	switch p {
	case ParticipantKindParticipant:
		return "participant"
	case ParticipantKindActor:
		return "actor"
	default:
		return "unknown"
	}
}

// Participant is a column in a sequence diagram.
type Participant struct {
	ID    string
	Alias string // display name; falls back to ID if empty
	Kind  ParticipantKind
}

// ArrowType describes the visual style of a sequence message arrow.
// Mermaid supports 8 distinct arrow types.
type ArrowType int

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

// String returns a human-readable name.
func (a ArrowType) String() string {
	switch a {
	case ArrowTypeSolid:
		return "solid"
	case ArrowTypeSolidNoHead:
		return "solid-no-head"
	case ArrowTypeDashed:
		return "dashed"
	case ArrowTypeDashedNoHead:
		return "dashed-no-head"
	case ArrowTypeSolidCross:
		return "solid-cross"
	case ArrowTypeDashedCross:
		return "dashed-cross"
	case ArrowTypeSolidOpen:
		return "solid-open"
	case ArrowTypeDashedOpen:
		return "dashed-open"
	default:
		return "unknown"
	}
}

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
type BlockKind int

const (
	BlockKindUnknown BlockKind = iota
	BlockKindAlt              // alt/else alternative branches
	BlockKindOpt              // opt conditional
	BlockKindLoop             // loop iteration
	BlockKindPar              // par parallel branches
	BlockKindCritical         // critical section
	BlockKindBreak            // break out of loop
	BlockKindRect             // rectangular visual grouping
)

// String returns a human-readable name.
func (b BlockKind) String() string {
	switch b {
	case BlockKindAlt:
		return "alt"
	case BlockKindOpt:
		return "opt"
	case BlockKindLoop:
		return "loop"
	case BlockKindPar:
		return "par"
	case BlockKindCritical:
		return "critical"
	case BlockKindBreak:
		return "break"
	case BlockKindRect:
		return "rect"
	default:
		return "unknown"
	}
}

// Block is a nested structural element in a sequence diagram.
// Else holds alternative branches for alt/par/critical blocks.
type Block struct {
	Kind     BlockKind
	Label    string
	Messages []Message
	Blocks   []Block // nested blocks
	Else     []Block // else/and/option branches
}

// NotePosition describes where a note is drawn relative to participants.
type NotePosition int

const (
	NotePositionUnknown NotePosition = iota
	NotePositionLeft
	NotePositionRight
	NotePositionOver
)

// String returns a human-readable name.
func (n NotePosition) String() string {
	switch n {
	case NotePositionLeft:
		return "left"
	case NotePositionRight:
		return "right"
	case NotePositionOver:
		return "over"
	default:
		return "unknown"
	}
}

// Note is an annotation on one or more participants.
type Note struct {
	Position     NotePosition
	Participants []string // participant IDs this note is attached to
	Text         string
}

// SequenceDiagram is the AST for a Mermaid sequence diagram.
type SequenceDiagram struct {
	Participants []Participant
	Messages     []Message
	Blocks       []Block
	Notes        []Note
	AutoNumber   bool
}

// Type implements the Diagram interface.
func (*SequenceDiagram) Type() DiagramType { return Sequence }
