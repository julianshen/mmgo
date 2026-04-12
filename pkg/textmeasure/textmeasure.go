// Package textmeasure computes text bounding boxes from font metrics.
//
// It replaces the browser's SVGTextElement.getBBox() API that Mermaid.js
// relies on. The layout engine needs accurate text dimensions to size
// nodes and lay out edges; a 10% error in text width cascades into
// overlapping labels and excessive whitespace.
//
// The default font is Go Regular (golang.org/x/image/font/gofont/goregular),
// a BSD-3-Clause licensed font shipped as a Go package. Callers may supply
// any TrueType/OpenType font bytes via NewRuler.
package textmeasure

import (
	"fmt"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
)

// Ruler measures text dimensions using font metrics.
type Ruler struct {
	font *sfnt.Font
}

// NewRuler creates a Ruler from TrueType or OpenType font bytes.
func NewRuler(fontData []byte) (*Ruler, error) {
	f, err := sfnt.Parse(fontData)
	if err != nil {
		return nil, fmt.Errorf("parsing font: %w", err)
	}
	return &Ruler{font: f}, nil
}

// NewDefaultRuler creates a Ruler using the bundled Go Regular font.
func NewDefaultRuler() (*Ruler, error) {
	return NewRuler(goregular.TTF)
}

// Measure returns the pixel width and height of text rendered at the given
// font size in points. Multi-line text (separated by \n) returns the maximum
// line width and the sum of line heights.
//
// Returns (0, 0) for empty text or non-positive font size.
func (r *Ruler) Measure(text string, fontSize float64) (width, height float64) {
	if text == "" || fontSize <= 0 {
		return 0, 0
	}

	face, err := opentype.NewFace(r.font, &opentype.FaceOptions{
		Size: fontSize,
		DPI:  72,
	})
	if err != nil {
		return 0, 0
	}
	defer face.Close()

	lineHeight := float64(face.Metrics().Height) / 64

	lines := strings.Split(text, "\n")
	var maxWidth float64
	for _, line := range lines {
		w := float64(font.MeasureString(face, line)) / 64
		if w > maxWidth {
			maxWidth = w
		}
	}

	return maxWidth, lineHeight * float64(len(lines))
}
