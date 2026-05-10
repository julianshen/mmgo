package mindmap

import (
	"encoding/xml"

	"github.com/julianshen/mmgo/pkg/renderer/svgutil"
)

type (
	svgDoc  = svgutil.Doc
	rect    = svgutil.Rect
	line    = svgutil.Line
	path    = svgutil.Path
	circle  = svgutil.Circle
	polygon = svgutil.Polygon
	group   = svgutil.Group
	text    = svgutil.Text
)

// rawSVG holds pre-formatted SVG markup that should be inserted verbatim.
// Deprecated: use github.com/julianshen/mmgo/pkg/renderer/text.ParseMathSVG
// instead to avoid XML escaping issues.
type rawSVG struct {
	Content string
}

func (r rawSVG) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return e.EncodeToken(xml.CharData(r.Content))
}
