package mindmap

import (
	"testing"

	"github.com/julianshen/mmgo/pkg/renderer/svgutil"
)

func TestParseMathSVGPathAndRect(t *testing.T) {
	input := `<path d="M0.034 -6.271 L0.034 -6.268"/><rect x="0.000" y="-9.771" width="2.393" height="0.875"/><path d="M0.036 -17.152 L0.034 -17.152"/>`
	elems := parseMathSVG(input)
	if len(elems) != 3 {
		t.Fatalf("want 3 elements, got %d", len(elems))
	}

	p1, ok := elems[0].(*path)
	if !ok {
		t.Fatalf("elem[0] type = %T, want *path", elems[0])
	}
	if p1.D != "M0.034 -6.271 L0.034 -6.268" {
		t.Errorf("path d = %q", p1.D)
	}

	r, ok := elems[1].(*rect)
	if !ok {
		t.Fatalf("elem[1] type = %T, want *rect", elems[1])
	}
	if float64(r.X) != 0.00 || float64(r.Y) != -9.771 || float64(r.Width) != 2.393 || float64(r.Height) != 0.875 {
		t.Errorf("rect = (%v, %v, %v, %v)", r.X, r.Y, r.Width, r.Height)
	}

	p2, ok := elems[2].(*path)
	if !ok {
		t.Fatalf("elem[2] type = %T, want *path", elems[2])
	}
	if p2.D != "M0.036 -17.152 L0.034 -17.152" {
		t.Errorf("path d = %q", p2.D)
	}
}

func TestParseMathSVGEmpty(t *testing.T) {
	elems := parseMathSVG("")
	if len(elems) != 0 {
		t.Errorf("want 0 elements, got %d", len(elems))
	}
}

func TestParseMathSVGInvalid(t *testing.T) {
	// No closing />
	elems := parseMathSVG("<path d=\"M0 0\"")
	if len(elems) != 0 {
		t.Errorf("want 0 elements for invalid input, got %d", len(elems))
	}
}

func TestParseAttrs(t *testing.T) {
	m := parseAttrs(`<path d="M0 0 L1 1" fill="none"/>`)
	if m["d"] != "M0 0 L1 1" {
		t.Errorf("d = %q", m["d"])
	}
	if m["fill"] != "none" {
		t.Errorf("fill = %q", m["fill"])
	}
}

func TestParseAttrsNoQuotes(t *testing.T) {
	m := parseAttrs(`<path d=M0 0/>`)
	if len(m) != 0 {
		t.Errorf("want empty map for unquoted attr, got %+v", m)
	}
}

func TestRawSVGDeprecated(t *testing.T) {
	// rawSVG is kept for backward compatibility but should not be used
	// for new math rendering; verify it still compiles.
	_ = rawSVG{Content: "<path d='M0 0'/>"}
}

func TestMathSVGIntegration(t *testing.T) {
	// Verify parseMathSVG produces types compatible with svgutil.MarshalSVG.
	elems := parseMathSVG(`<path d="M0 0"><rect x="1" y="2" width="3" height="4">`)
	doc := svgutil.Doc{
		XMLNS:   "http://www.w3.org/2000/svg",
		ViewBox: "0 0 10 10",
		Children: []any{
			&svgutil.Group{Children: elems},
		},
	}
	b, err := svgutil.MarshalSVG(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if len(b) == 0 {
		t.Error("empty SVG output")
	}
}
