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
	"fr-rect":    diagram.NodeShapeSubroutine,
	"subprocess": diagram.NodeShapeSubroutine,
	"subroutine": diagram.NodeShapeSubroutine,

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

	// --- Stage 2 —-----------------------------------------------------
	// Moved from pendingShapes now that renderer cases exist.

	// Simple polygons
	"tri":              diagram.NodeShapeTriangle,
	"triangle":         diagram.NodeShapeTriangle,
	"extract":          diagram.NodeShapeTriangle,
	"flip-tri":         diagram.NodeShapeFlippedTriangle,
	"flipped-triangle": diagram.NodeShapeFlippedTriangle,
	"manual-file":      diagram.NodeShapeFlippedTriangle,
	"hourglass":        diagram.NodeShapeHourglass,
	"collate":          diagram.NodeShapeHourglass,
	"notch-pent":       diagram.NodeShapeNotchedPentagon,
	"loop-limit":       diagram.NodeShapeNotchedPentagon,
	"notched-pentagon": diagram.NodeShapeNotchedPentagon,
	"odd":              diagram.NodeShapeOdd,
	"flag":             diagram.NodeShapeFlag,
	"paper-tape":       diagram.NodeShapeFlag,
	"sl-rect":          diagram.NodeShapeSlopedRect,
	"sloped-rectangle": diagram.NodeShapeSlopedRect,
	"manual-input":     diagram.NodeShapeSlopedRect,

	// Circle variants
	"sm-circ":        diagram.NodeShapeSmallCircle,
	"small-circle":   diagram.NodeShapeSmallCircle,
	"start":          diagram.NodeShapeSmallCircle,
	"f-circ":         diagram.NodeShapeFilledCircle,
	"filled-circle":  diagram.NodeShapeFilledCircle,
	"junction":       diagram.NodeShapeFilledCircle,
	"fr-circ":        diagram.NodeShapeFramedCircle,
	"framed-circle":  diagram.NodeShapeFramedCircle,
	"stop":           diagram.NodeShapeFramedCircle,
	"cross-circ":     diagram.NodeShapeCrossCircle,
	"crossed-circle": diagram.NodeShapeCrossCircle,
	"summary":        diagram.NodeShapeCrossCircle,

	// Modified rectangles
	"div-rect":          diagram.NodeShapeDividedRect,
	"div-proc":          diagram.NodeShapeDividedRect,
	"divided-process":   diagram.NodeShapeDividedRect,
	"divided-rectangle": diagram.NodeShapeDividedRect,
	"win-pane":          diagram.NodeShapeWindowPane,
	"internal-storage":  diagram.NodeShapeWindowPane,
	"window-pane":       diagram.NodeShapeWindowPane,
	"lin-rect":          diagram.NodeShapeLinedRect,
	"lin-proc":          diagram.NodeShapeLinedRect,
	"lined-process":     diagram.NodeShapeLinedRect,
	"lined-rectangle":   diagram.NodeShapeLinedRect,
	"shaded-process":    diagram.NodeShapeLinedRect,
	"fork":              diagram.NodeShapeForkJoin,
	"join":              diagram.NodeShapeForkJoin,
	"notch-rect":        diagram.NodeShapeNotchedRect,
	"card":              diagram.NodeShapeNotchedRect,
	"notched-rectangle": diagram.NodeShapeNotchedRect,

	// --- Stage 3 — path-based shapes ----------------------------------
	"cloud":          diagram.NodeShapeCloud,
	"bang":           diagram.NodeShapeBang,
	"bolt":           diagram.NodeShapeBolt,
	"com-link":       diagram.NodeShapeBolt,
	"lightning-bolt": diagram.NodeShapeBolt,
	"doc":            diagram.NodeShapeDocument,
	"document":       diagram.NodeShapeDocument,
	"lin-doc":        diagram.NodeShapeLinedDocument,
	"lined-document": diagram.NodeShapeLinedDocument,

	"delay":                  diagram.NodeShapeDelay,
	"half-rounded-rectangle": diagram.NodeShapeDelay,

	"h-cyl":               diagram.NodeShapeHorizontalCylinder,
	"das":                 diagram.NodeShapeHorizontalCylinder,
	"horizontal-cylinder": diagram.NodeShapeHorizontalCylinder,

	"lin-cyl":        diagram.NodeShapeLinedCylinder,
	"disk":           diagram.NodeShapeLinedCylinder,
	"lined-cylinder": diagram.NodeShapeLinedCylinder,

	"curv-trap":        diagram.NodeShapeCurvedTrapezoid,
	"curved-trapezoid": diagram.NodeShapeCurvedTrapezoid,
	"display":          diagram.NodeShapeCurvedTrapezoid,

	"bow-rect":          diagram.NodeShapeBowTieRect,
	"bow-tie-rectangle": diagram.NodeShapeBowTieRect,
	"stored-data":       diagram.NodeShapeBowTieRect,

	"tag-rect":         diagram.NodeShapeTaggedRect,
	"tag-proc":         diagram.NodeShapeTaggedRect,
	"tagged-process":   diagram.NodeShapeTaggedRect,
	"tagged-rectangle": diagram.NodeShapeTaggedRect,

	"tag-doc":         diagram.NodeShapeTaggedDocument,
	"tagged-document": diagram.NodeShapeTaggedDocument,

	"st-rect":           diagram.NodeShapeStackedRect,
	"procs":             diagram.NodeShapeStackedRect,
	"processes":         diagram.NodeShapeStackedRect,
	"stacked-rectangle": diagram.NodeShapeStackedRect,

	"docs":             diagram.NodeShapeStackedDocument,
	"documents":        diagram.NodeShapeStackedDocument,
	"st-doc":           diagram.NodeShapeStackedDocument,
	"stacked-document": diagram.NodeShapeStackedDocument,

	"brace":   diagram.NodeShapeBrace,
	"brace-l": diagram.NodeShapeBrace,
	"comment": diagram.NodeShapeBrace,
	"brace-r": diagram.NodeShapeBraceR,
	"braces":  diagram.NodeShapeBraces,

	"datastore":  diagram.NodeShapeDataStore,
	"data-store": diagram.NodeShapeDataStore,

	"text": diagram.NodeShapeTextBlock,
}

// pendingShapes lists Mermaid extended-syntax names that are
// recognized but not yet implemented. An empty set means the catalog
// is fully covered; Stage 3 cleared the last entries. The mechanism
// (and this map) stays in place so a future Mermaid release adding a
// new short name can re-populate it without reintroducing the silent-
// vs-error logic.
var pendingShapes = map[string]struct{}{}

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
