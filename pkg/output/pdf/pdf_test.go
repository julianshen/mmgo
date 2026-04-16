package pdf

import (
	"bytes"
	"strings"
	"testing"

	svgpkg "github.com/julianshen/mmgo/pkg/output/svg"
)

func renderTestSVG(t *testing.T) []byte {
	t.Helper()
	out, err := svgpkg.Render(strings.NewReader("graph LR\n    A[Start] --> B[End]"), nil)
	if err != nil {
		t.Fatalf("svg render: %v", err)
	}
	return out
}

func TestRenderNilInput(t *testing.T) {
	_, err := Render(nil, nil)
	if err == nil {
		t.Fatal("expected error for nil input")
	}
}

func TestRenderEmptyInput(t *testing.T) {
	_, err := Render([]byte{}, nil)
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestRenderInvalidSVG(t *testing.T) {
	_, err := Render([]byte("not valid svg at all"), nil)
	if err == nil {
		t.Fatal("expected error for invalid SVG")
	}
}

func TestRenderProducesValidPDF(t *testing.T) {
	svgBytes := renderTestSVG(t)
	pdfBytes, err := Render(svgBytes, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if len(pdfBytes) == 0 {
		t.Fatal("output should not be empty")
	}
	if !bytes.HasPrefix(pdfBytes, []byte("%PDF")) {
		t.Error("output should start with PDF magic bytes")
	}
}

func TestRenderDeterministic(t *testing.T) {
	svgBytes := renderTestSVG(t)
	first, err := Render(svgBytes, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	for i := 0; i < 5; i++ {
		next, err := Render(svgBytes, nil)
		if err != nil {
			t.Fatalf("iter %d: %v", i, err)
		}
		if !bytes.Equal(next, first) {
			t.Fatalf("iter %d: output diverges", i)
		}
	}
}

func TestRenderSequenceDiagram(t *testing.T) {
	svgBytes, err := svgpkg.Render(strings.NewReader("sequenceDiagram\n    A->>B: hello"), nil)
	if err != nil {
		t.Fatalf("svg: %v", err)
	}
	pdfBytes, err := Render(svgBytes, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !bytes.HasPrefix(pdfBytes, []byte("%PDF")) {
		t.Error("should produce valid PDF")
	}
}

func TestRenderPieDiagram(t *testing.T) {
	svgBytes, err := svgpkg.Render(strings.NewReader("pie title Pets\n    \"Dogs\" : 70\n    \"Cats\" : 30"), nil)
	if err != nil {
		t.Fatalf("svg: %v", err)
	}
	pdfBytes, err := Render(svgBytes, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !bytes.HasPrefix(pdfBytes, []byte("%PDF")) {
		t.Error("should produce valid PDF")
	}
}
