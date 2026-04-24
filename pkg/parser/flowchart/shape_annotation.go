package flowchart

import (
	"fmt"
	"strings"

	"github.com/julianshen/mmgo/pkg/diagram"
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

	// Asymmetric: extended short names for an asymmetric glyph exist
	// in Mermaid but the spec routes them to new enum values in
	// Stage 2 alongside their renderer cases. Stage 1 deliberately
	// has no `asym` / `asymmetric` alias here — users still get the
	// existing `>...]` traditional-delimiter syntax.

	// Double circle
	"dbl-circ":      diagram.NodeShapeDoubleCircle,
	"double-circle": diagram.NodeShapeDoubleCircle,
}

// pendingShapes lists Mermaid extended-syntax names that are
// recognized but not yet implemented. Encountering one falls back to
// Rectangle silently so a mid-migration diagram (which may mix
// Stage-1 shapes with not-yet-supported ones) still parses; users
// see a plain rectangle in place of the not-yet-implemented glyph.
//
// As Stage 2/3 land each new NodeShape constant + renderer case, the
// corresponding entries move from this set into shapeAliases.
var pendingShapes = map[string]struct{}{
	// Polygon / circle / rect family (Stage 2)
	"tri": {}, "extract": {}, "triangle": {},
	"flip-tri": {}, "flipped-triangle": {}, "manual-file": {},
	"hourglass": {}, "collate": {},
	"notch-pent": {}, "loop-limit": {}, "notched-pentagon": {},
	"odd": {},
	"flag": {}, "paper-tape": {},
	"sl-rect": {}, "manual-input": {}, "sloped-rectangle": {},
	"sm-circ": {}, "small-circle": {}, "start": {},
	"f-circ": {}, "filled-circle": {}, "junction": {},
	"fr-circ": {}, "framed-circle": {}, "stop": {},
	"cross-circ": {}, "crossed-circle": {}, "summary": {},
	"div-rect": {}, "div-proc": {}, "divided-process": {}, "divided-rectangle": {},
	"win-pane": {}, "internal-storage": {}, "window-pane": {},
	"lin-rect": {}, "lin-proc": {}, "lined-process": {}, "lined-rectangle": {}, "shaded-process": {},
	"fork": {}, "join": {},
	"notch-rect": {}, "card": {}, "notched-rectangle": {},

	// Path-based shapes (Stage 3)
	"cloud":              {},
	"bang":               {},
	"bolt": {}, "com-link": {}, "lightning-bolt": {},
	"doc": {}, "document": {},
	"lin-doc": {}, "lined-document": {},
	"delay": {}, "half-rounded-rectangle": {},
	"h-cyl": {}, "das": {}, "horizontal-cylinder": {},
	"lin-cyl": {}, "disk": {}, "lined-cylinder": {},
	"curv-trap": {}, "curved-trapezoid": {}, "display": {},
	"bow-rect": {}, "bow-tie-rectangle": {}, "stored-data": {},
	"tag-rect": {}, "tag-proc": {}, "tagged-process": {}, "tagged-rectangle": {},
	"tag-doc": {}, "tagged-document": {},
	"st-rect": {}, "procs": {}, "processes": {}, "stacked-rectangle": {},
	"docs": {}, "documents": {}, "st-doc": {}, "stacked-document": {},
	"brace": {}, "brace-l": {}, "comment": {},
	"brace-r": {},
	"braces": {},
	"datastore": {}, "data-store": {},
	"text": {},
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
		if resolved, known := shapeAliases[key]; known {
			shape = resolved
		} else if _, pending := pendingShapes[key]; pending {
			// Recognized name whose renderer hasn't shipped yet.
			// Fall back to Rectangle so the rest of the diagram
			// still parses; users see a plain box where the
			// extended shape would eventually render.
			shape = diagram.NodeShapeRectangle
		} else {
			return diagram.NodeShapeUnknown, "", false, 0, false, fmt.Errorf("unknown shape %q in @{%s}", key, inner)
		}
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
