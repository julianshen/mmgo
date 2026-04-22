package gitgraph

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
)

func TestRenderNilDiagram(t *testing.T) {
	if _, err := Render(nil, nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestRenderEmptyDiagram(t *testing.T) {
	d := &diagram.GitGraphDiagram{}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	assertValidSVG(t, out)
}

func TestRenderSingleCommit(t *testing.T) {
	d := &diagram.GitGraphDiagram{
		Branches: []string{"main"},
		Commits: []diagram.GitCommit{
			{ID: "c1", Branch: "main", Type: diagram.GitCommitNormal},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">c1<") {
		t.Error("commit ID label missing")
	}
	if !strings.Contains(raw, ">main<") {
		t.Error("branch label missing")
	}
	assertValidSVG(t, out)
}

func TestRenderLinearCommits(t *testing.T) {
	d := &diagram.GitGraphDiagram{
		Branches: []string{"main"},
		Commits: []diagram.GitCommit{
			{ID: "c1", Branch: "main"},
			{ID: "c2", Branch: "main", Parents: []string{"c1"}},
			{ID: "c3", Branch: "main", Parents: []string{"c2"}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "<circle") {
		t.Error("commit dot (circle) missing")
	}
	// No cross-branch edges expected on linear main-only history.
	if strings.Contains(raw, "<path") {
		t.Error("unexpected path element for linear same-lane history")
	}
}

func TestRenderBranchAndMerge(t *testing.T) {
	d := &diagram.GitGraphDiagram{
		Branches: []string{"main", "develop"},
		Commits: []diagram.GitCommit{
			{ID: "c1", Branch: "main"},
			{ID: "c2", Branch: "develop", Parents: []string{"c1"}},
			{ID: "c3", Branch: "main", Parents: []string{"c1"}},
			{ID: "m1", Branch: "main", Type: diagram.GitCommitMerge,
				Parents: []string{"c3", "c2"}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "<path") {
		t.Error("expected at least one curve for cross-branch parent edge")
	}
	if !strings.Contains(raw, ">develop<") {
		t.Error("develop lane label missing")
	}
	assertValidSVG(t, out)
}

func TestRenderHighlightAndReverseAndTag(t *testing.T) {
	d := &diagram.GitGraphDiagram{
		Branches: []string{"main"},
		Commits: []diagram.GitCommit{
			{ID: "c1", Branch: "main", Type: diagram.GitCommitHighlight, Tag: "v1"},
			{ID: "c2", Branch: "main", Type: diagram.GitCommitReverse, Parents: []string{"c1"}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, ">v1<") {
		t.Error("tag label missing (should render in place of ID)")
	}
	assertValidSVG(t, out)
}

func TestRenderDeterministic(t *testing.T) {
	d := &diagram.GitGraphDiagram{
		Branches: []string{"main", "feat"},
		Commits: []diagram.GitCommit{
			{ID: "c1", Branch: "main"},
			{ID: "c2", Branch: "feat", Parents: []string{"c1"}},
			{ID: "c3", Branch: "main", Parents: []string{"c1"}},
			{ID: "m1", Branch: "main", Type: diagram.GitCommitMerge,
				Parents: []string{"c3", "c2"}},
		},
	}
	first, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	for i := 0; i < 10; i++ {
		next, err := Render(d, nil)
		if err != nil {
			t.Fatalf("iter %d: %v", i, err)
		}
		if string(next) != string(first) {
			t.Fatalf("iter %d: output diverges", i)
		}
	}
}

func TestRenderCustomFontSize(t *testing.T) {
	d := &diagram.GitGraphDiagram{
		Branches: []string{"main"},
		Commits:  []diagram.GitCommit{{ID: "c1", Branch: "main"}},
	}
	out, err := Render(d, &Options{FontSize: 20})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(string(out), "font-size:20px") {
		t.Error("custom font size not applied")
	}
}

func TestRenderMissingParentSilentlySkipped(t *testing.T) {
	// A dangling parent ID must not crash the renderer and must not
	// produce a <path> element referring to a nonexistent coordinate.
	d := &diagram.GitGraphDiagram{
		Branches: []string{"main"},
		Commits: []diagram.GitCommit{
			{ID: "c1", Branch: "main", Parents: []string{"ghost"}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if strings.Contains(raw, "<path") {
		t.Error("dangling parent should not emit a path element")
	}
	if strings.Contains(raw, "ghost") {
		t.Error("dangling parent ID should not appear in output")
	}
}

// Branch label is rendered as a filled colored pill (rect) with white
// text inside — the identifying mmdc affordance for swim lanes.
func TestRenderBranchPill(t *testing.T) {
	d := &diagram.GitGraphDiagram{
		Branches: []string{"main", "dev"},
		Commits: []diagram.GitCommit{
			{ID: "c1", Branch: "main"},
			{ID: "c2", Branch: "dev", Parents: []string{"c1"}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	th := DefaultTheme()
	// The first branch's color must appear as a rect fill (the pill)
	// and the branch label text must use the pill text color.
	pillFill := fmt.Sprintf(`fill:%s;stroke:none`, th.BranchColors[0])
	if !strings.Contains(raw, pillFill) {
		t.Errorf("expected pill backdrop %q in output", pillFill)
	}
	if !strings.Contains(raw, "fill:"+th.BranchLabelText) {
		t.Errorf("expected pill text color %q", th.BranchLabelText)
	}
}

// Swimlane baseline: one dashed line per branch, drawn under the
// colored branch path, spanning the full chart.
func TestRenderSwimlaneBaseline(t *testing.T) {
	d := &diagram.GitGraphDiagram{
		Branches: []string{"main", "dev"},
		Commits: []diagram.GitCommit{
			{ID: "c1", Branch: "main"},
			{ID: "c2", Branch: "dev", Parents: []string{"c1"}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	// Dashed baselines count: one per branch.
	dashed := strings.Count(raw, "stroke-dasharray:4,4")
	if dashed != len(d.Branches) {
		t.Errorf("expected %d dashed swimlane lines, got %d", len(d.Branches), dashed)
	}
}

// HIGHLIGHT commits render as an outlined square (rect with the
// branch color stroke), not a larger circle.
func TestRenderHighlightIsSquare(t *testing.T) {
	d := &diagram.GitGraphDiagram{
		Branches: []string{"main"},
		Commits: []diagram.GitCommit{
			{ID: "c1", Branch: "main", Type: diagram.GitCommitHighlight},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	th := DefaultTheme()
	// A rect with the theme DotStrokeFill (white) and branch-color
	// stroke uniquely identifies the highlight-as-square shape.
	want := fmt.Sprintf("fill:%s;stroke:%s;stroke-width:2", th.DotStrokeFill, th.BranchColors[0])
	if !strings.Contains(raw, want) {
		t.Errorf("expected highlight square with %q", want)
	}
}

// Tag renders as a rounded callout above the commit: a rect with
// TagFill + TagStroke plus the tag text. A plain commit-id text on
// the same commit must NOT be emitted (tag wins).
func TestRenderTagCalloutSuppressesID(t *testing.T) {
	d := &diagram.GitGraphDiagram{
		Branches: []string{"main"},
		Commits: []diagram.GitCommit{
			{ID: "c1", Branch: "main", Tag: "v1.0"},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	th := DefaultTheme()
	want := fmt.Sprintf("fill:%s;stroke:%s;stroke-width:1", th.TagFill, th.TagStroke)
	if !strings.Contains(raw, want) {
		t.Errorf("expected tag callout fill+stroke %q", want)
	}
	if !strings.Contains(raw, ">v1.0<") {
		t.Error("tag text missing")
	}
	if strings.Contains(raw, ">c1<") {
		t.Error("commit ID text should be suppressed when a tag is present")
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
