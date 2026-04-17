package diagram

type TimelineEvent struct {
	Time   string
	Events []string
}

type TimelineSection struct {
	Name   string
	Events []TimelineEvent
}

type TimelineDiagram struct {
	Title    string
	Sections []TimelineSection
	Events   []TimelineEvent // top-level events when no sections
}

func (*TimelineDiagram) Type() DiagramType { return Timeline }

var _ Diagram = (*TimelineDiagram)(nil)
