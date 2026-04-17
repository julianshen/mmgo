package state

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
	d := &diagram.StateDiagram{}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	assertValidSVG(t, out)
}

func TestRenderSimpleStates(t *testing.T) {
	d := &diagram.StateDiagram{
		States: []diagram.StateDef{
			{ID: "Idle", Label: "Idle"},
			{ID: "Active", Label: "Active"},
		},
		Transitions: []diagram.StateTransition{
			{From: "Idle", To: "Active", Label: "start"},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">Idle<") || !strings.Contains(raw, ">Active<") {
		t.Error("state labels missing")
	}
	if !strings.Contains(raw, ">start<") {
		t.Error("transition label missing")
	}
	assertValidSVG(t, out)
}

func TestRenderStartEndStates(t *testing.T) {
	d := &diagram.StateDiagram{
		States: []diagram.StateDef{
			{ID: "Running", Label: "Running"},
		},
		Transitions: []diagram.StateTransition{
			{From: "[*]", To: "Running"},
			{From: "Running", To: "[*]"},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "<circle") {
		t.Error("expected circle elements for [*] pseudo-states")
	}
	assertValidSVG(t, out)
}

func TestRenderSpecialStates(t *testing.T) {
	for _, tc := range []struct {
		kind diagram.StateKind
		name string
	}{
		{diagram.StateKindFork, "fork"},
		{diagram.StateKindJoin, "join"},
		{diagram.StateKindChoice, "choice"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := &diagram.StateDiagram{
				States: []diagram.StateDef{
					{ID: "s1", Label: "s1", Kind: tc.kind},
					{ID: "A", Label: "A"},
				},
				Transitions: []diagram.StateTransition{
					{From: "s1", To: "A"},
				},
			}
			out, err := Render(d, nil)
			if err != nil {
				t.Fatalf("Render: %v", err)
			}
			assertValidSVG(t, out)
		})
	}
}

func TestRenderCompositeState(t *testing.T) {
	d := &diagram.StateDiagram{
		States: []diagram.StateDef{
			{
				ID: "Active", Label: "Active",
				Children: []diagram.StateDef{
					{ID: "Running", Label: "Running"},
					{ID: "Paused", Label: "Paused"},
				},
			},
		},
		Transitions: []diagram.StateTransition{
			{From: "Running", To: "Paused"},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">Active<") {
		t.Error("composite label missing")
	}
	assertValidSVG(t, out)
}

func TestRenderDeterministic(t *testing.T) {
	d := &diagram.StateDiagram{
		States: []diagram.StateDef{
			{ID: "A", Label: "A"},
			{ID: "B", Label: "B"},
		},
		Transitions: []diagram.StateTransition{
			{From: "A", To: "B"},
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
