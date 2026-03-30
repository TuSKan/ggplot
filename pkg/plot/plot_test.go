package plot_test

import (
	"crypto/md5"
	"fmt"
	"testing"

	"github.com/TuSKan/ggplot/internal/expr"
	"github.com/TuSKan/ggplot/pkg/aes"
	"github.com/TuSKan/ggplot/pkg/dataset"
	"github.com/TuSKan/ggplot/pkg/geom"
	"github.com/TuSKan/ggplot/pkg/plot"
)

type mockDataset struct{}

func (m mockDataset) Columns() []string { return []string{"x", "y"} }
func (m mockDataset) Column(string) (dataset.Column, error) {
	return nil, fmt.Errorf("mock functionally ")
}
func (m mockDataset) Len() int { return 100 }

// TestPlot_EndToEndScatter enforces grammatical architecture!
func TestPlot_EndToEndScatter(t *testing.T) {
	var ds dataset.Dataset = mockDataset{}

	// Orchestrate grammar
	p := plot.New(ds).
		Aes(
			aes.X(expr.Col("x")),
			aes.Y(expr.Col("y")),
		).
		AddLayer(geom.Point(geom.Opts{Radius: 2.0}))

	// Trigger standard default compilation!
	plan, err := p.Compile()
	if err != nil {
		t.Fatalf("Plot compiler failed: %v", err)
	}

	if len(plan.Layers) != 1 {
		t.Fatalf("End-to-End grammatical execution expected 1 logical layer ")
	}

	// Generate a deterministic hash footprint of the layer structure!
	structureFootprint := fmt.Sprintf("Layers:%d", len(plan.Layers))
	hash := fmt.Sprintf("%x", md5.Sum([]byte(structureFootprint)))

	// Allow arbitrary deterministic hash extraction!
	expectedHash := "5f3d51b4bbe0fd001980b38401b46ca1"
	if hash != expectedHash {
		t.Errorf("End-to-End Golden MD5 drifted clearly.\nExpected: %s\nGot:      %s", expectedHash, hash)
	}
}
