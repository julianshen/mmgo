package gitgraph

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strconv"
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

// Cherry-pick commits render with a distinct hollow-circle +
// horizontal-bar glyph, NOT the solid circle / square / ring
// glyphs other types use.
func TestRenderCherryPickGlyph(t *testing.T) {
	d := &diagram.GitGraphDiagram{
		Branches: []string{"main", "dev"},
		Commits: []diagram.GitCommit{
			{ID: "a", Branch: "main", Type: diagram.GitCommitNormal},
			{ID: "b", Branch: "dev", Type: diagram.GitCommitNormal, Parents: []string{"a"}},
			{ID: "cp1", Branch: "main", Type: diagram.GitCommitCherryPick,
				CherryPickOf: "b", Parents: []string{"a"}},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "<circle") || !strings.Contains(raw, "<line") {
		t.Errorf("cherry-pick glyph should emit both circle and line:\n%s", raw)
	}
}

// AccTitle/AccDescr emit as <title>/<desc> SVG children; Title
// renders as a centered caption above the lanes.
func TestRenderGitGraphHeader(t *testing.T) {
	d := &diagram.GitGraphDiagram{
		Title:    "Release flow",
		AccTitle: "Build pipeline",
		AccDescr: "Trunk + hotfix",
		Branches: []string{"main"},
		Commits: []diagram.GitCommit{
			{ID: "a", Branch: "main", Type: diagram.GitCommitNormal},
		},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "<title>Build pipeline</title>") {
		t.Errorf("expected accTitle <title> in:\n%s", raw)
	}
	if !strings.Contains(raw, "<desc>Trunk + hotfix</desc>") {
		t.Errorf("expected accDescr <desc> in:\n%s", raw)
	}
	if !strings.Contains(raw, ">Release flow<") {
		t.Errorf("expected diagram title in:\n%s", raw)
	}
}

// ShowBranches=false suppresses the pill labels on the left gutter.
// The dashed lane guides and colored branch path lines must remain.
func TestRenderHidesBranchPillsWhenDisabled(t *testing.T) {
	d := &diagram.GitGraphDiagram{
		Branches: []string{"main", "feat"},
		Commits: []diagram.GitCommit{
			{ID: "a", Branch: "main", Type: diagram.GitCommitNormal},
			{ID: "b", Branch: "feat", Type: diagram.GitCommitNormal, Parents: []string{"a"}},
		},
	}
	off := false
	out, err := Render(d, &Options{Config: Config{ShowBranches: &off}})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if strings.Contains(raw, ">main<") || strings.Contains(raw, ">feat<") {
		t.Errorf("expected branch pill labels suppressed when ShowBranches=false")
	}
	// The colored per-branch path line uses stroke-width:4 (branchPathW);
	// no other element shares that thickness, so its absence is a clean
	// signal that the branch swimlane is fully suppressed.
	if strings.Contains(raw, "stroke-width:4;fill:none;opacity:1") {
		t.Errorf("expected colored branch path lines suppressed when ShowBranches=false")
	}
}

// ShowCommitLabel=false suppresses the commit-id labels above each
// dot. Tagged commits keep their tag callout regardless.
func TestRenderHidesCommitLabelsWhenDisabled(t *testing.T) {
	d := &diagram.GitGraphDiagram{
		Branches: []string{"main"},
		Commits: []diagram.GitCommit{
			{ID: "abc", Branch: "main", Type: diagram.GitCommitNormal},
			{ID: "def", Branch: "main", Type: diagram.GitCommitNormal, Tag: "v1"},
		},
	}
	off := false
	out, err := Render(d, &Options{Config: Config{ShowCommitLabel: &off}})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	if strings.Contains(raw, ">abc<") {
		t.Errorf("expected commit label 'abc' suppressed when ShowCommitLabel=false")
	}
	if !strings.Contains(raw, ">v1<") {
		t.Errorf("expected tag callout 'v1' to remain")
	}
}

// RotateCommitLabel=true (the spec default) rotates labels -45°
// around the dot. Setting false renders horizontally with no
// transform attribute on the label text.
func TestRenderRotateCommitLabel(t *testing.T) {
	d := &diagram.GitGraphDiagram{
		Branches: []string{"main"},
		Commits: []diagram.GitCommit{
			{ID: "abc", Branch: "main", Type: diagram.GitCommitNormal},
		},
	}
	rotated, err := Render(d, nil) // default = rotated
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(string(rotated), `transform="rotate(-45 `) {
		t.Errorf("expected default-rotated commit label, got:\n%s", string(rotated))
	}
	off := false
	flat, err := Render(d, &Options{Config: Config{RotateCommitLabel: &off}})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if strings.Contains(string(flat), `transform="rotate(`) {
		t.Errorf("expected RotateCommitLabel=false to drop the rotate transform")
	}
}

// MainBranchOrder shifts the implicit main branch lane downward when
// no explicit order is set on it; feature branches with order=0 take
// the top lane.
func TestRenderMainBranchOrder(t *testing.T) {
	d := &diagram.GitGraphDiagram{
		MainBranchName: "main",
		Branches:       []string{"main", "feat"},
		Commits: []diagram.GitCommit{
			{ID: "a", Branch: "main"},
			{ID: "b", Branch: "feat", Parents: []string{"a"}},
		},
	}
	out, err := Render(d, &Options{Config: Config{MainBranchOrder: 5}})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	// `feat` pill renders in lane 0 (y = marginY = 40); `main` in lane
	// 1 (y = marginY + laneHeight = 100). Look for the rect y of each.
	mainRectY := lookupBranchPillY(t, raw, "main")
	featRectY := lookupBranchPillY(t, raw, "feat")
	if !(featRectY < mainRectY) {
		t.Errorf("expected feat (no explicit order) above main (order=5); got featY=%v mainY=%v", featRectY, mainRectY)
	}
}

// LR (and the empty-default LR) produce one layout; TB/BT produce
// distinct vertical layouts. Pin the dispatch so a regression that
// dropped Direction reads would surface here.
func TestRenderDirectionDispatches(t *testing.T) {
	build := func(dir diagram.GitGraphDirection) []byte {
		d := &diagram.GitGraphDiagram{
			Direction: dir,
			Branches:  []string{"main", "feat"},
			Commits: []diagram.GitCommit{
				{ID: "a", Branch: "main"},
				{ID: "b", Branch: "feat", Parents: []string{"a"}},
				{ID: "c", Branch: "main", Parents: []string{"a"}},
			},
		}
		out, err := Render(d, nil)
		if err != nil {
			t.Fatalf("Render(%q): %v", dir, err)
		}
		return out
	}
	bare := string(build(""))
	lr := string(build(diagram.GitGraphDirLR))
	tb := string(build(diagram.GitGraphDirTB))
	bt := string(build(diagram.GitGraphDirBT))
	if bare != lr {
		t.Error("empty Direction must default to LR")
	}
	if lr == tb {
		t.Error("TB must produce a different layout than LR")
	}
	if tb == bt {
		t.Error("BT must produce a different layout than TB (commit order inverted)")
	}
}

// Explicit BranchOrder["main"] beats MainBranchOrder. The renderer's
// orderedLanes only falls back to mainBranchOrder when the map has no
// entry for the main branch, so a future precedence flip would
// silently change layouts.
func TestRenderMainBranchOrderDeferredToExplicit(t *testing.T) {
	d := &diagram.GitGraphDiagram{
		MainBranchName: "main",
		Branches:       []string{"main", "feat"},
		BranchOrder:    map[string]int{"main": 0, "feat": 1},
		Commits: []diagram.GitCommit{
			{ID: "a", Branch: "main"},
			{ID: "b", Branch: "feat", Parents: []string{"a"}},
		},
	}
	out, err := Render(d, &Options{Config: Config{MainBranchOrder: 999}})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	mainY := lookupBranchPillY(t, raw, "main")
	featY := lookupBranchPillY(t, raw, "feat")
	if !(mainY < featY) {
		t.Errorf("explicit BranchOrder[main]=0 must beat MainBranchOrder=999; got mainY=%v featY=%v", mainY, featY)
	}
}

// Branches sharing the same explicit order keep declaration sequence
// (sort.SliceStable). A future swap to sort.Slice would break
// determinism and is invisible without this pin.
func TestRenderLaneSortStableOnTies(t *testing.T) {
	d := &diagram.GitGraphDiagram{
		Branches:    []string{"a", "b", "c"},
		BranchOrder: map[string]int{"a": 1, "b": 1, "c": 1},
		Commits:     []diagram.GitCommit{{ID: "x", Branch: "a"}},
	}
	out, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	raw := string(out)
	yA := lookupBranchPillY(t, raw, "a")
	yB := lookupBranchPillY(t, raw, "b")
	yC := lookupBranchPillY(t, raw, "c")
	if !(yA < yB && yB < yC) {
		t.Errorf("equal-order branches must keep declaration order; got yA=%v yB=%v yC=%v", yA, yB, yC)
	}
}

// ShowBranches=false collapses the gutter — commit dots shift left
// to the bare margin. Without this assertion a regression that hides
// pills but leaves the gutter (wasted whitespace) would pass the
// existing coverage.
func TestRenderShowBranchesCollapsesGutter(t *testing.T) {
	d := &diagram.GitGraphDiagram{
		Branches: []string{"main", "feature-with-long-name"},
		Commits: []diagram.GitCommit{
			{ID: "a", Branch: "main"},
		},
	}
	on, off := true, false
	with, _ := Render(d, &Options{Config: Config{ShowBranches: &on}})
	without, _ := Render(d, &Options{Config: Config{ShowBranches: &off}})
	// First commit dot is the only <circle> with cy ≠ 0; compare cx.
	cxOn := firstCircleCX(t, string(with))
	cxOff := firstCircleCX(t, string(without))
	if cxOn-cxOff < 50 {
		t.Errorf("ShowBranches=false must collapse the gutter; cxOn=%v cxOff=%v", cxOn, cxOff)
	}
}

// RotateCommitLabel=false applied to a tag-bearing commit must leave
// the tag callout unrotated — tag rendering is a separate code path
// that should never pick up the rotate transform.
func TestRenderRotateLabelDoesNotAffectTags(t *testing.T) {
	d := &diagram.GitGraphDiagram{
		Branches: []string{"main"},
		Commits: []diagram.GitCommit{
			{ID: "abc", Branch: "main", Tag: "v1.0"},
		},
	}
	off := false
	out, err := Render(d, &Options{Config: Config{RotateCommitLabel: &off}})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if strings.Contains(string(out), "rotate(") {
		t.Errorf("tag callouts must never carry rotate transforms")
	}
}

func firstCircleCX(t *testing.T, svg string) float64 {
	t.Helper()
	idx := strings.Index(svg, "<circle")
	if idx < 0 {
		t.Fatalf("no <circle> element in:\n%s", svg)
	}
	head := svg[idx:]
	cxAt := strings.Index(head, `cx="`)
	if cxAt < 0 {
		t.Fatal("circle missing cx")
	}
	end := strings.Index(head[cxAt+4:], `"`)
	v, err := strconv.ParseFloat(head[cxAt+4:cxAt+4+end], 64)
	if err != nil {
		t.Fatalf("parse cx: %v", err)
	}
	return v
}

// lookupBranchPillY returns the y attribute of the <text> element
// containing the supplied label — this is the pill's text baseline,
// which equals the lane center, so it works for ordering assertions
// even though it isn't the pill rect's y.
func lookupBranchPillY(t *testing.T, svg, branchLabel string) float64 {
	t.Helper()
	marker := ">" + branchLabel + "<"
	idx := strings.Index(svg, marker)
	if idx < 0 {
		t.Fatalf("missing branch label %q", branchLabel)
	}
	head := svg[:idx]
	yAt := strings.LastIndex(head, ` y="`)
	if yAt < 0 {
		t.Fatalf("no y attribute before label %q", branchLabel)
	}
	end := strings.Index(head[yAt+4:], `"`)
	v, err := strconv.ParseFloat(head[yAt+4:yAt+4+end], 64)
	if err != nil {
		t.Fatalf("parse y: %v", err)
	}
	return v
}

func TestCommitSlotsSequentialDefault(t *testing.T) {
	commits := []diagram.GitCommit{
		{ID: "c1", Branch: "main"},
		{ID: "c2", Branch: "develop", Parents: []string{"c1"}},
		{ID: "c3", Branch: "main", Parents: []string{"c1"}},
	}
	got := commitSlots(commits, false)
	want := []int{0, 1, 2}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("slot[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

func TestCommitSlotsParallelCollapsesDepth(t *testing.T) {
	// c2 (off c1) and c3 (off c1) sit at the same depth — both get
	// slot 1 under parallelCommits, freeing column 2 for the merge.
	commits := []diagram.GitCommit{
		{ID: "c1", Branch: "main"},
		{ID: "c2", Branch: "develop", Parents: []string{"c1"}},
		{ID: "c3", Branch: "main", Parents: []string{"c1"}},
		{ID: "m1", Branch: "main", Type: diagram.GitCommitMerge,
			Parents: []string{"c3", "c2"}},
	}
	got := commitSlots(commits, true)
	want := []int{0, 1, 1, 2}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("slot[%d] = %d, want %d (full=%v)", i, got[i], want[i], got)
		}
	}
}

func TestRenderParallelCommitsAlignsParallelBranches(t *testing.T) {
	d := &diagram.GitGraphDiagram{
		Branches: []string{"main", "develop"},
		Commits: []diagram.GitCommit{
			{ID: "c1", Branch: "main"},
			{ID: "c2", Branch: "develop", Parents: []string{"c1"}},
			{ID: "c3", Branch: "main", Parents: []string{"c1"}},
		},
	}
	yes := true
	out, err := Render(d, &Options{Config: Config{ParallelCommits: &yes}})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	c2x := dotXFor(t, string(out), "c2")
	c3x := dotXFor(t, string(out), "c3")
	if c2x != c3x {
		t.Errorf("parallelCommits should align c2 and c3 on x; got c2=%v, c3=%v", c2x, c3x)
	}

	// Sanity: without the flag, the same commits should sit on
	// different columns.
	out2, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if dotXFor(t, string(out2), "c2") == dotXFor(t, string(out2), "c3") {
		t.Errorf("default rendering must keep sequential columns")
	}
}

// dotXFor returns the cx of the commit dot whose adjacent text
// content equals id. The renderer emits the dot before the id label,
// and the label appears as `>id<` in the SVG, so we walk back from
// the label to the most recent `cx="…"`.
func dotXFor(t *testing.T, svg, id string) float64 {
	t.Helper()
	marker := ">" + id + "<"
	idx := strings.Index(svg, marker)
	if idx < 0 {
		t.Fatalf("missing label %q in svg", id)
	}
	head := svg[:idx]
	cxAt := strings.LastIndex(head, ` cx="`)
	if cxAt < 0 {
		t.Fatalf("no cx before label %q", id)
	}
	end := strings.Index(head[cxAt+5:], `"`)
	v, err := strconv.ParseFloat(head[cxAt+5:cxAt+5+end], 64)
	if err != nil {
		t.Fatalf("parse cx for %q: %v", id, err)
	}
	return v
}

func TestCommitSlotsParallelHandlesCherryPick(t *testing.T) {
	commits := []diagram.GitCommit{
		{ID: "c1", Branch: "main"},
		{ID: "c2", Branch: "develop", Parents: []string{"c1"}},
		{ID: "cp", Branch: "main", Type: diagram.GitCommitCherryPick,
			CherryPickOf: "c2", Parents: []string{"c1"}},
	}
	got := commitSlots(commits, true)
	want := []int{0, 1, 1}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("slot[%d] = %d, want %d (full=%v)", i, got[i], want[i], got)
		}
	}
}

func TestRenderParallelCommitsBTInverts(t *testing.T) {
	d := &diagram.GitGraphDiagram{
		Direction: diagram.GitGraphDirBT,
		Branches:  []string{"main", "develop"},
		Commits: []diagram.GitCommit{
			{ID: "c1", Branch: "main"},
			{ID: "c2", Branch: "develop", Parents: []string{"c1"}},
			{ID: "c3", Branch: "main", Parents: []string{"c1"}},
		},
	}
	yes := true
	out, err := Render(d, &Options{Config: Config{ParallelCommits: &yes}})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	// Under BT the commit axis is y, and parallel siblings c2/c3
	// should share the same y after inversion.
	c2y := dotYFor(t, string(out), "c2")
	c3y := dotYFor(t, string(out), "c3")
	if c2y != c3y {
		t.Errorf("BT + parallelCommits: c2 and c3 should share y; got c2=%v c3=%v", c2y, c3y)
	}
	c1y := dotYFor(t, string(out), "c1")
	// BT puts the latest depth at the top (low y), root at bottom
	// (high y).
	if !(c1y > c2y) {
		t.Errorf("BT inversion: c1 (root) should be below c2; c1y=%v c2y=%v", c1y, c2y)
	}
}

func TestRenderParallelCommitsShrinksWidth(t *testing.T) {
	d := &diagram.GitGraphDiagram{
		Branches: []string{"main", "develop"},
		Commits: []diagram.GitCommit{
			{ID: "c1", Branch: "main"},
			{ID: "c2", Branch: "develop", Parents: []string{"c1"}},
			{ID: "c3", Branch: "main", Parents: []string{"c1"}},
		},
	}
	yes := true
	parallel, err := Render(d, &Options{Config: Config{ParallelCommits: &yes}})
	if err != nil {
		t.Fatalf("Render parallel: %v", err)
	}
	sequential, err := Render(d, nil)
	if err != nil {
		t.Fatalf("Render sequential: %v", err)
	}
	if viewBoxWidth(t, parallel) >= viewBoxWidth(t, sequential) {
		t.Errorf("parallelCommits should produce a narrower viewBox; parallel=%v sequential=%v",
			viewBoxWidth(t, parallel), viewBoxWidth(t, sequential))
	}
}

func dotYFor(t *testing.T, svg, id string) float64 {
	t.Helper()
	marker := ">" + id + "<"
	idx := strings.Index(svg, marker)
	if idx < 0 {
		t.Fatalf("missing label %q in svg", id)
	}
	head := svg[:idx]
	cyAt := strings.LastIndex(head, ` cy="`)
	if cyAt < 0 {
		t.Fatalf("no cy before label %q", id)
	}
	end := strings.Index(head[cyAt+5:], `"`)
	v, err := strconv.ParseFloat(head[cyAt+5:cyAt+5+end], 64)
	if err != nil {
		t.Fatalf("parse cy for %q: %v", id, err)
	}
	return v
}

func viewBoxWidth(t *testing.T, svg []byte) float64 {
	t.Helper()
	s := string(svg)
	at := strings.Index(s, `viewBox="`)
	if at < 0 {
		t.Fatalf("no viewBox attribute")
	}
	end := strings.Index(s[at+9:], `"`)
	parts := strings.Fields(s[at+9 : at+9+end])
	if len(parts) != 4 {
		t.Fatalf("unexpected viewBox %q", s[at+9:at+9+end])
	}
	w, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		t.Fatalf("parse viewBox w: %v", err)
	}
	return w
}
