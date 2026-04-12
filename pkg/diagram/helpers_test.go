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
