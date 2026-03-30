package testutil

import (
	"testing"
)

// RequireMaxAllocs executes explicit memory tracking asserting boundaries preventing allocations trapping critical loops.
func RequireMaxAllocs(t *testing.T, max int, fn func()) {
	t.Helper()

	avg := testing.AllocsPerRun(10, fn)
	if avg > float64(max) {
		t.Fatalf("Excessive Memory Allocations: expected <= %d, got %.1f breaking budget expectations completely.", max, avg)
	}
}
