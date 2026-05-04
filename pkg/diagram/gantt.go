package diagram

import "time"

// TaskStatus is a bitmask. A single task can carry multiple status
// flags (Mermaid permits combinations such as `crit, active` or
// `crit, milestone`). The renderer maps the combined value to a
// fill via Theme.taskColor, which uses a fixed priority order
// (Crit > Active > Done > None) when more than one bit is set.
type TaskStatus int8

const (
	TaskStatusNone      TaskStatus = 0
	TaskStatusDone      TaskStatus = 1 << 0
	TaskStatusActive    TaskStatus = 1 << 1
	TaskStatusCrit      TaskStatus = 1 << 2
	TaskStatusMilestone TaskStatus = 1 << 3
)

// Has reports whether every bit in flag is set on s. Returns true
// for TaskStatusNone since 0&0 == 0.
func (s TaskStatus) Has(flag TaskStatus) bool { return s&flag == flag }

// String renders set flags as `crit|active|done|milestone` (in
// declaration order) for diagnostics. None → "none".
func (s TaskStatus) String() string {
	if s == TaskStatusNone {
		return "none"
	}
	var parts []string
	if s.Has(TaskStatusDone) {
		parts = append(parts, "done")
	}
	if s.Has(TaskStatusActive) {
		parts = append(parts, "active")
	}
	if s.Has(TaskStatusCrit) {
		parts = append(parts, "crit")
	}
	if s.Has(TaskStatusMilestone) {
		parts = append(parts, "milestone")
	}
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += "|"
		}
		out += p
	}
	return out
}

type GanttTask struct {
	ID      string
	Name    string
	Status  TaskStatus
	Start   time.Time
	End     time.Time
	After   []string
	Until   []string
	Section string
}

// GanttClickDef binds a click action to a task. Either URL or
// Callback is set; the renderer emits an `<a>` wrap for URL
// forms and an empty class+data hook for Callback forms.
type GanttClickDef struct {
	TaskID   string
	URL      string
	Tooltip  string
	Target   string
	Callback string
}

// GanttVert marks a vertical line at a date independent of any
// task, useful for highlighting external events on the timeline
// (release window, regulatory deadline, etc.).
type GanttVert struct {
	ID    string
	Date  time.Time
	Label string
}

// GanttDiagram is the parsed representation of a Mermaid Gantt
// chart. Calendar / axis directives are surfaced as raw strings;
// the renderer converts them into Go layout strings or interval
// definitions at render time.
type GanttDiagram struct {
	Title        string
	AccTitle     string
	AccDescr     string
	DateFormat   string
	AxisFormat   string
	TickInterval string
	Weekday      string
	Excludes     []string
	Includes     []string
	TodayMarker  string
	Sections     []string
	Tasks        []GanttTask
	Clicks       []GanttClickDef
	Verts        []GanttVert
}

func (*GanttDiagram) Type() DiagramType { return Gantt }

var _ Diagram = (*GanttDiagram)(nil)
