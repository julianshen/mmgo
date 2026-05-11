package math

import (
	"strings"
	"testing"
)

func TestRenderBasic(t *testing.T) {
	_, w, h, err := Render("x + y", 14)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if w <= 0 || h <= 0 {
		t.Errorf("invalid dimensions: w=%g, h=%g", w, h)
	}
}

func TestRenderSqrt(t *testing.T) {
	_, w, h, err := Render("\\sqrt{x}", 14)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if w <= 0 || h <= 0 {
		t.Errorf("invalid dimensions: w=%g, h=%g", w, h)
	}
}

func TestRenderFraction(t *testing.T) {
	svg, w, h, err := Render("\\frac{1}{2}", 14)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if w <= 0 || h <= 0 {
		t.Errorf("invalid dimensions: w=%g, h=%g", w, h)
	}
	if !strings.Contains(svg, "path") {
		t.Error("expected path elements in output")
	}
}

func TestRenderGreek(t *testing.T) {
	// go-latex/mtex requires math mode ($...$) to resolve Greek letters
	// and operators.  Render automatically wraps bare expressions.
	svg, w, h, err := Render(`\alpha + \beta`, 14)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if w <= 0 || h <= 0 {
		t.Errorf("invalid dimensions: w=%g, h=%g", w, h)
	}
	if !strings.Contains(svg, "path") {
		t.Error("expected path elements in output")
	}
	// In text mode the backslash glyph would be rendered; in math mode
	// we should see distinct paths for alpha and beta.
	if strings.Count(svg, "<path") < 2 {
		t.Error("expected at least two distinct path elements for alpha and beta")
	}
}

func TestNormalizeMathExpr(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{`\\frac{1}{2}`, `\frac{1}{2}`},
		{`\\alpha`, `\alpha`},
		{`\\sqrt{x}`, `\sqrt{x}`},
		{`\\frac{a}{b} + \\frac{c}{d}`, `\frac{a}{b} + \frac{c}{d}`},
		{`no backslash`, `no backslash`},
		{`single \ backslash`, `single \ backslash`},
	}
	for _, tc := range tests {
		got := normalizeMathExpr(tc.input)
		if got != tc.want {
			t.Errorf("normalizeMathExpr(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
