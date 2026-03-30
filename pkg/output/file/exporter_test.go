package file_test

import (
	"github.com/TuSKan/ggplot/pkg/output"
	"github.com/TuSKan/ggplot/pkg/output/file"
	"testing"
)

func TestFileExporter_Construct(t *testing.T) {
	e := file.NewFileExporter()
	if e == nil {
		t.Fatal("NewFileExporter returned nil")
	}

	err := e.Export(&output.Output{Width: 10, Height: 10}, "test.png")
	// Since Output has no Draw, it should fail immediately
	if err == nil || err.Error() != "no draw function provided in output" {
		t.Fatalf("expected 'no draw function provided in output' error, got %v", err)
	}
}
