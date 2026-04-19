package er

import (
	"math"
	"sort"
	"strings"
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
)

func TestClipToRectEdge(t *testing.T) {
	cases := []struct {
		name             string
		cx, cy, w, h     float64
		ox, oy           float64
		wantX, wantY     float64
	}{
		{"east", 0, 0, 10, 6, 100, 0, 5, 0},
		{"west", 0, 0, 10, 6, -100, 0, -5, 0},
		{"north", 0, 0, 10, 6, 0, -100, 0, -3},
		{"south", 0, 0, 10, 6, 0, 100, 0, 3},
		{"NE-w-limited", 0, 0, 10, 100, 50, 50, 5, 5},
		{"NE-h-limited", 0, 0, 100, 10, 50, 50, 5, 5},
		{"coincident", 3, 4, 10, 6, 3, 4, 3, 4},
		{"interior-clamped", 0, 0, 100, 100, 5, 5, 5, 5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := clipToRectEdge(tc.cx, tc.cy, tc.w, tc.h, tc.ox, tc.oy)
			if math.Abs(got.X-tc.wantX) > 1e-9 || math.Abs(got.Y-tc.wantY) > 1e-9 {
				t.Errorf("clipToRectEdge=(%v,%v) want=(%v,%v)", got.X, got.Y, tc.wantX, tc.wantY)
			}
		})
	}
}

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

// Only end markers are emitted via <marker> defs; start markers are
// inlined as <g> groups and don't go through buildERMarkers.
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

