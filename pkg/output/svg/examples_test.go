package svg

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestExamples re-renders every `examples/*/*.mmd` reference source
// through the implementation and asserts the committed `.svg` matches
// byte-for-byte. When a code change alters output, refresh the
// snapshots alongside the change by running:
//
//	go build -o /tmp/mmgo ./cmd/mmgo
//	for f in examples/*/*.mmd; do
//	    base="${f%.mmd}"
//	    /tmp/mmgo -i "$f" -o "${base}.svg" -q
//	    /tmp/mmgo -i "$f" -o "${base}.png" -q
//	done
//
// The PNG outputs are not byte-compared because tdewolff/canvas embeds
// timestamps in PNG metadata; only SVG is asserted.
func TestExamples(t *testing.T) {
	root := examplesRoot(t)
	matches, err := filepath.Glob(filepath.Join(root, "*", "*.mmd"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(matches) == 0 {
		t.Fatal("no example .mmd files found")
	}
	for _, src := range matches {
		src := src
		rel, _ := filepath.Rel(root, src)
		t.Run(rel, func(t *testing.T) {
			body, err := os.ReadFile(src)
			if err != nil {
				t.Fatalf("read %s: %v", src, err)
			}
			got, err := Render(bytes.NewReader(body), nil)
			if err != nil {
				t.Fatalf("Render %s: %v", src, err)
			}
			want, err := os.ReadFile(strings.TrimSuffix(src, ".mmd") + ".svg")
			if err != nil {
				t.Fatalf("read committed svg: %v", err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("%s: rendered output differs from committed snapshot — refresh examples/ to match", rel)
			}
		})
	}
}

// examplesRoot returns the path to the repo's examples/ directory,
// walking up from the test's package directory so the test runs the
// same whether invoked from `go test ./...` at the repo root or from
// within the package.
func examplesRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := wd
	for i := 0; i < 10; i++ {
		candidate := filepath.Join(dir, "examples")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("could not locate examples/ directory from %s", wd)
	return ""
}
