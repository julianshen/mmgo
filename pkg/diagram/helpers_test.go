package diagram

import (
	"fmt"
	"testing"
)

// checkStringer verifies that each enum value maps to the expected string.
func checkStringer[T interface {
	comparable
	fmt.Stringer
}](t *testing.T, cases map[T]string) {
	t.Helper()
	for v, want := range cases {
		if got := v.String(); got != want {
			t.Errorf("%T(%v).String() = %q, want %q", v, v, got, want)
		}
	}
}

// checkUniqueStringers verifies that each value has a non-empty, unique String().
func checkUniqueStringers[T fmt.Stringer](t *testing.T, values []T) {
	t.Helper()
	seen := make(map[string]bool)
	for _, v := range values {
		s := v.String()
		if s == "" {
			t.Errorf("%T(%v) has empty String()", v, v)
		}
		if seen[s] {
			t.Errorf("duplicate String(): %q", s)
		}
		seen[s] = true
	}
}
