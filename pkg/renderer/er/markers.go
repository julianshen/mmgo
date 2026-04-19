package er

import (
	"sort"

	"github.com/julianshen/mmgo/pkg/diagram"
	"github.com/julianshen/mmgo/pkg/renderer/svgutil"
)

// Cardinality slug used in marker IDs. Distinct from ERCardinality.String(),
// which returns human-readable names ("zero-or-one"); these slugs match
// mermaid-cli's marker id naming so SVG diffs against mmdc stay readable.
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

func markerStartID(c diagram.ERCardinality) string {
	if s := cardSlug(c); s != "" {
		return "er-" + s + "-start"
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

// buildERMarkers emits marker defs only for cardinalities the diagram
// actually uses, in deterministic order. Mirrors the flowchart pattern
// in pkg/renderer/flowchart/edges.go so empty/single-cardinality
// diagrams don't carry the full eight defs in their SVG output.
func buildERMarkers(d *diagram.ERDiagram) []svgutil.Marker {
	const stroke = "stroke:#333;stroke-width:1;fill:none"
	const optionalFill = "fill:#fff;stroke:#333;stroke-width:1"

	type key struct {
		card diagram.ERCardinality
		pos  string
	}
	used := map[key]bool{}
	for _, r := range d.Relationships {
		if cardSlug(r.FromCard) != "" {
			used[key{r.FromCard, "start"}] = true
		}
		if cardSlug(r.ToCard) != "" {
			used[key{r.ToCard, "end"}] = true
		}
	}
	if len(used) == 0 {
		return nil
	}

	defs := map[string]svgutil.Marker{
		"er-onlyOne-start": {
			ID: "er-onlyOne-start", ViewBox: "0 0 18 18",
			RefX: 0, RefY: 9, Width: 18, Height: 18, Orient: "auto",
			Children: []any{&path{D: "M9,0 L9,18 M15,0 L15,18", Style: stroke}},
		},
		"er-onlyOne-end": {
			ID: "er-onlyOne-end", ViewBox: "0 0 18 18",
			RefX: 18, RefY: 9, Width: 18, Height: 18, Orient: "auto",
			Children: []any{&path{D: "M3,0 L3,18 M9,0 L9,18", Style: stroke}},
		},
		"er-zeroOrOne-start": {
			ID: "er-zeroOrOne-start", ViewBox: "0 0 30 18",
			RefX: 0, RefY: 9, Width: 30, Height: 18, Orient: "auto",
			Children: []any{
				&circle{CX: 21, CY: 9, R: 6, Style: optionalFill},
				&path{D: "M9,0 L9,18", Style: stroke},
			},
		},
		"er-zeroOrOne-end": {
			ID: "er-zeroOrOne-end", ViewBox: "0 0 30 18",
			RefX: 30, RefY: 9, Width: 30, Height: 18, Orient: "auto",
			Children: []any{
				&circle{CX: 9, CY: 9, R: 6, Style: optionalFill},
				&path{D: "M21,0 L21,18", Style: stroke},
			},
		},
		"er-oneOrMore-start": {
			ID: "er-oneOrMore-start", ViewBox: "0 0 45 36",
			RefX: 18, RefY: 18, Width: 45, Height: 36, Orient: "auto",
			Children: []any{&path{D: "M0,18 Q18,0 36,18 Q18,36 0,18 M42,9 L42,27", Style: stroke}},
		},
		"er-oneOrMore-end": {
			ID: "er-oneOrMore-end", ViewBox: "0 0 45 36",
			RefX: 27, RefY: 18, Width: 45, Height: 36, Orient: "auto",
			Children: []any{&path{D: "M3,9 L3,27 M9,18 Q27,0 45,18 Q27,36 9,18", Style: stroke}},
		},
		"er-zeroOrMore-start": {
			ID: "er-zeroOrMore-start", ViewBox: "0 0 57 36",
			RefX: 18, RefY: 18, Width: 57, Height: 36, Orient: "auto",
			Children: []any{
				&circle{CX: 48, CY: 18, R: 6, Style: optionalFill},
				&path{D: "M0,18 Q18,0 36,18 Q18,36 0,18", Style: stroke},
			},
		},
		"er-zeroOrMore-end": {
			ID: "er-zeroOrMore-end", ViewBox: "0 0 57 36",
			RefX: 39, RefY: 18, Width: 57, Height: 36, Orient: "auto",
			Children: []any{
				&circle{CX: 9, CY: 18, R: 6, Style: optionalFill},
				&path{D: "M21,18 Q39,0 57,18 Q39,36 21,18", Style: stroke},
			},
		},
	}

	ids := make([]string, 0, len(used))
	for k := range used {
		ids = append(ids, "er-"+cardSlug(k.card)+"-"+k.pos)
	}
	sort.Strings(ids)
	out := make([]svgutil.Marker, 0, len(ids))
	for _, id := range ids {
		out = append(out, defs[id])
	}
	return out
}
