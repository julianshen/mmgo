package math

import (
	"strings"
	"testing"
)

func TestSplitTopLevel(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		wantOK   bool
		wantKind []partKind
		wantExpr []string
	}{
		{
			name:   "plain math, no split",
			expr:   `\frac{a}{b}`,
			wantOK: false,
		},
		{
			name:     "top-level comma",
			expr:     `f(x, t)`,
			wantOK:   true,
			wantKind: []partKind{partMath, partComma, partMath},
			wantExpr: []string{"f(x", "", " t)"},
		},
		{
			name:     "top-level superscript",
			expr:     `\pi r^2`,
			wantOK:   true,
			wantKind: []partKind{partMath, partSup},
			wantExpr: []string{`\pi r`, "2"},
		},
		{
			name:     "subscript with braced operand",
			expr:     `x_{i+1}`,
			wantOK:   true,
			wantKind: []partKind{partMath, partSub},
			wantExpr: []string{"x", "i+1"},
		},
		{
			name:   "comma inside braces stays atomic",
			expr:   `\frac{a,b}{c}`,
			wantOK: false,
		},
		{
			name:   "superscript inside braces stays atomic",
			expr:   `\frac{a}{b^2}`,
			wantOK: false,
		},
		{
			name:     "escaped comma not split",
			expr:     `a\,b, c`,
			wantOK:   true,
			wantKind: []partKind{partMath, partComma, partMath},
			wantExpr: []string{`a\,b`, "", " c"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			parts, ok := splitTopLevel(tc.expr)
			if ok != tc.wantOK {
				t.Errorf("ok = %v, want %v", ok, tc.wantOK)
			}
			if !tc.wantOK {
				return
			}
			if len(parts) != len(tc.wantKind) {
				t.Fatalf("got %d parts (%v), want %d (%v)",
					len(parts), parts, len(tc.wantKind), tc.wantKind)
			}
			for i, p := range parts {
				if p.kind != tc.wantKind[i] {
					t.Errorf("part %d kind = %v, want %v", i, p.kind, tc.wantKind[i])
				}
				if p.expr != tc.wantExpr[i] {
					t.Errorf("part %d expr = %q, want %q", i, p.expr, tc.wantExpr[i])
				}
			}
		})
	}
}

func TestReadOperand(t *testing.T) {
	tests := []struct {
		input        string
		wantOperand  string
		wantConsumed int
	}{
		{"2", "2", 1},
		{"{abc}", "abc", 5},
		{"{a{b}c}", "a{b}c", 7},
		{`\pi`, `\pi`, 3},
		{`\,x`, `\,`, 2},
		{"", "", 0},
		{"{unclosed", "unclosed", len("{unclosed")},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			gotOp, gotN := readOperand(tc.input)
			if gotOp != tc.wantOperand {
				t.Errorf("operand = %q, want %q", gotOp, tc.wantOperand)
			}
			if gotN != tc.wantConsumed {
				t.Errorf("consumed = %d, want %d", gotN, tc.wantConsumed)
			}
		})
	}
}

func TestHasNestedSupSub(t *testing.T) {
	tests := []struct {
		expr string
		want bool
	}{
		{`a + b`, false},
		{`a^2`, true},
		{`x_i`, true},
		{`\frac{a}{b^2}`, true},
		{`\sqrt{b^2 - 4ac}`, true},
		{`\pi r`, false},
		{`\,`, false}, // escaped comma, not subscript
	}
	for _, tc := range tests {
		t.Run(tc.expr, func(t *testing.T) {
			if got := hasNestedSupSub(tc.expr); got != tc.want {
				t.Errorf("hasNestedSupSub(%q) = %v, want %v", tc.expr, got, tc.want)
			}
		})
	}
}

func TestIsAlpha(t *testing.T) {
	for _, c := range "abcXYZ" {
		if !isAlpha(byte(c)) {
			t.Errorf("isAlpha(%q) = false, want true", c)
		}
	}
	for _, c := range "0123!@#" {
		if isAlpha(byte(c)) {
			t.Errorf("isAlpha(%q) = true, want false", c)
		}
	}
}

func TestAbsFloat(t *testing.T) {
	cases := map[float64]float64{0: 0, 1.5: 1.5, -2.5: 2.5}
	for in, want := range cases {
		if got := absFloat(in); got != want {
			t.Errorf("absFloat(%g) = %g, want %g", in, got, want)
		}
	}
}

func TestRenderRawAlias(t *testing.T) {
	// Render is the alias that drops the baseline return.
	svg, w, h, err := Render(`a`, 14)
	if err != nil {
		t.Skipf("math rendering not available: %v", err)
	}
	if svg == "" || w <= 0 || h <= 0 {
		t.Errorf("invalid result: svg=%q w=%g h=%g", svg, w, h)
	}
}

func TestRenderEmpty(t *testing.T) {
	// Empty expression should round-trip through the normalize/wrap
	// path; the result may have zero width and height — but the call
	// must not panic.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("panic on empty input: %v", r)
		}
	}()
	_, _, _, _ = Render("", 14)
}

func TestMedianBaseline(t *testing.T) {
	cases := []struct {
		ys   []float64
		want float64
	}{
		{nil, 0},
		{[]float64{5}, 5},
		{[]float64{1, 5, 3}, 3},                     // odd count, middle
		{[]float64{1, 3, 5, 7}, 4},                  // even count, mean of middle two
		{[]float64{11.99, 21.66, 21.66, 31.65}, 21.66}, // fraction + inline letters
	}
	for _, tc := range cases {
		got := medianBaseline(tc.ys)
		if got != tc.want {
			t.Errorf("medianBaseline(%v) = %g, want %g", tc.ys, got, tc.want)
		}
	}
}

func TestRenderRichComma(t *testing.T) {
	// f(x, t) needs the split renderer + manual comma glyph; covers
	// renderParts and renderCharGlyph end-to-end.
	svg, w, h, err := RenderRich(`f(x, t)`, 14)
	if err != nil {
		t.Skipf("math rendering not available in test environment: %v", err)
	}
	if w <= 0 || h <= 0 {
		t.Errorf("invalid dimensions: w=%g h=%g", w, h)
	}
	// At least two math chunks separated by a comma glyph → three
	// transform groups in the output.
	if got := strings.Count(svg, "<g "); got < 3 {
		t.Errorf("expected ≥3 <g> wrappers, got %d in %s", got, svg)
	}
}

func TestRenderRichSuperscript(t *testing.T) {
	// Top-level superscript exercises the partSup branch.
	_, w, h, err := RenderRich(`\pi r^2`, 14)
	if err != nil {
		t.Skipf("math rendering not available in test environment: %v", err)
	}
	if w <= 0 || h <= 0 {
		t.Errorf("invalid dimensions: w=%g h=%g", w, h)
	}
}

func TestRenderRichSupWithCommaOperand(t *testing.T) {
	// A superscript whose operand is itself a comma-split expression
	// exercises the renderParts() alias (recursion path).
	_, _, _, err := RenderRich(`x^{a,b}`, 14)
	if err != nil {
		t.Skipf("math rendering not available in test environment: %v", err)
	}
}

func TestRenderRichFlattensNestedSupSub(t *testing.T) {
	// \frac{a}{b^2} has a nested superscript inside the denominator
	// that mtex cannot render. RenderRich flattens to \frac{a}{b2}
	// rather than failing the render — the surrounding structure is
	// preserved even though the exponent is lost.
	svg, w, h, err := RenderRich(`\frac{a}{b^2}`, 14)
	if err != nil {
		t.Skipf("math rendering not available in test environment: %v", err)
	}
	if w <= 0 || h <= 0 {
		t.Errorf("invalid dimensions: w=%g h=%g", w, h)
	}
	if svg == "" {
		t.Error("expected non-empty svg output")
	}
}

func TestFlattenNestedSupSub(t *testing.T) {
	cases := map[string]string{
		`a + b`:              `a + b`,
		`b^2`:                `b2`,
		`x_i`:                `xi`,
		`\frac{a}{b^2}`:      `\frac{a}{b2}`,
		`\sqrt{b^2 - 4ac}`:   `\sqrt{b2 - 4ac}`,
		`a^{n+1}`:            `a{n+1}`, // marker gone; the brace group survives
	}
	for in, want := range cases {
		if got := flattenNestedSupSub(in); got != want {
			t.Errorf("flattenNestedSupSub(%q) = %q, want %q", in, got, want)
		}
	}
}
