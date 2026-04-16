package markdown

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProcessNoMermaidBlocks(t *testing.T) {
	input := "# Hello\n\nJust regular markdown.\n"
	out, err := Process(strings.NewReader(input), t.TempDir(), "out", "svg", nil)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if string(out) != input {
		t.Errorf("output should be unchanged:\ngot:  %q\nwant: %q", out, input)
	}
}

func TestProcessSingleBlock(t *testing.T) {
	input := "# Title\n\n```mermaid\ngraph LR\n    A --> B\n```\n\nEnd.\n"
	dir := t.TempDir()
	out, err := Process(strings.NewReader(input), dir, "diagram", "svg", nil)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	raw := string(out)
	if strings.Contains(raw, "```mermaid") {
		t.Error("mermaid block should be replaced")
	}
	if !strings.Contains(raw, "![](diagram-1.svg)") {
		t.Errorf("should contain image ref, got:\n%s", raw)
	}
	data, err := os.ReadFile(filepath.Join(dir, "diagram-1.svg"))
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if !strings.HasPrefix(string(data), "<?xml") {
		t.Error("output file should be SVG")
	}
}

func TestProcessMultipleBlocks(t *testing.T) {
	input := "```mermaid\ngraph LR\n    A --> B\n```\n\nText.\n\n```mermaid\npie title X\n    \"A\" : 1\n```\n"
	dir := t.TempDir()
	out, err := Process(strings.NewReader(input), dir, "out", "svg", nil)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	raw := string(out)
	if !strings.Contains(raw, "![](out-1.svg)") || !strings.Contains(raw, "![](out-2.svg)") {
		t.Errorf("should contain both image refs:\n%s", raw)
	}
	if _, err := os.Stat(filepath.Join(dir, "out-1.svg")); err != nil {
		t.Error("out-1.svg should exist")
	}
	if _, err := os.Stat(filepath.Join(dir, "out-2.svg")); err != nil {
		t.Error("out-2.svg should exist")
	}
}

func TestProcessPNGFormat(t *testing.T) {
	input := "```mermaid\ngraph LR\n    A --> B\n```\n"
	dir := t.TempDir()
	out, err := Process(strings.NewReader(input), dir, "img", "png", nil)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if !strings.Contains(string(out), "![](img-1.png)") {
		t.Errorf("should reference .png: %s", out)
	}
	data, err := os.ReadFile(filepath.Join(dir, "img-1.png"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(data) < 4 || string(data[:4]) != "\x89PNG" {
		t.Error("should be valid PNG")
	}
}

func TestProcessPDFFormat(t *testing.T) {
	input := "```mermaid\ngraph LR\n    A --> B\n```\n"
	dir := t.TempDir()
	out, err := Process(strings.NewReader(input), dir, "doc", "pdf", nil)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if !strings.Contains(string(out), "![](doc-1.pdf)") {
		t.Errorf("should reference .pdf: %s", out)
	}
}

func TestProcessEmptyMermaidBlock(t *testing.T) {
	input := "```mermaid\n```\n"
	_, err := Process(strings.NewReader(input), t.TempDir(), "out", "svg", nil)
	if err == nil {
		t.Fatal("expected error for empty mermaid block")
	}
}

func TestProcessPreservesNonMermaidCodeBlocks(t *testing.T) {
	input := "```go\nfmt.Println(\"hi\")\n```\n"
	out, err := Process(strings.NewReader(input), t.TempDir(), "out", "svg", nil)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if string(out) != input {
		t.Errorf("non-mermaid blocks should be preserved:\ngot:  %q\nwant: %q", out, input)
	}
}
