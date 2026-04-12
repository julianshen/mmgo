package diagram

import "testing"

func TestDiagramTypeConstants(t *testing.T) {
	// Unknown must be the zero value so uninitialized DiagramType vars
	// default to "not yet classified" rather than a valid diagram type.
	if Unknown != 0 {
		t.Errorf("Unknown must be zero-value, got %d", Unknown)
	}

	// Verify enum values are distinct.
	types := []DiagramType{Unknown, Flowchart, Sequence, Pie, Class, State, ER, Gantt}
	seen := make(map[DiagramType]bool)
	for _, dt := range types {
		if seen[dt] {
			t.Errorf("duplicate DiagramType value: %d", dt)
		}
		seen[dt] = true
	}
}
