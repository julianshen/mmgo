package timeline

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
	d := &diagram.TimelineDiagram{}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	assertValidSVG(t, out)
}

func TestRenderWithTitle(t *testing.T) {
	d := &diagram.TimelineDiagram{Title: "My History"}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(string(out), ">My History<") {
		t.Error("title missing")
	}
	assertValidSVG(t, out)
}

func TestRenderTopLevelEvents(t *testing.T) {
	d := &diagram.TimelineDiagram{
		Events: []diagram.TimelineEvent{
			{Time: "1990", Events: []string{"Born"}},
			{Time: "2020", Events: []string{"Graduated", "Moved"}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">1990<") || !strings.Contains(raw, ">2020<") {
		t.Error("time labels missing")
	}
	if !strings.Contains(raw, "Born") {
		t.Error("event text missing")
	}
	assertValidSVG(t, out)
}

func TestRenderSections(t *testing.T) {
	d := &diagram.TimelineDiagram{
		Title: "Milestones",
		Sections: []diagram.TimelineSection{
			{Name: "2020s", Events: []diagram.TimelineEvent{
				{Time: "2020", Events: []string{"A"}},
			}},
			{Name: "2030s", Events: []diagram.TimelineEvent{
				{Time: "2030", Events: []string{"B"}},
			}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">2020s<") || !strings.Contains(raw, ">2030s<") {
		t.Error("section names missing")
	}
	assertValidSVG(t, out)
}

func TestRenderDeterministic(t *testing.T) {
	d := &diagram.TimelineDiagram{
		Title: "Test",
		Events: []diagram.TimelineEvent{
			{Time: "A", Events: []string{"x"}},
			{Time: "B", Events: []string{"y"}},
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
