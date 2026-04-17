// Package diagram defines the shared AST types and interfaces for all
// Mermaid diagram types.
package diagram

// DiagramType identifies which kind of Mermaid diagram an AST represents.
type DiagramType int8

const (
	Unknown   DiagramType = iota // zero-value; prevents uninitialized vars from being a valid type
	Flowchart
	Sequence
	Pie
	Class
	State
	ER
	Gantt
	Mindmap
	Timeline
)

// Diagram is implemented by all diagram AST types.
type Diagram interface {
	Type() DiagramType
}

// enumString looks up v in names, returning "unknown" for out-of-range values.
// It replaces the repetitive switch-based String() implementations across the
// diagram enum types.
func enumString[T ~int8](v T, names []string) string {
	i := int(v)
	if i < 0 || i >= len(names) {
		return "unknown"
	}
	return names[i]
}
