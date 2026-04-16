package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	c := DefaultConfig()
	if c.Theme != ThemeDefault {
		t.Errorf("Theme = %v, want default", c.Theme)
	}
	if c.BackgroundColor != "" {
		t.Error("BackgroundColor should be empty (use theme default)")
	}
}

func TestBuiltInThemes(t *testing.T) {
	for _, name := range []ThemeName{ThemeDefault, ThemeDark, ThemeForest, ThemeNeutral} {
		t.Run(string(name), func(t *testing.T) {
			th, err := BuiltInTheme(name)
			if err != nil {
				t.Fatalf("BuiltInTheme(%q): %v", name, err)
			}
			if th.Primary == "" {
				t.Error("Primary color should not be empty")
			}
			if th.Background == "" {
				t.Error("Background should not be empty")
			}
		})
	}
}

func TestBuiltInThemeUnknown(t *testing.T) {
	_, err := BuiltInTheme("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown theme")
	}
}

func TestLoadFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"theme": "dark", "backgroundColor": "#000"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if c.Theme != ThemeDark {
		t.Errorf("Theme = %v, want dark", c.Theme)
	}
	if c.BackgroundColor != "#000" {
		t.Errorf("BackgroundColor = %q, want #000", c.BackgroundColor)
	}
}

func TestLoadFileNotFound(t *testing.T) {
	_, err := LoadFile("/nonexistent/path.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadFileEmptyJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.json")
	if err := os.WriteFile(path, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if c.Theme != ThemeDefault {
		t.Errorf("Theme = %v, want default", c.Theme)
	}
}

func TestLoadFileInvalidTheme(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad-theme.json")
	if err := os.WriteFile(path, []byte(`{"theme": "nope"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadFile(path)
	if err == nil {
		t.Fatal("expected error for unknown theme in config")
	}
}

func TestLoadFileInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte(`{not json`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadFile(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestFlowchartTheme(t *testing.T) {
	th, _ := BuiltInTheme(ThemeDark)
	ft := th.FlowchartTheme()
	if ft.NodeFill == "" || ft.Background == "" {
		t.Errorf("FlowchartTheme should populate fields: %+v", ft)
	}
}

func TestSequenceTheme(t *testing.T) {
	th, _ := BuiltInTheme(ThemeForest)
	st := th.SequenceTheme()
	if st.ParticipantFill == "" || st.Background == "" {
		t.Errorf("SequenceTheme should populate fields: %+v", st)
	}
}

func TestPieColors(t *testing.T) {
	th, _ := BuiltInTheme(ThemeDefault)
	colors := th.PieSliceColors()
	if len(colors) == 0 {
		t.Error("PieColors should return at least one color")
	}
}

func TestLoadFileWithFlowchartConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	data := `{"theme": "default", "flowchart": {"fontSize": 20, "padding": 30}}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if c.Flowchart.FontSize != 20 {
		t.Errorf("Flowchart.FontSize = %v, want 20", c.Flowchart.FontSize)
	}
}
