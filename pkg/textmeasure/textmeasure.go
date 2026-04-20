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
//
// Ruler is not safe for concurrent use.
package textmeasure

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
)

// avgCharWidth backs EstimateWidth: approximate per-character advance
// as a fraction of font size, tuned for the bundled Go Regular font.
const avgCharWidth = 0.6

// EstimateWidth returns a cheap approximate rendered width of s at
// the given font size. Use it for layout pre-passes where exact
// metrics aren't critical; use Ruler.Measure for accurate widths,
// especially with non-ASCII text or proportional-font output.
//
// Non-positive fontSize returns 0, matching Ruler.Measure's contract
// so callers don't need divergent guards.
func EstimateWidth(s string, fontSize float64) float64 {
	if fontSize <= 0 {
		return 0
	}
	return float64(utf8.RuneCountInString(s)) * fontSize * avgCharWidth
}

// Ruler measures text dimensions using font metrics. It caches font faces
// per size for efficiency in layout hot paths.
type Ruler struct {
	font  *sfnt.Font
	faces map[float64]cachedFace
}

type cachedFace struct {
	face       font.Face
	lineHeight float64
}

// NewRuler creates a Ruler from TrueType or OpenType font bytes.
func NewRuler(fontData []byte) (*Ruler, error) {
	f, err := sfnt.Parse(fontData)
	if err != nil {
		return nil, fmt.Errorf("parsing font: %w", err)
	}
	return &Ruler{
		font:  f,
		faces: make(map[float64]cachedFace),
	}, nil
}

// NewDefaultRuler creates a Ruler using the bundled Go Regular font.
func NewDefaultRuler() (*Ruler, error) {
	return NewRuler(goregular.TTF)
}

// Close releases font faces cached by the Ruler. After Close, the Ruler
// must not be used.
func (r *Ruler) Close() error {
	for _, cf := range r.faces {
		_ = cf.face.Close()
	}
	r.faces = nil
	return nil
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

	cf, err := r.faceFor(fontSize)
	if err != nil {
		return 0, 0
	}

	// Single-line fast path avoids substring iteration overhead.
	if !strings.ContainsRune(text, '\n') {
		w := float64(font.MeasureString(cf.face, text)) / 64
		return w, cf.lineHeight
	}

	var maxWidth float64
	lineCount := 0
	rest := text
	for {
		lineCount++
		var line string
		if i := strings.IndexByte(rest, '\n'); i >= 0 {
			line, rest = rest[:i], rest[i+1:]
		} else {
			line = rest
			w := float64(font.MeasureString(cf.face, line)) / 64
			if w > maxWidth {
				maxWidth = w
			}
			break
		}
		w := float64(font.MeasureString(cf.face, line)) / 64
		if w > maxWidth {
			maxWidth = w
		}
	}

	return maxWidth, cf.lineHeight * float64(lineCount)
}

// faceFor returns a cached font face for the given size, creating one
// on first use.
func (r *Ruler) faceFor(fontSize float64) (cachedFace, error) {
	if cf, ok := r.faces[fontSize]; ok {
		return cf, nil
	}
	face, err := opentype.NewFace(r.font, &opentype.FaceOptions{
		Size: fontSize,
		DPI:  72,
	})
	if err != nil {
		return cachedFace{}, fmt.Errorf("creating font face: %w", err)
	}
	cf := cachedFace{
		face:       face,
		lineHeight: float64(face.Metrics().Height) / 64,
	}
	r.faces[fontSize] = cf
	return cf, nil
}
