package svgutil

import (
	"math"
	"testing"
)

func TestRound2(t *testing.T) {
	if Round2(1.456) != 1.46 {
		t.Errorf("Round2(1.456) = %v", Round2(1.456))
	}
	if Round2(math.NaN()) != 0 {
		t.Error("NaN should round to 0")
	}
	if Round2(math.Inf(1)) != 0 {
		t.Error("Inf should round to 0")
	}
}

func TestSanitize(t *testing.T) {
	if Sanitize(-1) != 0 {
		t.Error("negative should sanitize to 0")
	}
	if Sanitize(42) != 42 {
		t.Error("positive should pass through")
	}
}

func TestViewBox(t *testing.T) {
	vb := ViewBox(100.5, 200)
	if vb != "0 0 100.50 200.00" {
		t.Errorf("ViewBox = %q", vb)
	}
}

func TestMarshalSVG(t *testing.T) {
	doc := Doc{XMLNS: "http://www.w3.org/2000/svg", ViewBox: "0 0 100 100"}
	out, err := MarshalSVG(doc)
	if err != nil {
		t.Fatalf("MarshalSVG: %v", err)
	}
	if len(out) == 0 {
		t.Error("output should not be empty")
	}
}
