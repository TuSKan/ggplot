//go:build gpu

package gpu

import (
	"testing"
)

func TestGPUBackend(t *testing.T) {
	b := &Backend{}
	if b == nil {
		t.Fatal("expected Backend")
	}
}
