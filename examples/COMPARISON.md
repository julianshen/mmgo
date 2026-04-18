# mmgo vs mmdc Output Comparison

Same 34 `.mmd` sources rendered by both `mmgo` (this project) and `mmdc`
(the official Node.js Mermaid CLI, v11.12.0). Raw `mmdc` outputs live in
[`examples-mmdc/`](../examples-mmdc/) alongside this repo's `examples/`
for side-by-side inspection.

## Coverage

| Diagram type | Source count | mmgo OK | mmdc OK | Notes |
|---|---:|---:|---:|---|
| flowchart    | 3 | 3 | 3 | |
| sequence     | 3 | 3 | 3 | |
| pie          | 2 | 2 | 2 | |
| class        | 2 | 2 | 2 | |
| state        | 2 | 2 | 2 | |
| er           | 2 | 2 | 2 | |
| gantt        | 2 | 2 | 2 | |
| mindmap      | 2 | 2 | 2 | |
| timeline     | 2 | 2 | 2 | |
| c4           | 2 | 2 | 2 | |
| block        | 2 | 2 | **0** | mmdc errors: `<path> attribute d: Expected number, "M10,NaNC300,NaN..."` |
| gitgraph     | 2 | 2 | 2 | |
| sankey       | 2 | 2 | 2 | |
| xychart      | 2 | 2 | 2 | |
| quadrant     | 2 | 2 | 2 | |
| kanban       | 2 | 2 | 2 | |
| **Total**    | **34** | **34** | **32** | |

## Speed (3 renders of `flowchart/simple.mmd`, wall clock)

| CLI  | Total | Per render | Startup overhead |
|---|---:|---:|---|
| mmgo | 0.04s | ~13ms | Static Go binary, no dependencies |
| mmdc | 2.67s | ~890ms | Node.js + headless Chromium boot per invocation |

**mmgo is ~70× faster per render** on small inputs. The gap narrows
proportionally on large diagrams (both are dominated by the same layout
work), but the per-render fixed cost in `mmdc` — Node runtime + Chromium
launch — is unavoidable.

## Output size

Per `ls`-level inspection, `mmgo` SVG is consistently **3-8× smaller**
than `mmdc` SVG: mmdc embeds an inline `<style>` block with the full
Mermaid theme CSS plus per-element class annotations; mmgo emits inline
`style="…"` per element. Functionally equivalent, but mmdc's output is
theme-swappable without re-rendering, while mmgo's is not (yet).

| Type       | mmgo SVG | mmdc SVG | Ratio |
|---|---:|---:|---:|
| pie/simple        | 1.8 KB  | 4.0 KB   | 2.3×  |
| sankey/simple     | 2.0 KB  | 5.7 KB   | 2.9×  |
| flowchart/simple  | 2.5 KB  | 15 KB    | 6.0×  |
| class/simple      | 3.1 KB  | 22 KB    | 7.2×  |
| er/blog           | 4.0 KB  | 57 KB    | 14×   |
| flowchart/shapes  | 3.9 KB  | 107 KB   | 27×   |

PNG sizes are closer (both rasterize the SVG) but mmdc's are often larger
because its SVGs encode heavier stroke/gradient effects.

## What the comparison reveals

1. **Correctness parity on the supported subset.** mmgo renders every
   diagram type without error. mmdc fails on `block-beta` (a Mermaid
   feature it supports but crashes on for these specific inputs — likely
   a bug in its layout branch for minimal block diagrams).

2. **Visual style differs.** mmgo produces intentionally simple,
   theme-unstyled output — enough for documentation, wiki embeds, and
   static site generators. mmdc produces richly-themed output that
   mirrors the Mermaid Live Editor's look.

3. **Deployment profile.** mmgo is a single ~20 MB static binary with no
   runtime dependencies. mmdc requires Node.js, a >100 MB npm tree, and
   downloads ~200 MB of headless Chromium on first use. For CI pipelines
   that render diagrams as part of a docs build, mmgo's profile is
   dramatically leaner.

4. **Feature surface.** mmdc supports every Mermaid diagram type
   including newer ones (radar, treemap, architecture, ZenUML, packet).
   mmgo covers 16 core types as of this snapshot. See
   [`../README.md`](../README.md) if present for the latest list.

## Regenerating the `mmdc` reference outputs

```sh
# Install mmdc once:
npm install -g @mermaid-js/mermaid-cli

# Render all examples:
for f in examples/*/*.mmd; do
    dir=$(dirname "$f" | sed 's|examples/|examples-mmdc/|')
    mkdir -p "$dir"
    base=$(basename "$f" .mmd)
    mmdc -i "$f" -o "$dir/$base.svg" 2>&1 | tail -1
    mmdc -i "$f" -o "$dir/$base.png" 2>&1 | tail -1
done
```
