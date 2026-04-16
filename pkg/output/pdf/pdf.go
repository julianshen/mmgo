// Package pdf converts SVG bytes to PDF using vector rendering.
package pdf

import (
	"bytes"
	"fmt"

	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers"
)

type Options struct{}

func Render(svgBytes []byte, opts *Options) ([]byte, error) {
	if len(svgBytes) == 0 {
		return nil, fmt.Errorf("pdf render: empty SVG input")
	}

	c, err := canvas.ParseSVG(bytes.NewReader(svgBytes))
	if err != nil {
		return nil, fmt.Errorf("pdf render: parse SVG: %w", err)
	}

	var buf bytes.Buffer
	if err := c.Write(&buf, renderers.PDF()); err != nil {
		return nil, fmt.Errorf("pdf render: write: %w", err)
	}
	return buf.Bytes(), nil
}
