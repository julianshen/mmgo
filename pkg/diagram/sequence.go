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
type Box struct {
	Label    string
	Fill     string
	HasAlpha bool
	Members  []string
}

type Participant struct {
	ID              string
	Alias           string
	Kind            ParticipantKind
	BoxIndex        int
	CreatedAtItem   int
	DestroyedAtItem int
}

func (p Participant) Label() string {
	if p.Alias != "" {
		return p.Alias
	}
	return p.ID
}

// ArrowType describes the visual style of a sequence message arrow.
//
// Unlike flowchart edges (which decompose cleanly into orthogonal LineStyle
// and ArrowHead enums), Mermaid sequence diagrams currently expose 8 core
// arrow variants. Modeling them as a single enum keeps parser and renderer
// logic simple. If Mermaid adds more variants in the future, extend this
// enum rather than splitting it.
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
	ArrowTypeSolidBi                // <<->>  solid bidirectional
	ArrowTypeDashedBi               // <<-->> dashed bidirectional
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
	"solid-bi",
	"dashed-bi",
}

func (a ArrowType) String() string { return enumString(a, arrowTypeNames) }

func (a ArrowType) HasArrowHead() bool {
	return a != ArrowTypeSolidNoHead && a != ArrowTypeDashedNoHead
}

func (a ArrowType) IsBidirectional() bool {
	return a == ArrowTypeSolidBi || a == ArrowTypeDashedBi
}

func (a ArrowType) IsDashed() bool {
	switch a {
	case ArrowTypeDashed, ArrowTypeDashedNoHead,
		ArrowTypeDashedCross, ArrowTypeDashedOpen,
		ArrowTypeDashedBi:
		return true
	default:
		return false
	}
}

func (a ArrowType) MarkerRef() string {
	if a.HasArrowHead() && !a.IsBidirectional() {
		return "url(#seq-arrow-" + a.String() + ")"
	}
	return ""
}

// LifelineEffect describes the effect of a message on the receiver's lifeline
// activation bar. Modeled as a single enum (rather than two bools) so that
// contradictory states are unrepresentable.
type LifelineEffect int8

const (
	LifelineEffectNone       LifelineEffect = iota // no change
	LifelineEffectActivate                         // + suffix: activate receiver
	LifelineEffectDeactivate                       // - suffix: deactivate sender
)

var lifelineEffectNames = []string{"none", "activate", "deactivate"}

func (l LifelineEffect) String() string { return enumString(l, lifelineEffectNames) }

// Message is a single arrow/interaction in a sequence diagram.
type Message struct {
	From      string
	To        string
	Label     string
	ArrowType ArrowType
	Lifeline  LifelineEffect
}

// SequenceItem is one source-ordered element in a sequence diagram body.
// Exactly one of Message, Block, or Note is populated. Use the Kind() method
// to dispatch.
//
// A tagged struct is used instead of an interface because the types are a
// closed set, and a value-typed container avoids interface allocation
// overhead during parser/renderer traversal.
type SequenceItem struct {
	Message    *Message
	Block      *Block
	Note       *Note
	Destroy    *string
	Activation *Activation
}

// Activation represents a standalone `activate <id>` or `deactivate <id>`
// statement. The inline `+`/`-` arrow shorthand is represented on Message.Lifeline
// instead; this struct exists for the source-ordered standalone form.
type Activation struct {
	Participant string
	Activate    bool
}

// NewMessageItem wraps a Message as a SequenceItem.
func NewMessageItem(m Message) SequenceItem { return SequenceItem{Message: &m} }

// NewBlockItem wraps a Block as a SequenceItem.
func NewBlockItem(b Block) SequenceItem { return SequenceItem{Block: &b} }

// NewNoteItem wraps a Note as a SequenceItem.
func NewNoteItem(n Note) SequenceItem { return SequenceItem{Note: &n} }

func NewDestroyItem(id string) SequenceItem { return SequenceItem{Destroy: &id} }

// NewActivationItem wraps an Activation as a SequenceItem.
func NewActivationItem(a Activation) SequenceItem { return SequenceItem{Activation: &a} }

// BlockKind identifies the structural block type.
type BlockKind int8

const (
	BlockKindUnknown  BlockKind = iota
	BlockKindAlt                // alt/else alternative branches
	BlockKindOpt                // opt conditional (no else)
	BlockKindLoop               // loop iteration
	BlockKindPar                // par parallel branches (uses 'and' for extras)
	BlockKindCritical           // critical section (uses 'option' for extras)
	BlockKindBreak              // break out of loop
	BlockKindRect               // rectangular visual grouping
)

var blockKindNames = []string{"unknown", "alt", "opt", "loop", "par", "critical", "break", "rect"}

func (b BlockKind) String() string { return enumString(b, blockKindNames) }

// Block is a nested structural element in a sequence diagram.
//
// Items holds the body in source order (messages, notes, nested blocks).
// Branches holds alternative branches for multi-branch kinds:
//   - BlockKindAlt:      else/else-if branches
//   - BlockKindPar:      'and' branches
//   - BlockKindCritical: 'option' branches
//
// Branches is always empty for single-branch kinds (opt, loop, break, rect).
// Each branch is itself a Block with Kind matching the parent.
type Block struct {
	Label    string
	Items    []SequenceItem
	Branches []Block
	Kind     BlockKind
	Fill     string
	HasAlpha bool
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
//
// Cardinality of Participants depends on Position:
//   - NotePositionLeft, NotePositionRight: exactly 1 participant
//   - NotePositionOver: 1 or 2 participants (spans range if 2)
type Note struct {
	Participants []string
	Text         string
	Position     NotePosition
}

type AutoNumber struct {
	Enabled bool
	Start   int
	Step    int
}

type SequenceDiagram struct {
	Participants []Participant
	Items        []SequenceItem
	AutoNumber   AutoNumber
	Boxes        []Box
	Title        string
	// AccTitle / AccDescr carry Mermaid's accessibility metadata. The
	// SVG renderer emits them as <title> / <desc> elements for screen
	// readers; they are otherwise invisible.
	AccTitle string
	AccDescr string
}

// Type implements Diagram.
func (*SequenceDiagram) Type() DiagramType { return Sequence }

var _ Diagram = (*SequenceDiagram)(nil)
