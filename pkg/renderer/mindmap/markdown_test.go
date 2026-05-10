package mindmap

import (
	"testing"

	richtext "github.com/julianshen/mmgo/pkg/renderer/text"
)

func TestParseMarkdownPlain(t *testing.T) {
	segs := richtext.Parse("hello world")
	if len(segs) != 1 {
		t.Fatalf("want 1 segment, got %d", len(segs))
	}
	if segs[0].Text != "hello world" {
		t.Errorf("text = %q", segs[0].Text)
	}
	if segs[0].Bold || segs[0].Italic {
		t.Error("plain segment should not be bold or italic")
	}
}

func TestParseMarkdownBold(t *testing.T) {
	segs := richtext.Parse("**bold**")
	if len(segs) != 1 {
		t.Fatalf("want 1 segment, got %d", len(segs))
	}
	if segs[0].Text != "bold" {
		t.Errorf("text = %q", segs[0].Text)
	}
	if !segs[0].Bold {
		t.Error("expected bold")
	}
	if segs[0].Italic {
		t.Error("should not be italic")
	}
}

func TestParseMarkdownItalic(t *testing.T) {
	segs := richtext.Parse("*italic*")
	if len(segs) != 1 {
		t.Fatalf("want 1 segment, got %d", len(segs))
	}
	if segs[0].Text != "italic" {
		t.Errorf("text = %q", segs[0].Text)
	}
	if !segs[0].Italic {
		t.Error("expected italic")
	}
	if segs[0].Bold {
		t.Error("should not be bold")
	}
}

func TestParseMarkdownBoldItalic(t *testing.T) {
	segs := richtext.Parse("***both***")
	if len(segs) != 1 {
		t.Fatalf("want 1 segment, got %d", len(segs))
	}
	if segs[0].Text != "both" {
		t.Errorf("text = %q", segs[0].Text)
	}
	if !segs[0].Bold {
		t.Error("expected bold")
	}
	if !segs[0].Italic {
		t.Error("expected italic")
	}
}

func TestParseMarkdownMixed(t *testing.T) {
	segs := richtext.Parse("**bold** and *italic*")
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

func TestParseMarkdownBoldWithSurroundingText(t *testing.T) {
	segs := richtext.Parse("before **mid** after")
	if len(segs) != 3 {
		t.Fatalf("want 3 segments, got %d", len(segs))
	}
	if segs[0].Text != "before " {
		t.Errorf("seg[0] = %+v", segs[0])
	}
	if segs[1].Text != "mid" || !segs[1].Bold {
		t.Errorf("seg[1] = %+v", segs[1])
	}
	if segs[2].Text != " after" {
		t.Errorf("seg[2] = %+v", segs[2])
	}
}

func TestParseMarkdownUnclosedBold(t *testing.T) {
	segs := richtext.Parse("**no closing")
	if len(segs) != 1 {
		t.Fatalf("want 1 segment, got %d", len(segs))
	}
	if segs[0].Text != "**no closing" {
		t.Errorf("text = %q", segs[0].Text)
	}
}

func TestParseMarkdownUnclosedItalic(t *testing.T) {
	segs := richtext.Parse("*no closing")
	if len(segs) != 1 {
		t.Fatalf("want 1 segment, got %d", len(segs))
	}
	if segs[0].Text != "*no closing" {
		t.Errorf("text = %q", segs[0].Text)
	}
}

func TestParseMarkdownEmpty(t *testing.T) {
	segs := richtext.Parse("")
	if len(segs) != 1 {
		t.Fatalf("want 1 segment (fallback), got %d", len(segs))
	}
	if segs[0].Text != "" {
		t.Errorf("text = %q", segs[0].Text)
	}
}

func TestParseMarkdownMultipleBold(t *testing.T) {
	segs := richtext.Parse("**a** **b**")
	if len(segs) != 3 {
		t.Fatalf("want 3 segments, got %d: %+v", len(segs), segs)
	}
	if segs[0].Text != "a" || !segs[0].Bold {
		t.Errorf("seg[0] = %+v", segs[0])
	}
	if segs[1].Text != " " {
		t.Errorf("seg[1] = %+v", segs[1])
	}
	if segs[2].Text != "b" || !segs[2].Bold {
		t.Errorf("seg[2] = %+v", segs[2])
	}
}

func TestParseSegmentsMathOnly(t *testing.T) {
	segs := richtext.Parse("$$\\frac{1}{2}$$")
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

func TestParseSegmentsMathAndText(t *testing.T) {
	segs := richtext.Parse("before $$\\alpha$$ after")
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

func TestParseSegmentsUnclosedMath(t *testing.T) {
	segs := richtext.Parse("$$\\frac{1}{2}")
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

func TestParseSegmentsMathWithMarkdown(t *testing.T) {
	segs := richtext.Parse("**bold** $$\\sqrt{x}$$ *italic*")
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
