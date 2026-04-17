package diagram

import "time"

type TaskStatus int8

const (
	TaskStatusNone   TaskStatus = iota
	TaskStatusDone
	TaskStatusActive
	TaskStatusCrit
)

var taskStatusNames = []string{"none", "done", "active", "crit"}

func (s TaskStatus) String() string { return enumString(s, taskStatusNames) }

type GanttTask struct {
	ID       string
	Name     string
	Status   TaskStatus
	Start    time.Time
	End      time.Time
	After    string
	Section  string
}

type GanttDiagram struct {
	Title      string
	DateFormat string
	Sections   []string
	Tasks      []GanttTask
}

func (*GanttDiagram) Type() DiagramType { return Gantt }

var _ Diagram = (*GanttDiagram)(nil)
