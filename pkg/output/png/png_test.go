package png

import (
	"bytes"
	"image"
	imagepng "image/png"
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

func TestRenderProducesValidPNG(t *testing.T) {
	svgBytes := renderTestSVG(t)
	pngBytes, err := Render(svgBytes, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if len(pngBytes) == 0 {
		t.Fatal("output should not be empty")
	}
	img := decodePNG(t, pngBytes)
	bounds := img.Bounds()
	if bounds.Dx() == 0 || bounds.Dy() == 0 {
		t.Errorf("image should have non-zero dimensions: %v", bounds)
	}
}

func TestRenderWithScale(t *testing.T) {
	svgBytes := renderTestSVG(t)
	png1x, err := Render(svgBytes, &Options{Scale: 1})
	if err != nil {
		t.Fatalf("1x: %v", err)
	}
	png2x, err := Render(svgBytes, &Options{Scale: 2})
	if err != nil {
		t.Fatalf("2x: %v", err)
	}
	img1 := decodePNG(t, png1x)
	img2 := decodePNG(t, png2x)
	if img2.Bounds().Dx() <= img1.Bounds().Dx() {
		t.Errorf("2x should be wider: %d vs %d", img2.Bounds().Dx(), img1.Bounds().Dx())
	}
}

func TestRenderWithFixedDimensions(t *testing.T) {
	svgBytes := renderTestSVG(t)
	pngBytes, err := Render(svgBytes, &Options{Width: 800, Height: 600})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	img := decodePNG(t, pngBytes)
	if img.Bounds().Dx() != 800 || img.Bounds().Dy() != 600 {
		t.Errorf("dimensions = %dx%d, want 800x600", img.Bounds().Dx(), img.Bounds().Dy())
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

func decodePNG(t *testing.T, data []byte) image.Image {
	t.Helper()
	img, err := imagepng.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("invalid PNG: %v", err)
	}
	return img
}
