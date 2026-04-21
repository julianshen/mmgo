// Package config provides Mermaid configuration loading and built-in
// theme definitions.
package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type ThemeName string

const (
	ThemeDefault ThemeName = "default"
	ThemeDark    ThemeName = "dark"
	ThemeForest  ThemeName = "forest"
	ThemeNeutral ThemeName = "neutral"
)

type Config struct {
	Theme           ThemeName       `json:"theme"`
	BackgroundColor string          `json:"backgroundColor,omitempty"`
	FontFamily      string          `json:"fontFamily,omitempty"`
	FontSize        float64         `json:"fontSize,omitempty"`
	Flowchart       FlowchartConfig `json:"flowchart,omitempty"`
	Sequence        SequenceConfig  `json:"sequence,omitempty"`
}

type FlowchartConfig struct {
	FontSize float64 `json:"fontSize,omitempty"`
	Padding  float64 `json:"padding,omitempty"`
}

type SequenceConfig struct {
	FontSize float64 `json:"fontSize,omitempty"`
	Padding  float64 `json:"padding,omitempty"`
}

func DefaultConfig() Config {
	return Config{Theme: ThemeDefault}
}

func LoadFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("config: invalid JSON: %w", err)
	}
	if c.Theme == "" {
		c.Theme = ThemeDefault
	}
	if _, err := BuiltInTheme(c.Theme); err != nil {
		return nil, err
	}
	return &c, nil
}

type ThemeColors struct {
	Primary    string
	Secondary  string
	Tertiary   string
	Text       string
	Background string
	LineColor  string
	NoteFill   string
	// MutedText is the de-emphasized text color used for stereotypes
	// like class «interface» tags, section dividers, and similar
	// secondary labels. Falls between Text and Background in contrast.
	MutedText string
	PieColors []string
}

func BuiltInTheme(name ThemeName) (*ThemeColors, error) {
	switch name {
	case ThemeDefault:
		return &ThemeColors{
			Primary:    "#fff",
			Secondary:  "#ECECFF",
			Tertiary:   "#eee",
			Text:       "#333",
			Background: "#fff",
			LineColor:  "#333",
			NoteFill:   "#ffffcc",
			MutedText:  "#999",
			PieColors:  []string{"#4e79a7", "#f28e2b", "#e15759", "#76b7b2", "#59a14f", "#edc948", "#b07aa1", "#ff9da7", "#9c755f", "#bab0ac"},
		}, nil
	case ThemeDark:
		return &ThemeColors{
			Primary:    "#1f2020",
			Secondary:  "#333",
			Tertiary:   "#444",
			Text:       "#ccc",
			Background: "#1f2020",
			LineColor:  "#81B1DB",
			NoteFill:   "#fff5ad",
			MutedText:  "#888",
			PieColors:  []string{"#81B1DB", "#FA6800", "#0F0F0F", "#CD5C5C", "#2E8B57", "#DAA520", "#BA55D3", "#FF69B4", "#8B4513", "#778899"},
		}, nil
	case ThemeForest:
		return &ThemeColors{
			Primary:    "#cde498",
			Secondary:  "#cdffb2",
			Tertiary:   "#eee",
			Text:       "#333",
			Background: "#fff",
			LineColor:  "#333",
			NoteFill:   "#ffffcc",
			MutedText:  "#666",
			PieColors:  []string{"#0b6623", "#2e8b57", "#50c878", "#8fbc8f", "#006400", "#3cb371", "#228b22", "#90ee90", "#32cd32", "#9acd32"},
		}, nil
	case ThemeNeutral:
		return &ThemeColors{
			Primary:    "#f4f4f4",
			Secondary:  "#eee",
			Tertiary:   "#ddd",
			Text:       "#333",
			Background: "#fff",
			LineColor:  "#666",
			NoteFill:   "#ffffcc",
			MutedText:  "#888",
			PieColors:  []string{"#4e79a7", "#f28e2b", "#e15759", "#76b7b2", "#59a14f", "#edc948", "#b07aa1", "#ff9da7", "#9c755f", "#bab0ac"},
		}, nil
	default:
		return nil, fmt.Errorf("config: unknown theme %q", name)
	}
}

