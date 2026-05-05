package diagram

// SankeyFlow is a single source→target link with a magnitude. Node
// identity is implicit in the Source/Target strings; the diagram
// has no separate node list.
type SankeyFlow struct {
	Source string
	Target string
	Value  float64
}

type SankeyDiagram struct {
	Title    string
	AccTitle string
	AccDescr string
	Flows    []SankeyFlow
}

// Nodes returns the unique node IDs in first-appearance order
// (source before target within each flow). Useful for deterministic
// layout.
func (d *SankeyDiagram) Nodes() []string {
	seen := make(map[string]bool, len(d.Flows)*2)
	var out []string
	for _, f := range d.Flows {
		for _, n := range [...]string{f.Source, f.Target} {
			if !seen[n] {
				seen[n] = true
				out = append(out, n)
			}
		}
	}
	return out
}

func (*SankeyDiagram) Type() DiagramType { return Sankey }

var _ Diagram = (*SankeyDiagram)(nil)
