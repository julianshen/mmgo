package er

import (
	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/renderer/svgutil"
)

// Crow's-foot marker geometry mirrors mermaid-cli's er marker defs:
// each cardinality renders two SVG markers (start/end) so the glyph
// faces away from the line in either direction. orient="auto" rotates
// the marker along the path tangent; refX positions it so the
// notation reads "outside" the relationship line at each entity edge.

const (
	cardNameOnlyOne    = "onlyOne"
	cardNameZeroOrOne  = "zeroOrOne"
	cardNameOneOrMore  = "oneOrMore"
	cardNameZeroOrMore = "zeroOrMore"
)

func cardName(c diagram.ERCardinality) string {
	switch c {
	case diagram.ERCardExactlyOne:
		return cardNameOnlyOne
	case diagram.ERCardZeroOrOne:
		return cardNameZeroOrOne
	case diagram.ERCardOneOrMore:
		return cardNameOneOrMore
	case diagram.ERCardZeroOrMore:
		return cardNameZeroOrMore
	}
	return ""
}

// markerID returns the def id for a cardinality at one path endpoint.
// pos is "start" or "end". Returns "" when the cardinality is unknown.
func markerID(c diagram.ERCardinality, pos string) string {
	name := cardName(c)
	if name == "" {
		return ""
	}
	return "er-" + name + "-" + pos
}

// markerRef wraps an id as the SVG marker-start/marker-end attribute
// value. Empty id yields empty string so the attribute is omitted.
func markerRef(id string) string {
	if id == "" {
		return ""
	}
	return "url(#" + id + ")"
}

// buildERMarkers returns the eight marker defs (4 cardinalities × 2
// endpoints). Geometry is the same as mermaid-cli's er markers, drawn
// with stroke=#333 and white fill on the "optional" circle so the
// relationship line is visually broken at the zero-or-X glyph.
func buildERMarkers() []svgutil.Marker {
	const stroke = "stroke:#333;stroke-width:1;fill:none"
	const optionalFill = "fill:#fff;stroke:#333;stroke-width:1"

	return []svgutil.Marker{
		// onlyOne: two parallel bars (||).
		{
			ID: markerID(diagram.ERCardExactlyOne, "start"), ViewBox: "0 0 18 18",
			RefX: 0, RefY: 9, Width: 18, Height: 18, Orient: "auto",
			Children: []any{&path{D: "M9,0 L9,18 M15,0 L15,18", Style: stroke}},
		},
		{
			ID: markerID(diagram.ERCardExactlyOne, "end"), ViewBox: "0 0 18 18",
			RefX: 18, RefY: 9, Width: 18, Height: 18, Orient: "auto",
			Children: []any{&path{D: "M3,0 L3,18 M9,0 L9,18", Style: stroke}},
		},

		// zeroOrOne: bar + open circle (|o or o|).
		{
			ID: markerID(diagram.ERCardZeroOrOne, "start"), ViewBox: "0 0 30 18",
			RefX: 0, RefY: 9, Width: 30, Height: 18, Orient: "auto",
			Children: []any{
				&circle{CX: 21, CY: 9, R: 6, Style: optionalFill},
				&path{D: "M9,0 L9,18", Style: stroke},
			},
		},
		{
			ID: markerID(diagram.ERCardZeroOrOne, "end"), ViewBox: "0 0 30 18",
			RefX: 30, RefY: 9, Width: 30, Height: 18, Orient: "auto",
			Children: []any{
				&circle{CX: 9, CY: 9, R: 6, Style: optionalFill},
				&path{D: "M21,0 L21,18", Style: stroke},
			},
		},

		// oneOrMore: crow's-foot + bar.
		{
			ID: markerID(diagram.ERCardOneOrMore, "start"), ViewBox: "0 0 45 36",
			RefX: 18, RefY: 18, Width: 45, Height: 36, Orient: "auto",
			Children: []any{&path{D: "M0,18 Q18,0 36,18 Q18,36 0,18 M42,9 L42,27", Style: stroke}},
		},
		{
			ID: markerID(diagram.ERCardOneOrMore, "end"), ViewBox: "0 0 45 36",
			RefX: 27, RefY: 18, Width: 45, Height: 36, Orient: "auto",
			Children: []any{&path{D: "M3,9 L3,27 M9,18 Q27,0 45,18 Q27,36 9,18", Style: stroke}},
		},

		// zeroOrMore: crow's-foot + open circle.
		{
			ID: markerID(diagram.ERCardZeroOrMore, "start"), ViewBox: "0 0 57 36",
			RefX: 18, RefY: 18, Width: 57, Height: 36, Orient: "auto",
			Children: []any{
				&circle{CX: 48, CY: 18, R: 6, Style: optionalFill},
				&path{D: "M0,18 Q18,0 36,18 Q18,36 0,18", Style: stroke},
			},
		},
		{
			ID: markerID(diagram.ERCardZeroOrMore, "end"), ViewBox: "0 0 57 36",
			RefX: 39, RefY: 18, Width: 57, Height: 36, Orient: "auto",
			Children: []any{
				&circle{CX: 9, CY: 18, R: 6, Style: optionalFill},
				&path{D: "M21,18 Q39,0 57,18 Q39,36 21,18", Style: stroke},
			},
		},
	}
}
