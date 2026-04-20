package textmeasure

import (
	"fmt"
	"math"
	"testing"

	"golang.org/x/image/font/gofont/goregular"
)

func TestEstimateWidth(t *testing.T) {
	cases := []struct {
		name     string
		s        string
		fontSize float64
		want     float64
	}{
		{"ascii", "hello", 14, 5 * 14 * 0.6},
		{"empty", "", 14, 0},
		{"multi-byte rune", "héllo", 14, 5 * 14 * 0.6}, // 5 runes, not 6 bytes
		{"cjk 3-byte runes", "日本語", 14, 3 * 14 * 0.6}, // 3 runes, not 9 bytes
		{"zero font size", "abc", 0, 0},
		{"negative font size", "abc", -5, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := EstimateWidth(tc.s, tc.fontSize)
			if math.Abs(got-tc.want) > 1e-9 {
				t.Errorf("EstimateWidth(%q,%v) = %v, want %v", tc.s, tc.fontSize, got, tc.want)
			}
		})
	}
}

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

func TestNewRulerInvalidInput(t *testing.T) {
	cases := map[string][]byte{
		"nil":     nil,
		"empty":   {},
		"garbage": []byte("not a font"),
	}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := NewRuler(data); err == nil {
				t.Error("expected error")
			}
		})
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
	w, _ := r.Measure("M\nMMMMMMMMMM", 12)
	longW, _ := r.Measure("MMMMMMMMMM", 12)
	if w < longW*0.99 {
		t.Errorf("expected multi-line width to match longest line: got=%f longest=%f", w, longW)
	}
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
	cases := []string{"héllo", "日本語", "Ω ∀ ∃"}
	for _, text := range cases {
		t.Run(text, func(t *testing.T) {
			w, h := r.Measure(text, 12)
			if w <= 0 || h <= 0 {
				t.Errorf("expected positive dimensions, got w=%f h=%f", w, h)
			}
		})
	}
}

func TestMeasureTrailingNewlineAddsLine(t *testing.T) {
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

func TestMeasureNonPositiveFontSize(t *testing.T) {
	r := mustRuler(t)
	sizes := []float64{0, -5, -0.0001}
	for _, size := range sizes {
		t.Run(fmt.Sprintf("%v", size), func(t *testing.T) {
			w, h := r.Measure("Hello", size)
			if w != 0 || h != 0 {
				t.Errorf("expected 0,0 for size %v, got w=%f h=%f", size, w, h)
			}
		})
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

func TestCloseReleasesFaces(t *testing.T) {
	r := mustRuler(t)
	r.Measure("Hello", 12) // populate cache
	if err := r.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
	if r.faces != nil {
		t.Error("faces should be nil after Close")
	}
}

func TestFaceCaching(t *testing.T) {
	r := mustRuler(t)
	r.Measure("Hello", 12)
	r.Measure("World", 12) // same size — should reuse cached face
	r.Measure("Hello", 14) // different size — new face
	if len(r.faces) != 2 {
		t.Errorf("expected 2 cached faces, got %d", len(r.faces))
	}
}
