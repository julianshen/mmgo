package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func buildBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "mmdc")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = "."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	return bin
}

func TestCLIHelp(t *testing.T) {
	bin := buildBinary(t)
	out, err := exec.Command(bin, "--help").CombinedOutput()
	if err != nil {
		t.Fatalf("--help should not error: %v\n%s", err, out)
	}
	raw := string(out)
	if !strings.Contains(raw, "--input") || !strings.Contains(raw, "--output") {
		t.Errorf("help should list --input and --output flags:\n%s", raw)
	}
}

func TestCLINoInput(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected error when no input specified")
	}
	if !strings.Contains(string(out), "input") {
		t.Errorf("error should mention input: %s", out)
	}
}

func TestCLIFileToSVG(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	input := filepath.Join(dir, "test.mmd")
	output := filepath.Join(dir, "test.svg")
	if err := os.WriteFile(input, []byte("graph LR\n    A --> B"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(bin, "-i", input, "-o", output)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cli: %v\n%s", err, out)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if !strings.HasPrefix(string(data), "<?xml") {
		t.Error("output should be valid SVG")
	}
}

func TestCLIStdinToStdout(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "-i", "-")
	cmd.Stdin = strings.NewReader("graph LR\n    A --> B")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cli: %v\n%s", err, out)
	}
	if !strings.HasPrefix(string(out), "<?xml") {
		t.Error("stdout should contain SVG")
	}
}

func TestCLIWithTheme(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "-i", "-", "-t", "dark")
	cmd.Stdin = strings.NewReader("graph LR\n    A --> B")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cli: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "#1f2020") {
		t.Error("dark theme should apply")
	}
}

func TestCLIWithConfigFile(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(cfgPath, []byte(`{"theme": "forest"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(bin, "-i", "-", "-c", cfgPath)
	cmd.Stdin = strings.NewReader("graph LR\n    A --> B")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cli: %v\n%s", err, out)
	}
	if !strings.HasPrefix(string(out), "<?xml") {
		t.Error("should produce SVG")
	}
}

func TestCLIUnsupportedOutputFormat(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	input := filepath.Join(dir, "test.mmd")
	output := filepath.Join(dir, "test.png")
	if err := os.WriteFile(input, []byte("graph LR\n    A --> B"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(bin, "-i", input, "-o", output)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected error for .png output")
	}
	if !strings.Contains(string(out), "not yet supported") {
		t.Errorf("error should mention unsupported format: %s", out)
	}
}

func TestCLIQuietMode(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	input := filepath.Join(dir, "test.mmd")
	output := filepath.Join(dir, "test.svg")
	if err := os.WriteFile(input, []byte("graph LR\n    A --> B"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(bin, "-i", input, "-o", output, "-q")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cli: %v\n%s", err, out)
	}
	if len(out) != 0 {
		t.Errorf("quiet mode should produce no output, got: %s", out)
	}
}

func TestCLISequenceDiagram(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "-i", "-")
	cmd.Stdin = strings.NewReader("sequenceDiagram\n    A->>B: hello")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cli: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), ">hello<") {
		t.Error("sequence message label missing")
	}
}

func TestCLIPieDiagram(t *testing.T) {
	bin := buildBinary(t)
	cmd := exec.Command(bin, "-i", "-")
	cmd.Stdin = strings.NewReader("pie title Pets\n    \"Dogs\" : 70\n    \"Cats\" : 30")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cli: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), ">Pets<") {
		t.Error("pie title missing")
	}
}
