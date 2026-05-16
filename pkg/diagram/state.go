package diagram

type StateKind int8

const (
	StateKindNormal StateKind = iota
	StateKindFork             // <<fork>>
	StateKindJoin             // <<join>>
	StateKindChoice           // <<choice>>
	StateKindHistory          // <<history>>
	StateKindDeepHistory      // <<deepHistory>>
)

var stateKindNames = []string{"normal", "fork", "join", "choice", "history", "deepHistory"}

func (s StateKind) String() string { return enumString(s, stateKindNames) }

type StateDef struct {
	ID    string
	Label string
	Kind  StateKind
	// Children holds the inner states for a composite state with
	// a single (default) region. For composite states split by
	// `--` separators into parallel regions, see Regions instead;
	// when Regions is non-empty, Children is the concatenated
	// view across all regions in source order.
	Children []StateDef
	// Regions holds the parallel regions of a composite state.
	// Empty for non-composite or single-region composites. Each
	// entry is the slice of states that belong to that region.
	Regions [][]StateDef
	// CSSClasses are user-defined CSS class names attached via
	// `class S1,S2 foo` or the inline `S1:::foo` shorthand.
	CSSClasses []string
}

type StateTransition struct {
	From  string
	To    string
	Label string
	// Scope is the ID of the enclosing composite state in which the
	// transition was written, or "" for the root scope. Pseudo-state
	// endpoints ([*]) are resolved against this scope: `[*] --> Foo`
	// inside `state Bar { … }` denotes the initial state of Bar, not
	// of the root diagram.
	Scope string
	// RegionIdx distinguishes parallel regions within Scope. When the
	// enclosing composite has no `--` separators, RegionIdx is 0 for
	// every transition. With separators, transitions before the first
	// `--` are region 0, between the first and second `--` are
	// region 1, and so on. Mermaid scopes `[*]` per region (each
	// region gets its own initial / final state), so the renderer
	// keys pseudo-state dedup off (Scope, RegionIdx) rather than
	// Scope alone.
	RegionIdx int
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

// StateStyleDef is a per-state inline style override parsed from
// `style ID fill:#f9f,stroke:#333`.
type StateStyleDef struct {
	StateID string
	CSS     string
}

// StateClickDef binds a click action to a state. Either URL or
// Callback is set; the renderer emits a hyperlink for URL forms and
// keeps Callback as metadata for downstream tooling.
type StateClickDef struct {
	StateID  string
	URL      string
	Tooltip  string
	Target   string
	Callback string
}

type StateDiagram struct {
	States      []StateDef
	Transitions []StateTransition
	Notes       []StateNote
	// CSSClasses maps a user-defined class name (from `classDef foo …`)
	// to its semicolon-separated CSS declarations.
	CSSClasses map[string]string
	// Styles are per-state inline overrides from `style ID …` lines.
	Styles []StateStyleDef
	// Clicks are click / link / callback bindings.
	Clicks []StateClickDef
	// Direction is the layout flow. DirectionUnknown means "use the
	// renderer's default" (currently top-to-bottom).
	Direction Direction
	// Title and accessibility metadata.
	Title    string
	AccTitle string
	AccDescr string
}

func (*StateDiagram) Type() DiagramType { return State }

var _ Diagram = (*StateDiagram)(nil)
