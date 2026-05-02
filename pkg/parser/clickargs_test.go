package parser

import (
	"reflect"
	"testing"
)

func TestSplitClickArgsBasic(t *testing.T) {
	cases := []struct {
		in   string
		max  int
		want []string
	}{
		{`"https://example.com" "Open the docs" "_blank"`, 3, []string{"https://example.com", "Open the docs", "_blank"}},
		{`bare-url "tooltip"`, 3, []string{"bare-url", "tooltip"}},
		{``, 3, nil},
		{`"only one"`, 3, []string{"only one"}},
		// max caps the result; remaining content is dropped.
		{`"a" "b" "c" "d"`, 2, []string{"a", "b"}},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got, err := SplitClickArgs(tc.in, tc.max)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %#v, want %#v", got, tc.want)
			}
		})
	}
}

// An unterminated `"` must surface as an error; previously the
// scanner silently captured the rest of the line as the quoted run,
// producing misshapen URLs/tooltips with no diagnostic.
func TestSplitClickArgsUnterminatedQuote(t *testing.T) {
	for _, in := range []string{
		`"https://example.com`,
		`"first" "second-without-close`,
	} {
		t.Run(in, func(t *testing.T) {
			_, err := SplitClickArgs(in, 3)
			if err == nil {
				t.Errorf("expected error for unterminated quote in %q", in)
			}
		})
	}
}
