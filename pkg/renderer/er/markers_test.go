package er

import (
	"sort"
	"strings"
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
)

func TestMarkerRefEmpty(t *testing.T) {
	if got := markerRef(""); got != "" {
		t.Errorf("markerRef(\"\") = %q, want empty", got)
	}
	if got := markerRef("foo"); got != "url(#foo)" {
		t.Errorf("markerRef(\"foo\") = %q", got)
	}
}

func TestCardSlug(t *testing.T) {
	cases := map[diagram.ERCardinality]string{
		diagram.ERCardExactlyOne: "onlyOne",
		diagram.ERCardZeroOrOne:  "zeroOrOne",
		diagram.ERCardOneOrMore:  "oneOrMore",
		diagram.ERCardZeroOrMore: "zeroOrMore",
		diagram.ERCardUnknown:    "",
	}
	for c, want := range cases {
		if got := cardSlug(c); got != want {
			t.Errorf("cardSlug(%v) = %q, want %q", c, got, want)
		}
	}
}

func TestBuildERMarkersEmptyDiagram(t *testing.T) {
	if got := buildERMarkers(&diagram.ERDiagram{}); got != nil {
		t.Errorf("empty diagram should yield no marker defs, got %d", len(got))
	}
}

func TestBuildERMarkersUnknownCardinalitySkipped(t *testing.T) {
	d := &diagram.ERDiagram{
		Relationships: []diagram.ERRelationship{
			{From: "A", To: "B"}, // both cards = ERCardUnknown
		},
	}
	if got := buildERMarkers(d); got != nil {
		t.Errorf("unknown-cardinality relationship should yield no marker defs, got %d", len(got))
	}
}

func TestBuildERMarkersDeterministicOrder(t *testing.T) {
	d := &diagram.ERDiagram{
		Relationships: []diagram.ERRelationship{
			{From: "A", To: "B", FromCard: diagram.ERCardOneOrMore, ToCard: diagram.ERCardZeroOrOne},
			{From: "B", To: "C", FromCard: diagram.ERCardExactlyOne, ToCard: diagram.ERCardZeroOrMore},
		},
	}
	markers := buildERMarkers(d)
	ids := make([]string, len(markers))
	for i, m := range markers {
		ids[i] = m.ID
	}
	if !sort.StringsAreSorted(ids) {
		t.Errorf("marker IDs not sorted: %v", ids)
	}
}

func TestBuildERMarkersFiltersToUsed(t *testing.T) {
	d := &diagram.ERDiagram{
		Relationships: []diagram.ERRelationship{
			{From: "A", To: "B", FromCard: diagram.ERCardExactlyOne, ToCard: diagram.ERCardZeroOrMore},
		},
	}
	markers := buildERMarkers(d)
	got := map[string]bool{}
	for _, m := range markers {
		got[m.ID] = true
	}
	if !got["er-zeroOrMore-end"] {
		t.Errorf("missing marker def er-zeroOrMore-end; got: %v", got)
	}
	for _, unwanted := range []string{
		"er-onlyOne-end", "er-zeroOrOne-end", "er-oneOrMore-end",
		"er-onlyOne-start", "er-zeroOrMore-start",
		"er-zeroOrOne-start", "er-oneOrMore-start",
	} {
		if got[unwanted] {
			t.Errorf("unexpected marker def %q emitted", unwanted)
		}
	}
}

// Pins the 2-point edge aliasing fix: pts[0] and pts[len-2] alias for
// 2-point edges, so srcDir/dstDir must be cached before mutating
// either endpoint or the dst clip reads the already-clipped src.
func TestRenderEdgeClipsBothEndpointsIndependently(t *testing.T) {
	d := &diagram.ERDiagram{
		Entities: []diagram.EREntity{{Name: "A"}, {Name: "B"}},
		Relationships: []diagram.ERRelationship{
			{From: "A", To: "B", FromCard: diagram.ERCardExactlyOne, ToCard: diagram.ERCardExactlyOne},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, `<g transform="translate(`) {
		t.Error("missing inline start marker group")
	}
	if !strings.Contains(raw, `marker-end="url(#er-onlyOne-end)"`) {
		t.Error("missing end marker")
	}
	if !hasNonDegenerateLine(raw) {
		t.Errorf("expected non-degenerate relationship line in:\n%s", raw)
	}
}

// hasNonDegenerateLine is a coarse check that some <line> with
// distinct x1/x2 or y1/y2 exists in the SVG output.
func hasNonDegenerateLine(svg string) bool {
	const lineTag = "<line "
	for i := 0; ; {
		idx := strings.Index(svg[i:], lineTag)
		if idx < 0 {
			return false
		}
		i += idx
		end := strings.Index(svg[i:], ">")
		if end < 0 {
			return false
		}
		tag := svg[i : i+end]
		x1 := attrVal(tag, "x1")
		x2 := attrVal(tag, "x2")
		y1 := attrVal(tag, "y1")
		y2 := attrVal(tag, "y2")
		if x1 != "" && (x1 != x2 || y1 != y2) {
			return true
		}
		i += end
	}
}

func attrVal(tag, name string) string {
	key := name + `="`
	idx := strings.Index(tag, key)
	if idx < 0 {
		return ""
	}
	idx += len(key)
	end := strings.Index(tag[idx:], `"`)
	if end < 0 {
		return ""
	}
	return tag[idx : idx+end]
}

