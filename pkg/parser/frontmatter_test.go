package parser

import "testing"

func TestSplitFrontmatter(t *testing.T) {
	cases := []struct {
		name      string
		src       string
		wantFront string
		wantBody  string
	}{
		{
			name:      "no frontmatter",
			src:       "sequenceDiagram\nA->>B: hi\n",
			wantFront: "",
			wantBody:  "sequenceDiagram\nA->>B: hi\n",
		},
		{
			name:      "title only",
			src:       "---\ntitle: Hello\n---\nsequenceDiagram\nA->>B: hi\n",
			wantFront: "title: Hello\n",
			wantBody:  "sequenceDiagram\nA->>B: hi\n",
		},
		{
			name:      "leading blanks and comment",
			src:       "\n%% comment\n---\ntitle: x\n---\ngraph TD\n",
			wantFront: "title: x\n",
			wantBody:  "graph TD\n",
		},
		{
			name:      "unterminated frontmatter",
			src:       "---\ntitle: x\nsequenceDiagram\n",
			wantFront: "",
			wantBody:  "---\ntitle: x\nsequenceDiagram\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fm, body := SplitFrontmatter([]byte(tc.src))
			if string(fm) != tc.wantFront {
				t.Errorf("frontmatter = %q, want %q", fm, tc.wantFront)
			}
			if string(body) != tc.wantBody {
				t.Errorf("body = %q, want %q", body, tc.wantBody)
			}
		})
	}
}

func TestFrontmatterValue(t *testing.T) {
	body := []byte("title: Hello World\nconfig:\n  theme: dark\nfoo: 'quoted'\n")
	cases := map[string]string{
		"title":   "Hello World",
		"foo":     "quoted",
		"missing": "",
	}
	for key, want := range cases {
		t.Run(key, func(t *testing.T) {
			got := FrontmatterValue(body, key)
			if got != want {
				t.Errorf("%s = %q, want %q", key, got, want)
			}
		})
	}
}
