package diagram

import "testing"

func TestPieImplementsDiagram(t *testing.T) {
	var d Diagram = &PieDiagram{}
	if d.Type() != Pie {
		t.Errorf("expected Type() = Pie, got %v", d.Type())
	}
}

func TestPieConstruction(t *testing.T) {
	p := &PieDiagram{
		Title: "Pets",
		Slices: []Slice{
			{Label: "Dogs", Value: 386},
			{Label: "Cats", Value: 85},
			{Label: "Rats", Value: 15},
		},
		ShowData: true,
	}
	if p.Title != "Pets" {
		t.Errorf("unexpected title: %q", p.Title)
	}
	if len(p.Slices) != 3 {
		t.Errorf("expected 3 slices, got %d", len(p.Slices))
	}
	if !p.ShowData {
		t.Error("ShowData should be true")
	}
}

func TestSliceNegativeValueAllowed(t *testing.T) {
	// Validation happens at parse time; AST accepts any float.
	s := Slice{Label: "odd", Value: -1.5}
	if s.Value != -1.5 {
		t.Error("AST should preserve any float value")
	}
}
