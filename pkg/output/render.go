// Package output provides shared output format conversion.
package output

import (
	"fmt"

	pdfpkg "github.com/julianshen/mmgo/pkg/output/pdf"
	pngpkg "github.com/julianshen/mmgo/pkg/output/png"
)

// ConvertSVG converts SVG bytes to the requested format.
// Supported formats: "svg" (passthrough), "png", "pdf".
func ConvertSVG(svgBytes []byte, format string) ([]byte, error) {
	switch format {
	case "svg", "":
		return svgBytes, nil
	case "png":
		return pngpkg.Render(svgBytes, nil)
	case "pdf":
		return pdfpkg.Render(svgBytes, nil)
	default:
		return nil, fmt.Errorf("unsupported output format: %q", format)
	}
}
