package mindmap

import (
	"testing"
)

func TestParseMarkdownPlain(t *testing.T) {
	segs := parseMarkdown("hello world")
	if len(segs) != 1 {
		t.Fatalf("want 1 segment, got %d", len(segs))
	}
	if segs[0].text != "hello world" {
		t.Errorf("text = %q", segs[0].text)
	}
	if segs[0].bold || segs[0].italic {
		t.Error("plain segment should not be bold or italic")
	}
}

func TestParseMarkdownBold(t *testing.T) {
	segs := parseMarkdown("**bold**")
	if len(segs) != 1 {
		t.Fatalf("want 1 segment, got %d", len(segs))
	}
	if segs[0].text != "bold" {
		t.Errorf("text = %q", segs[0].text)
	}
	if !segs[0].bold {
		t.Error("expected bold")
	}
	if segs[0].italic {
		t.Error("should not be italic")
	}
}

func TestParseMarkdownItalic(t *testing.T) {
	segs := parseMarkdown("*italic*")
	if len(segs) != 1 {
		t.Fatalf("want 1 segment, got %d", len(segs))
	}
	if segs[0].text != "italic" {
		t.Errorf("text = %q", segs[0].text)
	}
	if !segs[0].italic {
		t.Error("expected italic")
	}
	if segs[0].bold {
		t.Error("should not be bold")
	}
}

func TestParseMarkdownBoldItalic(t *testing.T) {
	segs := parseMarkdown("***both***")
	if len(segs) != 1 {
		t.Fatalf("want 1 segment, got %d", len(segs))
	}
	if segs[0].text != "both" {
		t.Errorf("text = %q", segs[0].text)
	}
	if !segs[0].bold {
		t.Error("expected bold")
	}
	if !segs[0].italic {
		t.Error("expected italic")
	}
}

func TestParseMarkdownMixed(t *testing.T) {
	segs := parseMarkdown("**bold** and *italic*")
	if len(segs) != 3 {
		t.Fatalf("want 3 segments, got %d: %+v", len(segs), segs)
	}
	if segs[0].text != "bold" || !segs[0].bold {
		t.Errorf("seg[0] = %+v", segs[0])
	}
	if segs[1].text != " and " {
		t.Errorf("seg[1] text = %q", segs[1].text)
	}
	if segs[2].text != "italic" || !segs[2].italic {
		t.Errorf("seg[2] = %+v", segs[2])
	}
}

func TestParseMarkdownBoldWithSurroundingText(t *testing.T) {
	segs := parseMarkdown("before **mid** after")
	if len(segs) != 3 {
		t.Fatalf("want 3 segments, got %d", len(segs))
	}
	if segs[0].text != "before " {
		t.Errorf("seg[0] = %q", segs[0].text)
	}
	if segs[1].text != "mid" || !segs[1].bold {
		t.Errorf("seg[1] = %+v", segs[1])
	}
	if segs[2].text != " after" {
		t.Errorf("seg[2] = %q", segs[2].text)
	}
}

func TestParseMarkdownUnclosedBold(t *testing.T) {
	segs := parseMarkdown("**no closing")
	if len(segs) != 1 {
		t.Fatalf("want 1 segment, got %d", len(segs))
	}
	if segs[0].text != "**no closing" {
		t.Errorf("text = %q", segs[0].text)
	}
}

func TestParseMarkdownUnclosedItalic(t *testing.T) {
	segs := parseMarkdown("*no closing")
	if len(segs) != 1 {
		t.Fatalf("want 1 segment, got %d", len(segs))
	}
	if segs[0].text != "*no closing" {
		t.Errorf("text = %q", segs[0].text)
	}
}

func TestParseMarkdownEmpty(t *testing.T) {
	segs := parseMarkdown("")
	if len(segs) != 1 {
		t.Fatalf("want 1 segment (fallback), got %d", len(segs))
	}
	if segs[0].text != "" {
		t.Errorf("text = %q", segs[0].text)
	}
}

func TestParseMarkdownMultipleBold(t *testing.T) {
	segs := parseMarkdown("**a** **b**")
	if len(segs) != 3 {
		t.Fatalf("want 3 segments, got %d: %+v", len(segs), segs)
	}
	if segs[0].text != "a" || !segs[0].bold {
		t.Errorf("seg[0] = %+v", segs[0])
	}
	if segs[1].text != " " {
		t.Errorf("seg[1] = %+v", segs[1])
	}
	if segs[2].text != "b" || !segs[2].bold {
		t.Errorf("seg[2] = %+v", segs[2])
	}
}

func TestParseSegmentsMathOnly(t *testing.T) {
	segs := parseSegments("$$\\frac{1}{2}$$")
	if len(segs) != 1 {
		t.Fatalf("want 1 segment, got %d: %+v", len(segs), segs)
	}
	if segs[0].math != `\frac{1}{2}` {
		t.Errorf("math = %q", segs[0].math)
	}
	if segs[0].text != "" {
		t.Errorf("text should be empty, got %q", segs[0].text)
	}
}

func TestParseSegmentsMathAndText(t *testing.T) {
	segs := parseSegments("before $$\\alpha$$ after")
	if len(segs) != 3 {
		t.Fatalf("want 3 segments, got %d: %+v", len(segs), segs)
	}
	if segs[0].text != "before " || segs[0].math != "" {
		t.Errorf("seg[0] = %+v", segs[0])
	}
	if segs[1].math != `\alpha` {
		t.Errorf("seg[1].math = %q", segs[1].math)
	}
	if segs[2].text != " after" || segs[2].math != "" {
		t.Errorf("seg[2] = %+v", segs[2])
	}
}

func TestParseSegmentsUnclosedMath(t *testing.T) {
	segs := parseSegments("$$\\frac{1}{2}")
	if len(segs) != 1 {
		t.Fatalf("want 1 segment, got %d: %+v", len(segs), segs)
	}
	if segs[0].text != "$$\\frac{1}{2}" {
		t.Errorf("text = %q", segs[0].text)
	}
	if segs[0].math != "" {
		t.Errorf("math should be empty, got %q", segs[0].math)
	}
}

func TestParseSegmentsMathWithMarkdown(t *testing.T) {
	segs := parseSegments("**bold** $$\\sqrt{x}$$ *italic*")
	if len(segs) != 5 {
		t.Fatalf("want 5 segments, got %d: %+v", len(segs), segs)
	}
	if !segs[0].bold || segs[0].text != "bold" {
		t.Errorf("seg[0] = %+v", segs[0])
	}
	if segs[1].text != " " || segs[1].math != "" {
		t.Errorf("seg[1] = %+v", segs[1])
	}
	if segs[2].math != `\sqrt{x}` {
		t.Errorf("seg[2].math = %q", segs[2].math)
	}
	if segs[3].text != " " || segs[3].math != "" {
		t.Errorf("seg[3] = %+v", segs[3])
	}
	if !segs[4].italic || segs[4].text != "italic" {
		t.Errorf("seg[4] = %+v", segs[4])
	}
}
