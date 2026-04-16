package pie

import (
	"strings"
	"testing"
)

func TestParseHeaderRequired(t *testing.T) {
	_, err := Parse(strings.NewReader(`"Dogs" : 10`))
	if err == nil {
		t.Fatal("expected error for missing header")
	}
}

func TestParseEmptyInput(t *testing.T) {
	_, err := Parse(strings.NewReader(""))
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestParseHeaderOnly(t *testing.T) {
	d, err := Parse(strings.NewReader("pie"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Title != "" || len(d.Slices) != 0 {
		t.Errorf("empty pie should have no title/slices: %+v", d)
	}
}

func TestParseTitleOnHeader(t *testing.T) {
	d, err := Parse(strings.NewReader("pie title My Chart"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Title != "My Chart" {
		t.Errorf("Title = %q, want %q", d.Title, "My Chart")
	}
}

func TestParseTitleOnSeparateLine(t *testing.T) {
	d, err := Parse(strings.NewReader("pie\n    title Pets"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Title != "Pets" {
		t.Errorf("Title = %q, want %q", d.Title, "Pets")
	}
}

func TestParseSlices(t *testing.T) {
	input := `pie title Pets
    "Dogs" : 386
    "Cats" : 85
    "Rats" : 15`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Slices) != 3 {
		t.Fatalf("got %d slices, want 3", len(d.Slices))
	}
	want := []struct {
		label string
		value float64
	}{
		{"Dogs", 386},
		{"Cats", 85},
		{"Rats", 15},
	}
	for i, w := range want {
		if d.Slices[i].Label != w.label {
			t.Errorf("slice[%d].Label = %q, want %q", i, d.Slices[i].Label, w.label)
		}
		if d.Slices[i].Value != w.value {
			t.Errorf("slice[%d].Value = %v, want %v", i, d.Slices[i].Value, w.value)
		}
	}
}

func TestParseShowData(t *testing.T) {
	d, err := Parse(strings.NewReader("pie showData\n    \"A\" : 1"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !d.ShowData {
		t.Error("ShowData should be true")
	}
}

func TestParseComments(t *testing.T) {
	input := `pie title X
    %% a comment
    "A" : 10 %% trailing`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(d.Slices) != 1 || d.Slices[0].Label != "A" {
		t.Errorf("unexpected: %+v", d.Slices)
	}
}

func TestParseSliceMissingQuotes(t *testing.T) {
	_, err := Parse(strings.NewReader("pie\n    Dogs : 10"))
	if err == nil {
		t.Fatal("expected error: slice label must be quoted")
	}
}

func TestParseSliceMissingColon(t *testing.T) {
	_, err := Parse(strings.NewReader("pie\n    \"Dogs\" 10"))
	if err == nil {
		t.Fatal("expected error: missing colon")
	}
}

func TestParseSliceInvalidValue(t *testing.T) {
	_, err := Parse(strings.NewReader("pie\n    \"Dogs\" : abc"))
	if err == nil {
		t.Fatal("expected error: non-numeric value")
	}
}

func TestParseDecimalValues(t *testing.T) {
	d, err := Parse(strings.NewReader("pie\n    \"A\" : 3.14"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.Slices[0].Value != 3.14 {
		t.Errorf("Value = %v, want 3.14", d.Slices[0].Value)
	}
}
