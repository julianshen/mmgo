// Package png rasterizes SVG bytes to PNG.
package png

import (
	"bytes"
	"fmt"
	"image"
	imagepng "image/png"

	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers/rasterizer"
	xdraw "golang.org/x/image/draw"
)

type Options struct {
	Scale  float64
	Width  int
	Height int
}

func Render(svgBytes []byte, opts *Options) ([]byte, error) {
	if len(svgBytes) == 0 {
		return nil, fmt.Errorf("png render: empty SVG input")
	}

	c, err := canvas.ParseSVG(bytes.NewReader(svgBytes))
	if err != nil {
		return nil, fmt.Errorf("png render: parse SVG: %w", err)
	}

	scale := 1.0
	if opts != nil && opts.Scale > 0 {
		scale = opts.Scale
	}

	cw, _ := c.Size()
	if cw <= 0 {
		cw = 1
	}

	dpi := canvas.DPI(96 * scale)
	if opts != nil && opts.Width > 0 && opts.Height > 0 {
		dpi = canvas.DPI(float64(opts.Width) / cw * 96)
	}

	img := rasterizer.Draw(c, dpi, canvas.DefaultColorSpace)

	var out image.Image = img
	if opts != nil && opts.Width > 0 && opts.Height > 0 &&
		(img.Bounds().Dx() != opts.Width || img.Bounds().Dy() != opts.Height) {
		dst := image.NewRGBA(image.Rect(0, 0, opts.Width, opts.Height))
		// NearestNeighbor preserves sharp edges in diagram line art.
		xdraw.NearestNeighbor.Scale(dst, dst.Bounds(), img, img.Bounds(), xdraw.Src, nil)
		img = nil // free the original before encoding
		out = dst
	}

	var buf bytes.Buffer
	if err := imagepng.Encode(&buf, out); err != nil {
		return nil, fmt.Errorf("png render: encode: %w", err)
	}
	return buf.Bytes(), nil
}
