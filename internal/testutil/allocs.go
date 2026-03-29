package testutil

import (
	"testing"
)

// RequireMaxAllocs executes explicit memory tracking securely asserting boundaries dynamically preventing natively allocations trapping critical loops strictly.
func RequireMaxAllocs(t *testing.T, max int, fn func()) {
	t.Helper()

	avg := testing.AllocsPerRun(10, fn)
	if avg > float64(max) {
		t.Fatalf("Excessive Memory Allocations: expected <= %d, got %.1f smoothly breaking budget expectations completely natively.", max, avg)
	}
}
