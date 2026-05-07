package sankey

import (
	"encoding/xml"

	"github.com/julianshen/mmgo/pkg/renderer/svgutil"
)

type (
	svgFloat = svgutil.Float
	svgDoc   = svgutil.Doc
	rect     = svgutil.Rect
	path     = svgutil.Path
	text     = svgutil.Text
)

// sankeyDefs is a local <defs> wrapper that holds linear-gradient
// children. The shared svgutil.Defs only carries marker entries
// today — gradient support is sankey-specific so the type lives
// here until another renderer needs it.
type sankeyDefs struct {
	XMLName  xml.Name `xml:"defs"`
	Children []any    `xml:",any"`
}

// linearGradient maps to <linearGradient id="…" x1="…" y1="…" x2="…"
// y2="…"><stop … /></linearGradient>. Sankey uses it for the
// `linkColor: gradient` fill mode where each ribbon transitions
// from its source-node color to its target-node color.
type linearGradient struct {
	XMLName xml.Name       `xml:"linearGradient"`
	ID      string         `xml:"id,attr"`
	X1      string         `xml:"x1,attr"`
	Y1      string         `xml:"y1,attr"`
	X2      string         `xml:"x2,attr"`
	Y2      string         `xml:"y2,attr"`
	Stops   []gradientStop `xml:"stop"`
}

type gradientStop struct {
	XMLName   xml.Name `xml:"stop"`
	Offset    string   `xml:"offset,attr"`
	StopColor string   `xml:"stop-color,attr"`
}
