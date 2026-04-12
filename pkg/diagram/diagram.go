// Package diagram defines the shared AST types and interfaces for all
// Mermaid diagram types.
package diagram

// DiagramType identifies which kind of Mermaid diagram an AST represents.
type DiagramType int

const (
	Unknown   DiagramType = iota // zero-value; prevents uninitialized vars from being a valid type
	Flowchart
	Sequence
	Pie
	Class
	State
	ER
	Gantt
)

// Diagram is implemented by all diagram AST types.
type Diagram interface {
	Type() DiagramType
}
