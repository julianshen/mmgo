package gitgraph

import (
	"bytes"
	"encoding/xml"
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
	// A parent ID that doesn't exist in the commit map shouldn't
	// crash the renderer; it's simply a dangling reference (the
	// parser should never emit one, but render must be defensive).
	d := &diagram.GitGraphDiagram{
		Branches: []string{"main"},
		Commits: []diagram.GitCommit{
			{ID: "c1", Branch: "main", Parents: []string{"ghost"}},
		},
	}
	if _, err := Render(d, nil); err != nil {
		t.Fatalf("Render: %v", err)
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
