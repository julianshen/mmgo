package gitgraph

import (
	"strings"
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
)

func TestParseHeaderRequired(t *testing.T) {
	_, err := Parse(strings.NewReader("commit\n"))
	if err == nil {
		t.Fatal("expected error for missing header")
	}
}

func TestParseEmptyInput(t *testing.T) {
	_, err := Parse(strings.NewReader(""))
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestParseHeaderVariants(t *testing.T) {
	cases := []string{
		"gitGraph\n",
		"gitGraph:\n",
		"gitGraph LR\n",
		"gitGraph LR:\n",
		"gitGraph TB\n",
		"gitGraph BT:\n",
	}
	for _, c := range cases {
		d, err := Parse(strings.NewReader(c))
		if err != nil {
			t.Errorf("header %q: %v", c, err)
		}
		if d == nil {
			t.Errorf("header %q: nil diagram", c)
		}
	}
}

func TestParseBadHeader(t *testing.T) {
	_, err := Parse(strings.NewReader("graph LR\n"))
	if err == nil {
		t.Fatal("expected error for non-gitGraph header")
	}
}

func TestParseSingleCommit(t *testing.T) {
	d, err := Parse(strings.NewReader("gitGraph\ncommit\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Commits) != 1 {
		t.Fatalf("got %d commits, want 1", len(d.Commits))
	}
	c := d.Commits[0]
	if c.Branch != "main" {
		t.Errorf("branch = %q, want main", c.Branch)
	}
	if c.ID == "" {
		t.Error("ID should be auto-generated")
	}
	if len(c.Parents) != 0 {
		t.Errorf("first commit parents = %v, want none", c.Parents)
	}
	if len(d.Branches) != 1 || d.Branches[0] != "main" {
		t.Errorf("branches = %v, want [main]", d.Branches)
	}
}

func TestParseLinearCommits(t *testing.T) {
	input := `gitGraph
commit
commit
commit
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Commits) != 3 {
		t.Fatalf("got %d commits, want 3", len(d.Commits))
	}
	// Each commit after the first should have the previous as parent.
	for i := 1; i < 3; i++ {
		if len(d.Commits[i].Parents) != 1 {
			t.Errorf("commit %d parents = %v, want 1", i, d.Commits[i].Parents)
			continue
		}
		if d.Commits[i].Parents[0] != d.Commits[i-1].ID {
			t.Errorf("commit %d parent = %q, want %q",
				i, d.Commits[i].Parents[0], d.Commits[i-1].ID)
		}
	}
}

func TestParseCommitAttrs(t *testing.T) {
	input := `gitGraph
commit id: "initial" tag: "v1.0" type: HIGHLIGHT
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Commits) != 1 {
		t.Fatalf("got %d commits", len(d.Commits))
	}
	c := d.Commits[0]
	if c.ID != "initial" {
		t.Errorf("ID = %q, want initial", c.ID)
	}
	if c.Tag != "v1.0" {
		t.Errorf("Tag = %q, want v1.0", c.Tag)
	}
	if c.Type != diagram.GitCommitHighlight {
		t.Errorf("Type = %v, want Highlight", c.Type)
	}
}

func TestParseCommitTypeReverse(t *testing.T) {
	d, err := Parse(strings.NewReader("gitGraph\ncommit type: REVERSE\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if d.Commits[0].Type != diagram.GitCommitReverse {
		t.Errorf("Type = %v, want Reverse", d.Commits[0].Type)
	}
}

func TestParseCommitTypeUnknownFallsBackToNormal(t *testing.T) {
	d, err := Parse(strings.NewReader("gitGraph\ncommit type: BOGUS\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if d.Commits[0].Type != diagram.GitCommitNormal {
		t.Errorf("Type = %v, want Normal", d.Commits[0].Type)
	}
}

func TestParseBranchCheckoutMerge(t *testing.T) {
	input := `gitGraph
commit
branch develop
commit
checkout main
commit
merge develop
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Branches) != 2 {
		t.Fatalf("branches = %v, want [main develop]", d.Branches)
	}
	if d.Branches[0] != "main" || d.Branches[1] != "develop" {
		t.Errorf("branches = %v, want [main develop]", d.Branches)
	}
	if len(d.Commits) != 4 {
		t.Fatalf("got %d commits, want 4", len(d.Commits))
	}
	// Last commit is merge: has 2 parents.
	merge := d.Commits[3]
	if merge.Type != diagram.GitCommitMerge {
		t.Errorf("merge type = %v, want Merge", merge.Type)
	}
	if len(merge.Parents) != 2 {
		t.Errorf("merge parents = %v, want 2", merge.Parents)
	}
	if merge.Branch != "main" {
		t.Errorf("merge branch = %q, want main", merge.Branch)
	}
}

func TestParseBranchWithOrder(t *testing.T) {
	input := `gitGraph
commit
branch feature order: 2
commit
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Branches) != 2 || d.Branches[1] != "feature" {
		t.Errorf("branches = %v, want [main feature]", d.Branches)
	}
}

func TestParseBranchEmpty(t *testing.T) {
	_, err := Parse(strings.NewReader("gitGraph\nbranch\n"))
	if err == nil {
		t.Fatal("expected error for branch with no name")
	}
}

func TestParseCheckoutEmpty(t *testing.T) {
	_, err := Parse(strings.NewReader("gitGraph\ncheckout\n"))
	if err == nil {
		t.Fatal("expected error for checkout with no name")
	}
}

func TestParseMergeEmpty(t *testing.T) {
	_, err := Parse(strings.NewReader("gitGraph\nmerge\n"))
	if err == nil {
		t.Fatal("expected error for merge with no branch")
	}
}

func TestParseMergeWithAttrs(t *testing.T) {
	input := `gitGraph
commit
branch develop
commit
checkout main
merge develop id: "M1" tag: "release"
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	m := d.Commits[len(d.Commits)-1]
	if m.ID != "M1" {
		t.Errorf("merge ID = %q, want M1", m.ID)
	}
	if m.Tag != "release" {
		t.Errorf("merge tag = %q, want release", m.Tag)
	}
}

func TestParseCommentsIgnored(t *testing.T) {
	input := `gitGraph
%% this is a comment
commit %% trailing comment
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Commits) != 1 {
		t.Fatalf("got %d commits, want 1", len(d.Commits))
	}
}

func TestParseUnknownKeywordIgnored(t *testing.T) {
	// Unknown directives should be skipped so we're tolerant of
	// Mermaid grammar extensions (themes, directives, etc.).
	input := `gitGraph
theme base
commit
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Commits) != 1 {
		t.Fatalf("got %d commits, want 1", len(d.Commits))
	}
}

func TestParseCheckoutUnknownBranchRegisters(t *testing.T) {
	// Checking out a branch that was never declared should still
	// register it — Mermaid permits this loose style.
	input := `gitGraph
commit
checkout develop
commit
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Branches) != 2 {
		t.Errorf("branches = %v, want 2", d.Branches)
	}
}

func TestParseMergeIntoSameBranch(t *testing.T) {
	// Merging a branch into itself is a no-op on parent count
	// (one parent, not two) — defensive guard.
	input := `gitGraph
commit
merge main
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	merge := d.Commits[1]
	if len(merge.Parents) != 1 {
		t.Errorf("merge-into-self parents = %v, want 1", merge.Parents)
	}
}

func TestTokenizeAttrsQuotedSpaces(t *testing.T) {
	// Quoted values should preserve internal whitespace.
	input := `gitGraph
commit id: "my commit" tag: "v 1"
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	c := d.Commits[0]
	if c.ID != "my commit" {
		t.Errorf("ID = %q, want %q", c.ID, "my commit")
	}
	if c.Tag != "v 1" {
		t.Errorf("Tag = %q, want %q", c.Tag, "v 1")
	}
}

func TestGitCommitTypeString(t *testing.T) {
	cases := map[diagram.GitCommitType]string{
		diagram.GitCommitNormal:    "normal",
		diagram.GitCommitReverse:   "reverse",
		diagram.GitCommitHighlight: "highlight",
		diagram.GitCommitMerge:     "merge",
	}
	for v, want := range cases {
		if got := v.String(); got != want {
			t.Errorf("%d.String() = %q, want %q", v, got, want)
		}
	}
}

func TestGitGraphDiagramType(t *testing.T) {
	var d diagram.Diagram = &diagram.GitGraphDiagram{}
	if d.Type() != diagram.GitGraph {
		t.Errorf("Type() = %v, want GitGraph", d.Type())
	}
}

// `cherry-pick id: "x"` lands a CherryPickOf commit on the current
// branch with the original id stashed for the renderer.
func TestParseCherryPick(t *testing.T) {
	d, err := Parse(strings.NewReader(`gitGraph
commit id: "a"
branch dev
commit id: "b"
checkout main
cherry-pick id: "b"`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	last := d.Commits[len(d.Commits)-1]
	if last.Type != diagram.GitCommitCherryPick {
		t.Errorf("last commit type = %v, want cherry_pick", last.Type)
	}
	if last.CherryPickOf != "b" {
		t.Errorf("CherryPickOf = %q, want %q", last.CherryPickOf, "b")
	}
	if last.Branch != "main" {
		t.Errorf("branch = %q, want main", last.Branch)
	}
	if !strings.HasPrefix(last.ID, "cp") {
		t.Errorf("auto id = %q, want cpN prefix", last.ID)
	}
}

// `cherry-pick` without an `id:` is rejected per spec.
func TestParseCherryPickRequiresID(t *testing.T) {
	_, err := Parse(strings.NewReader("gitGraph\ncommit\ncherry-pick"))
	if err == nil {
		t.Error("expected error for cherry-pick without id")
	}
}

// `switch <branch>` is an alias for `checkout <branch>`.
func TestParseSwitchAlias(t *testing.T) {
	d, err := Parse(strings.NewReader(`gitGraph
commit
branch dev
commit
switch main
commit`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	last := d.Commits[len(d.Commits)-1]
	if last.Branch != "main" {
		t.Errorf("commit after switch should land on main, got %q", last.Branch)
	}
}

// `commit msg: "..."` populates the new Msg field, distinct from
// id and tag.
func TestParseCommitMsg(t *testing.T) {
	d, err := Parse(strings.NewReader(`gitGraph
commit id: "x" msg: "hello world"`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	c := d.Commits[0]
	if c.ID != "x" || c.Msg != "hello world" {
		t.Errorf("commit = %+v", c)
	}
}

// `branch <name> order: N` records the lane order without dropping
// the branch.
func TestParseBranchOrder(t *testing.T) {
	d, err := Parse(strings.NewReader(`gitGraph
commit
branch feature order: 2
commit
branch hotfix order: 1
commit`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if d.BranchOrder["feature"] != 2 {
		t.Errorf("feature order = %d, want 2", d.BranchOrder["feature"])
	}
	if d.BranchOrder["hotfix"] != 1 {
		t.Errorf("hotfix order = %d, want 1", d.BranchOrder["hotfix"])
	}
}

// Quoted branch names (`branch "release/1.0"`) survive without
// being split on the slash or whitespace.
func TestParseQuotedBranchName(t *testing.T) {
	d, err := Parse(strings.NewReader(`gitGraph
commit
branch "release/1.0"
commit
checkout "release/1.0"
commit`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	found := false
	for _, b := range d.Branches {
		if b == "release/1.0" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("branches = %v, missing quoted name", d.Branches)
	}
}
