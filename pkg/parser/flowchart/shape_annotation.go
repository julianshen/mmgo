package flowchart

import (
	"fmt"
	"strings"

	"github.com/julianshen/mmgo/pkg/diagram"
	parserutil "github.com/julianshen/mmgo/pkg/parser"
)

// shapeAliases maps every Mermaid extended-syntax short name to its
// canonical NodeShape. Stage 1 covers only names that resolve to an
// existing shape constant — new shapes (cloud, bolt, bang, …) are
// added in Stage 2 alongside their renderer cases.
//
// Lowercase is the canonical lookup key; parseShapeAnnotation
// lowercases the parsed value before lookup, so authors can write
// either `shape: Diamond` or `shape: DIAMOND`.
var shapeAliases = map[string]diagram.NodeShape{
	// Rectangle family
	"rect":      diagram.NodeShapeRectangle,
	"rectangle": diagram.NodeShapeRectangle,
	"proc":      diagram.NodeShapeRectangle,
	"process":   diagram.NodeShapeRectangle,

	// Rounded rectangle
	"rounded": diagram.NodeShapeRoundedRectangle,
	"event":   diagram.NodeShapeRoundedRectangle,

	// Stadium / terminal
	"stadium":  diagram.NodeShapeStadium,
	"terminal": diagram.NodeShapeStadium,
	"pill":     diagram.NodeShapeStadium,

	// Subroutine / framed rectangle
	"fr-rect":     diagram.NodeShapeSubroutine,
	"subprocess":  diagram.NodeShapeSubroutine,
	"subroutine":  diagram.NodeShapeSubroutine,

	// Cylinder / database
	"cyl":      diagram.NodeShapeCylinder,
	"cylinder": diagram.NodeShapeCylinder,
	"db":       diagram.NodeShapeCylinder,
	"database": diagram.NodeShapeCylinder,

	// Circle
	"circ":   diagram.NodeShapeCircle,
	"circle": diagram.NodeShapeCircle,

	// Diamond / decision / question
	"diam":     diagram.NodeShapeDiamond,
	"diamond":  diagram.NodeShapeDiamond,
	"decision": diagram.NodeShapeDiamond,
	"question": diagram.NodeShapeDiamond,

	// Hexagon / prepare
	"hex":     diagram.NodeShapeHexagon,
	"hexagon": diagram.NodeShapeHexagon,
	"prepare": diagram.NodeShapeHexagon,

	// Parallelogram / lean-right (in/out)
	"lean-r":     diagram.NodeShapeParallelogram,
	"lean-right": diagram.NodeShapeParallelogram,
	"in-out":     diagram.NodeShapeParallelogram,

	// Parallelogram-alt / lean-left (out/in)
	"lean-l":    diagram.NodeShapeParallelogramAlt,
	"lean-left": diagram.NodeShapeParallelogramAlt,
	"out-in":    diagram.NodeShapeParallelogramAlt,

	// Trapezoid / priority (wide base)
	"trap-b":    diagram.NodeShapeTrapezoid,
	"trapezoid": diagram.NodeShapeTrapezoid,
	"priority":  diagram.NodeShapeTrapezoid,

	// Trapezoid-alt / manual (wide top)
	"trap-t":        diagram.NodeShapeTrapezoidAlt,
	"trapezoid-top": diagram.NodeShapeTrapezoidAlt,
	"inv-trapezoid": diagram.NodeShapeTrapezoidAlt,
	"manual":        diagram.NodeShapeTrapezoidAlt,

	// Asymmetric (flag) — kept under its legacy name here; extended
	// short names for an asymmetric glyph also exist in Mermaid but
	// Stage 2 gets them since they map to new enum values.
	"asym":       diagram.NodeShapeAsymmetric,
	"asymmetric": diagram.NodeShapeAsymmetric,

	// Double circle
	"dbl-circ":      diagram.NodeShapeDoubleCircle,
	"double-circle": diagram.NodeShapeDoubleCircle,
}

// parseShapeAnnotation looks for an `@{ ... }` extended-syntax
// annotation at the start of s and, if present, returns the resolved
// shape, an optional label override, and the number of bytes consumed
// so the caller can continue parsing what remains.
//
// Returns ok=false (and consumed=0) when s does not begin with `@{`.
// Returns an error for malformed content (missing `}`, unknown shape,
// empty shape value). Unknown shapes fail loudly rather than silently
// falling back to Rectangle — a typo like `@{shape:diamon}` is almost
// always a bug in the user's diagram.
func parseShapeAnnotation(s string) (shape diagram.NodeShape, label string, labelSet bool, consumed int, ok bool, err error) {
	if !strings.HasPrefix(s, "@{") {
		return diagram.NodeShapeUnknown, "", false, 0, false, nil
	}
	closeIdx := strings.IndexByte(s[2:], '}')
	if closeIdx < 0 {
		return diagram.NodeShapeUnknown, "", false, 0, false, fmt.Errorf("unclosed @{ in %q", s)
	}
	inner := s[2 : 2+closeIdx]
	consumed = 2 + closeIdx + 1

	kv, parseErr := parseKV(inner)
	if parseErr != nil {
		return diagram.NodeShapeUnknown, "", false, 0, false, fmt.Errorf("parsing @{%s}: %w", inner, parseErr)
	}

	shape = diagram.NodeShapeUnknown
	if raw, has := kv["shape"]; has {
		key := strings.ToLower(strings.TrimSpace(raw))
		if key == "" {
			return diagram.NodeShapeUnknown, "", false, 0, false, fmt.Errorf("empty shape value in @{%s}", inner)
		}
		resolved, known := shapeAliases[key]
		if !known {
			return diagram.NodeShapeUnknown, "", false, 0, false, fmt.Errorf("unknown shape %q in @{%s}", key, inner)
		}
		shape = resolved
	}
	if raw, has := kv["label"]; has {
		label = processLabel(raw)
		labelSet = true
	}
	return shape, label, labelSet, consumed, true, nil
}

// parseKV splits a `key: value, key: value` string as used inside
// `@{ ... }`. Keys are lowercased; values keep their case so labels
// survive round-trip. Quoted values (`label: "foo, bar"`) protect the
// inner comma from the top-level splitter.
func parseKV(raw string) (map[string]string, error) {
	out := map[string]string{}
	i := 0
	for i < len(raw) {
		// skip leading separators / whitespace
		for i < len(raw) && (raw[i] == ' ' || raw[i] == '\t' || raw[i] == ',') {
			i++
		}
		if i >= len(raw) {
			break
		}
		// read key up to ':'
		keyStart := i
		for i < len(raw) && raw[i] != ':' && raw[i] != ',' {
			i++
		}
		if i >= len(raw) || raw[i] != ':' {
			return nil, fmt.Errorf("expected ':' after key %q", strings.TrimSpace(raw[keyStart:i]))
		}
		key := strings.ToLower(strings.TrimSpace(raw[keyStart:i]))
		i++ // consume ':'
		// skip whitespace before value
		for i < len(raw) && (raw[i] == ' ' || raw[i] == '\t') {
			i++
		}
		// read value: quoted or until next ','
		var val string
		if i < len(raw) && (raw[i] == '"' || raw[i] == '\'') {
			quote := raw[i]
			i++
			valStart := i
			for i < len(raw) && raw[i] != quote {
				i++
			}
			if i >= len(raw) {
				return nil, fmt.Errorf("unterminated quoted value for key %q", key)
			}
			val = raw[valStart:i]
			i++ // consume closing quote
		} else {
			valStart := i
			for i < len(raw) && raw[i] != ',' {
				i++
			}
			val = strings.TrimSpace(raw[valStart:i])
		}
		if key == "" {
			return nil, fmt.Errorf("empty key before ':'")
		}
		out[key] = val
	}
	return out, nil
}

// processLabel is shared with the traditional-delimiter parser in
// parser.go; re-declared here only as a reminder — Go resolves it to
// the single definition at compile time.
var _ = parserutil.Unquote
