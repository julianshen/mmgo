package diagram

// KanbanTask is a single card within a Kanban section. ID is optional
// (auto-generated if the source didn't supply one). Metadata holds
// `@{ key: value, ... }` attributes such as priority, assignee, or
// ticket reference — whatever the source provides.
type KanbanTask struct {
	ID       string
	Text     string
	Metadata map[string]string
}

// KanbanSection is one column on the board. Tasks are listed in
// source order. ID is empty when the source didn't supply an
// `id[title]` prefix. Metadata mirrors KanbanTask.Metadata and is
// reserved for future per-column styling (icons, colors) that
// Mermaid supports.
type KanbanSection struct {
	ID       string
	Title    string
	Metadata map[string]string
	Tasks    []KanbanTask
}

// KanbanDiagram is the AST for a Mermaid kanban diagram. Sections are
// rendered as columns left-to-right in declaration order.
type KanbanDiagram struct {
	Title    string
	AccTitle string
	AccDescr string
	// TicketBaseURL is sourced from frontmatter
	// `config.kanban.ticketBaseUrl`. Renderers that consume the
	// `ticket` task metadata key wrap the card in `<a href>` after
	// substituting `#TICKET#` with the ticket id (Phase 2).
	TicketBaseURL string
	Sections      []KanbanSection
}

func (*KanbanDiagram) Type() DiagramType { return Kanban }

var _ Diagram = (*KanbanDiagram)(nil)
