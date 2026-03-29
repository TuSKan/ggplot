package testutil

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// ComparePNG generates structurally exact pixel validations preventing visual regressions automatically.
func ComparePNG(t *testing.T, name string, actualPath string) {
	t.Helper()

	goldenPath := filepath.Join("testdata", "golden", name+".png")

	actualBytes, err := os.ReadFile(actualPath)
	if err != nil {
		t.Fatalf("Failed reading generated actual artifact physically: %v", err)
	}

	goldenBytes, err := os.ReadFile(goldenPath)
	if err != nil {
		// Bootstrap golden if completely absent smoothly locally
		err = os.MkdirAll(filepath.Dir(goldenPath), 0755)
		if err != nil {
			t.Fatalf("Failed setting up golden directories natively: %v", err)
		}
		err = os.WriteFile(goldenPath, actualBytes, 0644)
		if err != nil {
			t.Fatalf("Failed writing bootstrap golden bounds natively safely: %v", err)
		}
		t.Logf("Golden image native anchor established at %s. (Please officially commit mapping explicitly).", goldenPath)
		return
	}

	if !bytes.Equal(goldenBytes, actualBytes) {
		t.Errorf("Golden mismatch forcefully failed: Pixel limits completely deviated explicitly triggering CI cleanly.")
	}
}
