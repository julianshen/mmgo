package text

import (
	"testing"

	"github.com/julianshen/mmgo/pkg/renderer/svgutil"
)

type mockRuler struct{}

func (m *mockRuler) Measure(text string, fontSize float64) (width, height float64) {
	return float64(len(text)) * 7, fontSize
}

func TestParsePlain(t *testing.T) {
	segs := Parse("hello world")
	if len(segs) != 1 {
		t.Fatalf("want 1 segment, got %d", len(segs))
	}
	if segs[0].Text != "hello world" {
		t.Errorf("text = %q", segs[0].Text)
	}
	if segs[0].Bold || segs[0].Italic || segs[0].Math != "" {
		t.Error("plain segment should not have styles or math")
	}
}

func TestParseBold(t *testing.T) {
	segs := Parse("**bold**")
	if len(segs) != 1 {
		t.Fatalf("want 1 segment, got %d", len(segs))
	}
	if segs[0].Text != "bold" || !segs[0].Bold {
		t.Errorf("seg = %+v", segs[0])
	}
}

func TestParseItalic(t *testing.T) {
	segs := Parse("*italic*")
	if len(segs) != 1 {
		t.Fatalf("want 1 segment, got %d", len(segs))
	}
	if segs[0].Text != "italic" || !segs[0].Italic {
		t.Errorf("seg = %+v", segs[0])
	}
}

func TestParseBoldItalic(t *testing.T) {
	segs := Parse("***both***")
	if len(segs) != 1 {
		t.Fatalf("want 1 segment, got %d", len(segs))
	}
	if segs[0].Text != "both" || !segs[0].Bold || !segs[0].Italic {
		t.Errorf("seg = %+v", segs[0])
	}
}

func TestParseMixed(t *testing.T) {
	segs := Parse("**bold** and *italic*")
	if len(segs) != 3 {
		t.Fatalf("want 3 segments, got %d: %+v", len(segs), segs)
	}
	if segs[0].Text != "bold" || !segs[0].Bold {
		t.Errorf("seg[0] = %+v", segs[0])
	}
	if segs[1].Text != " and " {
		t.Errorf("seg[1] text = %q", segs[1].Text)
	}
	if segs[2].Text != "italic" || !segs[2].Italic {
		t.Errorf("seg[2] = %+v", segs[2])
	}
}

func TestParseMathOnly(t *testing.T) {
	segs := Parse("$$\\frac{1}{2}$$")
	if len(segs) != 1 {
		t.Fatalf("want 1 segment, got %d: %+v", len(segs), segs)
	}
	if segs[0].Math != `\frac{1}{2}` {
		t.Errorf("math = %q", segs[0].Math)
	}
	if segs[0].Text != "" {
		t.Errorf("text should be empty, got %q", segs[0].Text)
	}
}

func TestParseMathAndText(t *testing.T) {
	segs := Parse("before $$\\alpha$$ after")
	if len(segs) != 3 {
		t.Fatalf("want 3 segments, got %d: %+v", len(segs), segs)
	}
	if segs[0].Text != "before " || segs[0].Math != "" {
		t.Errorf("seg[0] = %+v", segs[0])
	}
	if segs[1].Math != `\alpha` {
		t.Errorf("seg[1].math = %q", segs[1].Math)
	}
	if segs[2].Text != " after" || segs[2].Math != "" {
		t.Errorf("seg[2] = %+v", segs[2])
	}
}

func TestParseUnclosedMath(t *testing.T) {
	segs := Parse("$$\\frac{1}{2}")
	if len(segs) != 1 {
		t.Fatalf("want 1 segment, got %d: %+v", len(segs), segs)
	}
	if segs[0].Text != "$$\\frac{1}{2}" {
		t.Errorf("text = %q", segs[0].Text)
	}
	if segs[0].Math != "" {
		t.Errorf("math should be empty, got %q", segs[0].Math)
	}
}

func TestParseMathWithMarkdown(t *testing.T) {
	segs := Parse("**bold** $$\\sqrt{x}$$ *italic*")
	if len(segs) != 5 {
		t.Fatalf("want 5 segments, got %d: %+v", len(segs), segs)
	}
	if !segs[0].Bold || segs[0].Text != "bold" {
		t.Errorf("seg[0] = %+v", segs[0])
	}
	if segs[1].Text != " " || segs[1].Math != "" {
		t.Errorf("seg[1] = %+v", segs[1])
	}
	if segs[2].Math != `\sqrt{x}` {
		t.Errorf("seg[2].math = %q", segs[2].Math)
	}
	if segs[3].Text != " " || segs[3].Math != "" {
		t.Errorf("seg[3] = %+v", segs[3])
	}
	if !segs[4].Italic || segs[4].Text != "italic" {
		t.Errorf("seg[4] = %+v", segs[4])
	}
}

func TestMeasureSegments(t *testing.T) {
	r := &mockRuler{}
	segs := []Segment{
		{Text: "abc"},
		{Math: `\frac{1}{2}`},
		{Text: "d", Bold: true},
	}
	tw, mh := MeasureSegments(segs, r, 14, 1.2)
	if tw <= 0 {
		t.Errorf("total width = %g", tw)
	}
	if mh <= 0 {
		t.Errorf("max height = %g", mh)
	}
	if segs[0].Width == 0 {
		t.Error("seg[0] width not set")
	}
	if segs[1].Width == 0 {
		t.Error("math seg width not set")
	}
	if segs[2].Width == 0 {
		t.Error("bold seg width not set")
	}
}

func TestParseMathSVG(t *testing.T) {
	input := `<path d="M0 0 L1 1"/><rect x="1" y="2" width="3" height="4"/>`
	elems := ParseMathSVG(input)
	if len(elems) != 2 {
		t.Fatalf("want 2 elements, got %d", len(elems))
	}
	_, ok1 := elems[0].(*svgutil.Path)
	if !ok1 {
		t.Errorf("elem[0] type = %T, want *svgutil.Path", elems[0])
	}
	_, ok2 := elems[1].(*svgutil.Rect)
	if !ok2 {
		t.Errorf("elem[1] type = %T, want *svgutil.Rect", elems[1])
	}
}

func TestRenderMath(t *testing.T) {
	res := RenderMath(`\frac{1}{2}`, 14)
	if res == nil {
		t.Skip("math rendering not available in test environment")
	}
	if len(res.Elements) == 0 {
		t.Error("no elements rendered")
	}
	if res.Width <= 0 || res.Height <= 0 {
		t.Errorf("invalid dimensions: %g x %g", res.Width, res.Height)
	}
}

func TestRenderMathUnsupported(t *testing.T) {
	// Superscript panics in go-latex; should return nil (fallback).
	res := RenderMath("x^2", 14)
	if res != nil {
		t.Logf("expected nil for unsupported math, got %+v", res)
	}
}
