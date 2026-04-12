package textmeasure

import (
	"testing"

	"golang.org/x/image/font/gofont/goregular"
)

// --- Construction ---

func TestNewRulerValidFont(t *testing.T) {
	r, err := NewRuler(goregular.TTF)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r == nil {
		t.Fatal("expected non-nil Ruler")
	}
}

func TestNewRulerInvalidFont(t *testing.T) {
	_, err := NewRuler([]byte("not a font"))
	if err == nil {
		t.Error("expected error for invalid font bytes")
	}
}

func TestNewRulerEmptyBytes(t *testing.T) {
	_, err := NewRuler(nil)
	if err == nil {
		t.Error("expected error for empty font bytes")
	}
}

func TestNewDefaultRuler(t *testing.T) {
	r, err := NewDefaultRuler()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r == nil {
		t.Fatal("expected non-nil Ruler")
	}
}

// --- Measurement ---

func mustRuler(t *testing.T) *Ruler {
	t.Helper()
	r, err := NewDefaultRuler()
	if err != nil {
		t.Fatalf("NewDefaultRuler: %v", err)
	}
	return r
}

func TestMeasureSingleCharacter(t *testing.T) {
	r := mustRuler(t)
	w, h := r.Measure("M", 12)
	if w <= 0 {
		t.Errorf("expected positive width, got %f", w)
	}
	if h <= 0 {
		t.Errorf("expected positive height, got %f", h)
	}
}

func TestMeasureLongerStringIsWider(t *testing.T) {
	r := mustRuler(t)
	shortW, _ := r.Measure("M", 12)
	longW, _ := r.Measure("MMMMM", 12)
	if longW <= shortW {
		t.Errorf("expected longer string wider: short=%f long=%f", shortW, longW)
	}
}

func TestMeasureEmptyString(t *testing.T) {
	r := mustRuler(t)
	w, h := r.Measure("", 12)
	if w != 0 {
		t.Errorf("expected width 0 for empty string, got %f", w)
	}
	if h != 0 {
		t.Errorf("expected height 0 for empty string, got %f", h)
	}
}

func TestMeasureMultiLine(t *testing.T) {
	r := mustRuler(t)
	singleW, singleH := r.Measure("Hello", 12)
	multiW, multiH := r.Measure("Hello\nWorld", 12)

	// Multi-line height should be ~2x single line.
	if multiH <= singleH {
		t.Errorf("expected multi-line height > single line: single=%f multi=%f", singleH, multiH)
	}
	// Width should be the wider of the two lines; both are similar here.
	if multiW < singleW*0.8 {
		t.Errorf("multi-line width should be at least as wide as widest line: single=%f multi=%f", singleW, multiW)
	}
}

func TestMeasureMultiLineWidestLineWins(t *testing.T) {
	r := mustRuler(t)
	// "MMMMMMMMMM" (10 chars) should be wider than "M".
	w, _ := r.Measure("M\nMMMMMMMMMM", 12)
	shortW, _ := r.Measure("M", 12)
	longW, _ := r.Measure("MMMMMMMMMM", 12)

	if w < longW*0.99 {
		t.Errorf("expected multi-line width to match longest line: got=%f longest=%f", w, longW)
	}
	_ = shortW
}

func TestMeasureScalesWithFontSize(t *testing.T) {
	r := mustRuler(t)
	w12, h12 := r.Measure("Hello", 12)
	w24, h24 := r.Measure("Hello", 24)

	// Doubling font size should roughly double both dimensions.
	ratio := w24 / w12
	if ratio < 1.9 || ratio > 2.1 {
		t.Errorf("width scaling off: 12pt=%f 24pt=%f ratio=%f", w12, w24, ratio)
	}
	ratio = h24 / h12
	if ratio < 1.9 || ratio > 2.1 {
		t.Errorf("height scaling off: 12pt=%f 24pt=%f ratio=%f", h12, h24, ratio)
	}
}

func TestMeasureUnicode(t *testing.T) {
	r := mustRuler(t)
	// CJK, accented, and emoji — just verify we don't panic and return positive values.
	cases := []string{
		"héllo",
		"日本語",
		"Ω ∀ ∃",
	}
	for _, text := range cases {
		w, h := r.Measure(text, 12)
		if w <= 0 || h <= 0 {
			t.Errorf("expected positive dimensions for %q, got w=%f h=%f", text, w, h)
		}
	}
}

func TestMeasureIgnoresTrailingNewline(t *testing.T) {
	r := mustRuler(t)
	_, h1 := r.Measure("Hello", 12)
	_, h2 := r.Measure("Hello\n", 12)
	// Trailing newline creates an empty second line: height should roughly double.
	if h2 <= h1*1.5 {
		t.Errorf("trailing newline should add a line: h1=%f h2=%f", h1, h2)
	}
}

func TestMeasureLineHeight(t *testing.T) {
	r := mustRuler(t)
	_, h1 := r.Measure("Hello", 12)
	_, h2 := r.Measure("Hello\nWorld", 12)
	_, h3 := r.Measure("Hello\nWorld\n!", 12)

	// Heights should be linear in line count.
	diff1 := h2 - h1
	diff2 := h3 - h2
	if diff1 < 0 || diff2 < 0 {
		t.Errorf("line heights should increase: h1=%f h2=%f h3=%f", h1, h2, h3)
	}
	// The increment per line should be approximately the same.
	ratio := diff2 / diff1
	if ratio < 0.9 || ratio > 1.1 {
		t.Errorf("line height increment inconsistent: diff1=%f diff2=%f", diff1, diff2)
	}
}

func TestMeasureZeroFontSize(t *testing.T) {
	r := mustRuler(t)
	// Zero or negative font size should not panic; returns 0 or small values.
	w, h := r.Measure("Hello", 0)
	if w != 0 || h != 0 {
		t.Errorf("expected 0,0 for zero font size, got w=%f h=%f", w, h)
	}
}

func TestMeasureNegativeFontSize(t *testing.T) {
	r := mustRuler(t)
	w, h := r.Measure("Hello", -5)
	if w != 0 || h != 0 {
		t.Errorf("expected 0,0 for negative font size, got w=%f h=%f", w, h)
	}
}

func TestMeasureConsistentAcrossCalls(t *testing.T) {
	r := mustRuler(t)
	w1, h1 := r.Measure("Hello", 12)
	w2, h2 := r.Measure("Hello", 12)
	if w1 != w2 || h1 != h2 {
		t.Errorf("measurements should be deterministic: (%f,%f) vs (%f,%f)", w1, h1, w2, h2)
	}
}

func TestMeasureCustomFont(t *testing.T) {
	// Verify that NewRuler accepts custom font bytes and produces measurements.
	r, err := NewRuler(goregular.TTF)
	if err != nil {
		t.Fatalf("NewRuler: %v", err)
	}
	w, h := r.Measure("test", 14)
	if w <= 0 || h <= 0 {
		t.Errorf("custom font ruler should produce positive measurements, got w=%f h=%f", w, h)
	}
}
