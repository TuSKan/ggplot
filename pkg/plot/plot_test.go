package plot_test

import (
	"fmt"
	"testing"

	"github.com/TuSKan/ggplot/internal/dataset"
	"github.com/TuSKan/ggplot/pkg/aes"
	"github.com/TuSKan/ggplot/pkg/geom"
	"github.com/TuSKan/ggplot/pkg/plot"
	"github.com/TuSKan/ggplot/pkg/stat"
)

func ExampleNew() {
	// A strictly mocked dataset representation
	var d dataset.Dataset = nil

	p := plot.New(d).
		AddLayer(
			geom.Point(geom.Opts{Radius: 1.5, Opacity: 0.1}),
			aes.X("PhaseX"), aes.Y("PhaseY"),
			aes.Color("Velocity"),
		).
		AddLayer(
			geom.Smooth(stat.MethodLoess), // Automatically calculates trendline
			aes.X("PhaseX"), aes.Y("PhaseY"),
		)

	// Since we expect all aesthetics (X,Y) to be satisfied, it should compile purely
	compiledPlan, err := p.Compile()
	if err != nil {
		fmt.Printf("Compile failed: %v", err)
	} else {
		fmt.Printf("Successfully compiled %d layers", len(compiledPlan.Layers))
	}

	// Output:
	// Successfully compiled 2 layers
}

func TestValidation_MissingAesthetics(t *testing.T) {
	// Geom needs an X and Y
	p := plot.New(dataset.Dataset(nil)).
		AddLayer(
			geom.Point(geom.Opts{}),
			aes.X("PhaseX"),
		)

	_, err := p.Compile()
	if err == nil {
		t.Errorf("Expected compile plan validation to fail due to missing Y, passed instead")
	}
}

func TestImmutability(t *testing.T) {
	root := plot.New(dataset.Dataset(nil))

	p1 := root.AddLayer(geom.Point(), aes.X("a"), aes.Y("b"))
	p2 := root.AddLayer(geom.Smooth(stat.MethodLoess), aes.X("a"), aes.Y("c"))

	_, err1 := p1.Compile()
	_, err2 := p2.Compile()

	if err1 != nil || err2 != nil {
		t.Fatalf("Failed cleanly compiling individual clones")
	}

	// Ensure root remains unmutated
	res, _ := root.Compile()
	if len(res.Layers) != 0 {
		t.Fatalf("Expected root object to stay immutable, but has %d layers", len(res.Layers))
	}
}
