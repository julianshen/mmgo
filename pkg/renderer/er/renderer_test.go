package er

import (
	"bytes"
	"encoding/xml"
	"strings"
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
)

func TestRenderNilDiagram(t *testing.T) {
	_, err := Render(nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRenderEmptyDiagram(t *testing.T) {
	d := &diagram.ERDiagram{}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	assertValidSVG(t, out)
}

func TestRenderEntityWithAttributes(t *testing.T) {
	d := &diagram.ERDiagram{
		Entities: []diagram.EREntity{
			{
				Name: "CUSTOMER",
				Attributes: []diagram.ERAttribute{
					{Type: "string", Name: "name"},
					{Type: "int", Name: "id", Key: diagram.ERKeyPK},
				},
			},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">CUSTOMER<") {
		t.Error("entity name missing")
	}
	if !strings.Contains(raw, "int id PK") {
		t.Error("attribute with key missing")
	}
	assertValidSVG(t, out)
}

func TestRenderRelationship(t *testing.T) {
	d := &diagram.ERDiagram{
		Entities: []diagram.EREntity{
			{Name: "A"},
			{Name: "B"},
		},
		Relationships: []diagram.ERRelationship{
			{From: "A", To: "B", FromCard: diagram.ERCardExactlyOne, ToCard: diagram.ERCardZeroOrMore, Label: "has"},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">has<") {
		t.Error("relationship label missing")
	}
	assertValidSVG(t, out)
}

func TestRenderRelationshipCardinalityMarkers(t *testing.T) {
	d := &diagram.ERDiagram{
		Entities: []diagram.EREntity{{Name: "A"}, {Name: "B"}},
		Relationships: []diagram.ERRelationship{
			{From: "A", To: "B", FromCard: diagram.ERCardExactlyOne, ToCard: diagram.ERCardZeroOrMore},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	// Only end markers are emitted as <marker> defs; start markers are
	// inlined as <g transform> groups because tdewolff/canvas
	// mis-positions marker-start when marker-end is also present.
	for _, want := range []string{
		`<defs>`,
		`id="er-zeroOrMore-end"`,
		`marker-end="url(#er-zeroOrMore-end)"`,
		`<g transform="translate(`, // inline start marker group
	} {
		if !strings.Contains(raw, want) {
			t.Errorf("missing %q in:\n%s", want, raw)
		}
	}
	for _, unwanted := range []string{
		`marker-start=`,
		`id="er-onlyOne-start"`,
	} {
		if strings.Contains(raw, unwanted) {
			t.Errorf("unwanted substring %q in:\n%s", unwanted, raw)
		}
	}
}

func TestRenderAllCardinalities(t *testing.T) {
	cases := []struct {
		card diagram.ERCardinality
		name string
	}{
		{diagram.ERCardExactlyOne, "onlyOne"},
		{diagram.ERCardZeroOrOne, "zeroOrOne"},
		{diagram.ERCardOneOrMore, "oneOrMore"},
		{diagram.ERCardZeroOrMore, "zeroOrMore"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := &diagram.ERDiagram{
				Entities: []diagram.EREntity{{Name: "A"}, {Name: "B"}},
				Relationships: []diagram.ERRelationship{
					{From: "A", To: "B", FromCard: tc.card, ToCard: tc.card},
				},
			}
			out, _ := Render(d, nil)
			raw := string(out)
			if !strings.Contains(raw, "id=\"er-"+tc.name+"-end\"") {
				t.Errorf("missing end marker def for %s", tc.name)
			}
			// Start markers are inline — just assert the transformed <g>.
			if !strings.Contains(raw, `<g transform="translate(`) {
				t.Errorf("missing inline start marker group for %s", tc.name)
			}
		})
	}
}

func TestRenderMultipleEntities(t *testing.T) {
	d := &diagram.ERDiagram{
		Entities: []diagram.EREntity{
			{Name: "A", Attributes: []diagram.ERAttribute{{Type: "int", Name: "id", Key: diagram.ERKeyPK}}},
			{Name: "B"},
			{Name: "C"},
		},
		Relationships: []diagram.ERRelationship{
			{From: "A", To: "B", FromCard: diagram.ERCardExactlyOne, ToCard: diagram.ERCardZeroOrMore, Label: "has"},
			{From: "B", To: "C", FromCard: diagram.ERCardOneOrMore, ToCard: diagram.ERCardOneOrMore},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">A<") || !strings.Contains(raw, ">B<") || !strings.Contains(raw, ">C<") {
		t.Error("entity names missing")
	}
	assertValidSVG(t, out)
}

func TestRenderDeterministic(t *testing.T) {
	d := &diagram.ERDiagram{
		Entities: []diagram.EREntity{
			{Name: "A", Attributes: []diagram.ERAttribute{{Type: "int", Name: "id"}}},
			{Name: "B"},
		},
		Relationships: []diagram.ERRelationship{
			{From: "A", To: "B", FromCard: diagram.ERCardExactlyOne, ToCard: diagram.ERCardOneOrMore},
		},
	}
	first, _ := Render(d, nil)
	for i := 0; i < 10; i++ {
		next, _ := Render(d, nil)
		if string(next) != string(first) {
			t.Fatalf("iter %d: output diverges", i)
		}
	}
}

func assertValidSVG(t *testing.T, svgBytes []byte) {
	t.Helper()
	body := svgBytes
	if i := bytes.Index(body, []byte("<svg")); i >= 0 {
		body = body[i:]
	}
	var doc struct {
		XMLName xml.Name `xml:"svg"`
		ViewBox string   `xml:"viewBox,attr"`
	}
	if err := xml.Unmarshal(body, &doc); err != nil {
		t.Fatalf("invalid SVG: %v\n%s", err, svgBytes)
	}
	if doc.ViewBox == "" {
		t.Error("viewBox missing")
	}
}
