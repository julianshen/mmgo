package er

import (
	"sort"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/renderer/svgutil"
)

// Slugs match mermaid-cli's marker id naming so SVG diffs against
// mmdc stay readable; ERCardinality.String() returns human-readable
// names ("zero-or-one") and is unsuitable as an id fragment.
func cardSlug(c diagram.ERCardinality) string {
	switch c {
	case diagram.ERCardExactlyOne:
		return "onlyOne"
	case diagram.ERCardZeroOrOne:
		return "zeroOrOne"
	case diagram.ERCardOneOrMore:
		return "oneOrMore"
	case diagram.ERCardZeroOrMore:
		return "zeroOrMore"
	}
	return ""
}

func markerEndID(c diagram.ERCardinality) string {
	if s := cardSlug(c); s != "" {
		return "er-" + s + "-end"
	}
	return ""
}

func markerRef(id string) string {
	if id == "" {
		return ""
	}
	return "url(#" + id + ")"
}

const (
	erMarkerStroke  = "stroke:#333;stroke-width:1;fill:none"
	erOptionalCircle = "fill:#fff;stroke:#333;stroke-width:1"
)

// Start-position cardinality glyphs are inlined under a transformed
// <g> rather than referenced via marker-start. tdewolff/canvas (our
// PNG rasterizer) mis-positions marker-start at canvas (0,0) when
// marker-end is also set on the same element; browsers render the
// inline group identically, so SVG output is consistent across both.
//
// End geometry differs from start geometry by a viewBox shift so the
// glyph hugs refX=markerWidth; see buildERMarkers below.
type startGeom struct {
	children   []any
	refX, refY float64
}

var erStartGeoms = map[diagram.ERCardinality]startGeom{
	diagram.ERCardExactlyOne: {
		children: []any{&path{D: "M9,0 L9,18 M15,0 L15,18", Style: erMarkerStroke}},
		refX:     0, refY: 9,
	},
	diagram.ERCardZeroOrOne: {
		children: []any{
			&circle{CX: 21, CY: 9, R: 6, Style: erOptionalCircle},
			&path{D: "M9,0 L9,18", Style: erMarkerStroke},
		},
		refX: 0, refY: 9,
	},
	diagram.ERCardOneOrMore: {
		children: []any{&path{D: "M0,18 Q18,0 36,18 Q18,36 0,18 M42,9 L42,27", Style: erMarkerStroke}},
		refX:     18, refY: 18,
	},
	diagram.ERCardZeroOrMore: {
		children: []any{
			&circle{CX: 48, CY: 18, R: 6, Style: erOptionalCircle},
			&path{D: "M0,18 Q18,0 36,18 Q18,36 0,18", Style: erMarkerStroke},
		},
		refX: 18, refY: 18,
	},
}

func startMarkerGeom(c diagram.ERCardinality) (children []any, refX, refY float64, ok bool) {
	g, ok := erStartGeoms[c]
	if !ok {
		return nil, 0, 0, false
	}
	return g.children, g.refX, g.refY, true
}

// Sorted ids keep SVG output stable across runs; map iteration alone
// is not deterministic.
func buildERMarkers(d *diagram.ERDiagram) []svgutil.Marker {
	used := map[diagram.ERCardinality]bool{}
	for _, r := range d.Relationships {
		if cardSlug(r.ToCard) != "" {
			used[r.ToCard] = true
		}
	}
	if len(used) == 0 {
		return nil
	}

	endDefs := map[string]svgutil.Marker{
		"er-onlyOne-end": {
			ID: "er-onlyOne-end", ViewBox: "0 0 18 18",
			RefX: 18, RefY: 9, Width: 18, Height: 18, Orient: "auto",
			Children: []any{&path{D: "M3,0 L3,18 M9,0 L9,18", Style: erMarkerStroke}},
		},
		"er-zeroOrOne-end": {
			ID: "er-zeroOrOne-end", ViewBox: "0 0 30 18",
			RefX: 30, RefY: 9, Width: 30, Height: 18, Orient: "auto",
			Children: []any{
				&circle{CX: 9, CY: 9, R: 6, Style: erOptionalCircle},
				&path{D: "M21,0 L21,18", Style: erMarkerStroke},
			},
		},
		"er-oneOrMore-end": {
			ID: "er-oneOrMore-end", ViewBox: "0 0 45 36",
			RefX: 27, RefY: 18, Width: 45, Height: 36, Orient: "auto",
			Children: []any{&path{D: "M3,9 L3,27 M9,18 Q27,0 45,18 Q27,36 9,18", Style: erMarkerStroke}},
		},
		"er-zeroOrMore-end": {
			ID: "er-zeroOrMore-end", ViewBox: "0 0 57 36",
			RefX: 39, RefY: 18, Width: 57, Height: 36, Orient: "auto",
			Children: []any{
				&circle{CX: 9, CY: 18, R: 6, Style: erOptionalCircle},
				&path{D: "M21,18 Q39,0 57,18 Q39,36 21,18", Style: erMarkerStroke},
			},
		},
	}

	ids := make([]string, 0, len(used))
	for c := range used {
		ids = append(ids, "er-"+cardSlug(c)+"-end")
	}
	sort.Strings(ids)
	out := make([]svgutil.Marker, 0, len(ids))
	for _, id := range ids {
		out = append(out, endDefs[id])
	}
	return out
}
