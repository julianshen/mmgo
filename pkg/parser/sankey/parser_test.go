package sankey

import (
	"strings"
	"testing"

	"github.com/julianshen/mmgo/pkg/diagram"
)

func TestParseHeaderRequired(t *testing.T) {
	if _, err := Parse(strings.NewReader("A,B,10\n")); err == nil {
		t.Fatal("expected error for missing header")
	}
}

func TestParseEmpty(t *testing.T) {
	if _, err := Parse(strings.NewReader("")); err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestParseBadHeader(t *testing.T) {
	if _, err := Parse(strings.NewReader("flowchart LR\nA,B,10\n")); err == nil {
		t.Fatal("expected error for wrong header")
	}
}

func TestParseHeaderVariants(t *testing.T) {
	cases := []string{
		"sankey-beta\n",
		"sankey-beta:\n",
		"sankey-beta \n",
		"sankey-beta\tfoo\n",
		"sankey-beta: trailing junk\n",
	}
	for _, c := range cases {
		if _, err := Parse(strings.NewReader(c)); err != nil {
			t.Errorf("header %q: %v", c, err)
		}
	}
}

// Pins the permissive "sankey-beta: <anything>" semantics: content
// after the colon on the header line is ignored, not parsed as a flow
// row. Any future tightening of the header grammar should update this
// test deliberately.
func TestParseHeaderTrailingContentIgnored(t *testing.T) {
	input := `sankey-beta: this is not a flow row
A,B,10
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Flows) != 1 {
		t.Fatalf("flows = %d, want 1 (header trailing content must not become a flow)", len(d.Flows))
	}
	if d.Flows[0].Source != "A" {
		t.Errorf("first flow source = %q, want A", d.Flows[0].Source)
	}
}

func TestParseSimpleFlow(t *testing.T) {
	input := `sankey-beta
A,B,10
B,C,5
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Flows) != 2 {
		t.Fatalf("flows = %d, want 2", len(d.Flows))
	}
	if d.Flows[0] != (diagram.SankeyFlow{Source: "A", Target: "B", Value: 10}) {
		t.Errorf("flow[0] = %+v", d.Flows[0])
	}
}

func TestParseColumnHeaderSkipped(t *testing.T) {
	input := `sankey-beta
source,target,value
A,B,10
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Flows) != 1 {
		t.Fatalf("flows = %d, want 1", len(d.Flows))
	}
}

func TestParseColumnHeaderOnlyMatchesFirstDataRow(t *testing.T) {
	// A flow whose fields happen to be literally (source, target,
	// value) must not be silently skipped when it appears after
	// real data rows. Only the *first* data row is treated as a
	// potential column header.
	input := `sankey-beta
A,B,10
source,target,15
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Flows) != 2 {
		t.Fatalf("flows = %d, want 2 (the second row must not be treated as a column header)", len(d.Flows))
	}
	if d.Flows[1].Source != "source" || d.Flows[1].Target != "target" {
		t.Errorf("second flow = %+v, want {source, target, 15}", d.Flows[1])
	}
}

func TestParseColumnHeaderCaseInsensitive(t *testing.T) {
	input := `sankey-beta
SOURCE,Target,Value
A,B,10
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Flows) != 1 {
		t.Errorf("flows = %d, want 1", len(d.Flows))
	}
}

func TestParseQuotedFields(t *testing.T) {
	input := `sankey-beta
"Agricultural waste","Bio-conversion",124.729
"Bio-conversion","Liquid, refined",0.597
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if d.Flows[1].Target != "Liquid, refined" {
		t.Errorf("target = %q, want 'Liquid, refined'", d.Flows[1].Target)
	}
	if d.Flows[0].Value != 124.729 {
		t.Errorf("value = %v, want 124.729", d.Flows[0].Value)
	}
}

func TestParseFloatValues(t *testing.T) {
	input := `sankey-beta
A,B,0.5
C,D,1e3
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if d.Flows[0].Value != 0.5 || d.Flows[1].Value != 1000 {
		t.Errorf("values = %v", d.Flows)
	}
}

func TestParseBadColumnCount(t *testing.T) {
	if _, err := Parse(strings.NewReader("sankey-beta\nA,B\n")); err == nil {
		t.Fatal("expected error for 2 columns")
	}
	if _, err := Parse(strings.NewReader("sankey-beta\nA,B,C,10\n")); err == nil {
		t.Fatal("expected error for 4 columns")
	}
}

func TestParseBadValue(t *testing.T) {
	if _, err := Parse(strings.NewReader("sankey-beta\nA,B,not-a-number\n")); err == nil {
		t.Fatal("expected error for non-numeric value")
	}
}

func TestParseNegativeValueRejected(t *testing.T) {
	if _, err := Parse(strings.NewReader("sankey-beta\nA,B,-5\n")); err == nil {
		t.Fatal("expected error for negative value")
	}
}

// strconv.ParseFloat happily decodes "NaN", "+Inf", "-Inf". The
// non-negative check does not catch NaN (NaN < 0 is false), so an
// explicit finiteness guard is required to keep non-finite values
// from poisoning the renderer's magnitude math.
func TestParseNonFiniteValuesRejected(t *testing.T) {
	cases := []string{
		"sankey-beta\nA,B,NaN\n",
		"sankey-beta\nA,B,Inf\n",
		"sankey-beta\nA,B,+Inf\n",
		"sankey-beta\nA,B,-Inf\n",
	}
	for _, c := range cases {
		if _, err := Parse(strings.NewReader(c)); err == nil {
			t.Errorf("expected error for input:\n%s", c)
		}
	}
}

// A self-loop (A -> A) has no sensible interpretation in a sankey
// flow graph. Reject at parse time rather than produce a degenerate
// zero-length ribbon downstream.
func TestParseSelfLoopRejected(t *testing.T) {
	if _, err := Parse(strings.NewReader("sankey-beta\nA,A,10\n")); err == nil {
		t.Fatal("expected error for self-loop")
	}
}

// Value zero is explicitly accepted; the renderer applies a minimum
// bar-height floor so the node remains visible.
func TestParseZeroValueAccepted(t *testing.T) {
	d, err := Parse(strings.NewReader("sankey-beta\nA,B,0\n"))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Flows) != 1 || d.Flows[0].Value != 0 {
		t.Errorf("flows = %v, want one flow with Value=0", d.Flows)
	}
}

func TestParseEmptySourceOrTarget(t *testing.T) {
	if _, err := Parse(strings.NewReader("sankey-beta\n,B,10\n")); err == nil {
		t.Fatal("expected error for empty source")
	}
	if _, err := Parse(strings.NewReader("sankey-beta\nA,,10\n")); err == nil {
		t.Fatal("expected error for empty target")
	}
}

func TestParseCommentsIgnored(t *testing.T) {
	input := `sankey-beta
%% a comment
A,B,10 %% trailing
`
	d, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(d.Flows) != 1 {
		t.Fatalf("flows = %d", len(d.Flows))
	}
}

func TestSankeyNodesFirstAppearanceOrder(t *testing.T) {
	d := &diagram.SankeyDiagram{
		Flows: []diagram.SankeyFlow{
			{Source: "A", Target: "B", Value: 1},
			{Source: "B", Target: "C", Value: 1},
			{Source: "A", Target: "C", Value: 1},
		},
	}
	got := d.Nodes()
	want := []string{"A", "B", "C"}
	if len(got) != len(want) {
		t.Fatalf("nodes = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("nodes[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestSankeyDiagramType(t *testing.T) {
	var d diagram.Diagram = &diagram.SankeyDiagram{}
	if d.Type() != diagram.Sankey {
		t.Errorf("Type() = %v, want Sankey", d.Type())
	}
}
