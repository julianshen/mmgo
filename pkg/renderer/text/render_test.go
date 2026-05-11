package text

import (
	"testing"

	"github.com/julianshen/mmgo/pkg/renderer/svgutil"
)

func TestLabelElementsPlain(t *testing.T) {
	r := &mockRuler{}
	elems := LabelElements("hello", 100, 50, 14, svgutil.AnchorMiddle, svgutil.BaselineCentral, "fill:#000", r, 1.2)
	if len(elems) != 1 {
		t.Fatalf("want 1 element, got %d", len(elems))
	}
	txt, ok := elems[0].(*svgutil.Text)
	if !ok {
		t.Fatalf("want *svgutil.Text, got %T", elems[0])
	}
	if txt.Content != "hello" {
		t.Errorf("content = %q", txt.Content)
	}
}

func TestLabelElementsMath(t *testing.T) {
	r := &mockRuler{}
	elems := LabelElements("$$\\frac{1}{2}$$", 100, 50, 14, svgutil.AnchorMiddle, svgutil.BaselineCentral, "fill:#000", r, 1.2)
	if len(elems) == 0 {
		t.Skip("math rendering not available")
	}
	// Should contain a group with math paths.
	_, ok := elems[0].(*svgutil.Group)
	if !ok {
		t.Fatalf("want *svgutil.Group for math, got %T", elems[0])
	}
}

func TestLabelElementsMultiLine(t *testing.T) {
	r := &mockRuler{}
	elems := LabelElements("line1\nline2", 100, 50, 14, svgutil.AnchorMiddle, svgutil.BaselineCentral, "fill:#000", r, 1.2)
	if len(elems) != 2 {
		t.Fatalf("want 2 elements, got %d", len(elems))
	}
}

func TestMeasureLabelPlain(t *testing.T) {
	r := &mockRuler{}
	w, h := MeasureLabel("abc", r, 14, 1.2)
	if w <= 0 || h <= 0 {
		t.Errorf("invalid dimensions: %g x %g", w, h)
	}
}

func TestMeasureLabelMultiLine(t *testing.T) {
	r := &mockRuler{}
	w, h := MeasureLabel("abc\ndef", r, 14, 1.2)
	if w <= 0 || h <= 0 {
		t.Errorf("invalid dimensions: %g x %g", w, h)
	}
	// Two lines should be taller than one.
	_, h1 := MeasureLabel("abc", r, 14, 1.2)
	if h <= h1 {
		t.Errorf("two-line height %g should be > one-line height %g", h, h1)
	}
}
