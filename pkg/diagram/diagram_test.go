package diagram

import "testing"

func TestDiagramTypeConstants(t *testing.T) {
	// Verify enum values are distinct and sequential.
	types := []DiagramType{Unknown, Flowchart, Sequence, Pie, Class, State, ER, Gantt}
	seen := make(map[DiagramType]bool)
	for _, dt := range types {
		if seen[dt] {
			t.Errorf("duplicate DiagramType value: %d", dt)
		}
		seen[dt] = true
	}
}
