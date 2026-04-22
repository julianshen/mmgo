package flowchart

import "testing"

func TestDefaultTheme(t *testing.T) {
	th := DefaultTheme()
	if th.NodeFill != "#ECECFF" {
		t.Errorf("NodeFill = %q, want %q", th.NodeFill, "#ECECFF")
	}
	if th.NodeStroke != "#9370DB" {
		t.Errorf("NodeStroke = %q, want %q", th.NodeStroke, "#9370DB")
	}
	if th.NodeText != "#333" {
		t.Errorf("NodeText = %q, want %q", th.NodeText, "#333")
	}
	if th.EdgeStroke != "#333" {
		t.Errorf("EdgeStroke = %q, want %q", th.EdgeStroke, "#333")
	}
	if th.EdgeText != "#333" {
		t.Errorf("EdgeText = %q, want %q", th.EdgeText, "#333")
	}
	if th.SubgraphFill != "#eee" {
		t.Errorf("SubgraphFill = %q, want %q", th.SubgraphFill, "#eee")
	}
	if th.SubgraphStroke != "#999" {
		t.Errorf("SubgraphStroke = %q, want %q", th.SubgraphStroke, "#999")
	}
	if th.SubgraphText != "#333" {
		t.Errorf("SubgraphText = %q, want %q", th.SubgraphText, "#333")
	}
	if th.Background != "#fff" {
		t.Errorf("Background = %q, want %q", th.Background, "#fff")
	}
}

func TestResolveBackground(t *testing.T) {
	defaultBg := "#fff"
	cases := []struct {
		name string
		opts *Options
		thBg string
		want string
	}{
		{"nil opts", nil, defaultBg, "#fff"},
		{"empty opts", &Options{}, defaultBg, "#fff"},
		{"theme bg passed in", nil, "#000", "#000"},
		{"background override", &Options{Background: "transparent"}, "#000", "transparent"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			th := Theme{Background: tc.thBg}
			got := resolveBackground(tc.opts, th)
			if got != tc.want {
				t.Errorf("resolveBackground() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResolveThemeBackground(t *testing.T) {
	opts := &Options{Theme: Theme{Background: "#000"}}
	th := resolveTheme(opts)
	if th.Background != "#000" {
		t.Errorf("resolveTheme().Background = %q, want #000", th.Background)
	}

	bg := resolveBackground(opts, th)
	if bg != "#000" {
		t.Errorf("combined resolution = %q, want #000", bg)
	}

	opts2 := &Options{Theme: Theme{Background: "#000"}, Background: "transparent"}
	th2 := resolveTheme(opts2)
	bg2 := resolveBackground(opts2, th2)
	if bg2 != "transparent" {
		t.Errorf("opts.Background should win = %q, want transparent", bg2)
	}
}

func TestResolveFontSize(t *testing.T) {
	if resolveFontSize(nil) != 16 {
		t.Errorf("nil opts should default to 16")
	}
	if resolveFontSize(&Options{}) != 16 {
		t.Errorf("zero FontSize should default to 16")
	}
	if resolveFontSize(&Options{FontSize: 20}) != 20 {
		t.Errorf("explicit FontSize should be used")
	}
}

func TestResolvePadding(t *testing.T) {
	if resolvePadding(nil) != 20 {
		t.Errorf("nil opts should default to 20")
	}
	if resolvePadding(&Options{}) != 20 {
		t.Errorf("zero Padding should default to 20")
	}
	if resolvePadding(&Options{Padding: 40}) != 40 {
		t.Errorf("explicit Padding should be used")
	}
}
