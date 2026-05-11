package ggplot_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/TuSKan/ggplot"
	"github.com/TuSKan/ggplot/aes"
	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/dataset/memory"
	"github.com/TuSKan/ggplot/geom"
)

var updateGoldens = flag.Bool("update-goldens", false, "overwrite golden files with current output")

// goldenDir returns the platform-specific golden directory.
// Goldens differ by OS due to font rendering / anti-aliasing differences.
func goldenDir() string {
	return filepath.Join("testdata", "golden", runtime.GOOS+"_"+runtime.GOARCH)
}

// assertGolden compares rendered PNG bytes against a checked-in golden file.
//
// On first run (or with -update-goldens), the golden is written.
// On subsequent runs, the SHA-256 hash is compared.
func assertGolden(t *testing.T, name string, got []byte) {
	t.Helper()

	dir := goldenDir()
	path := filepath.Join(dir, name+".png")

	if *updateGoldens {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("create golden dir: %v", err)
		}

		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatalf("write golden %s: %v", path, err)
		}

		t.Logf("updated golden: %s (%d bytes, sha256:%s)", path, len(got), sha256hex(got))

		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		// No golden exists — generate it and skip (not fail).
		if os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatalf("create golden dir: %v", err)
			}

			if err := os.WriteFile(path, got, 0o644); err != nil {
				t.Fatalf("write initial golden %s: %v", path, err)
			}

			t.Logf("created initial golden: %s (%d bytes, sha256:%s)", path, len(got), sha256hex(got))
			t.Logf("re-run the test to validate against this golden")

			return
		}

		t.Fatalf("read golden %s: %v", path, err)
	}

	wantHash := sha256hex(want)

	gotHash := sha256hex(got)
	if wantHash != gotHash {
		// Write the actual output for diffing.
		actualPath := filepath.Join(dir, name+"_actual.png")
		_ = os.WriteFile(actualPath, got, 0o644)
		t.Fatalf("golden mismatch for %s:\n  want sha256: %s (%d bytes)\n  got  sha256: %s (%d bytes)\n  actual saved to: %s\n  run with -update-goldens to accept the new output",
			name, wantHash, len(want), gotHash, len(got), actualPath)
	}
}

func sha256hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// renderPNG is a helper that renders a plot to PNG bytes.
func renderPNG(t *testing.T, p *ggplot.Plot, w, h int) []byte {
	t.Helper()

	var buf bytes.Buffer

	_, err := p.WriteTo(context.Background(), &buf, "png", w, h)
	if err != nil {
		t.Fatalf("render PNG: %v", err)
	}

	if buf.Len() < 100 {
		t.Fatalf("rendered PNG too small: %d bytes", buf.Len())
	}

	return buf.Bytes()
}

// --- Golden snapshot tests ---
//
// Each test renders a deterministic plot (fixed data, fixed size, fixed theme)
// and compares the output to a checked-in golden PNG. This catches:
//   - Rendering regressions (geometry disappeared, axes shifted, colors changed)
//   - Theme/layout changes that affect pixel output
//   - Platform-specific rendering differences (caught per-GOOS golden)
//
// Usage:
//   go test -run TestGolden                     # validate against goldens
//   go test -run TestGolden -update-goldens     # regenerate goldens

func TestGolden_ScatterPlot(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}),
		eng.NewFloat64Column("y", []float64{2.1, 4.3, 3.0, 7.8, 5.5, 8.1, 6.9, 9.2, 8.5, 10.0}),
	)
	if err != nil {
		t.Fatal(err)
	}

	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point(geom.WithSize(4), geom.WithColor("#E74C3C"))).
		Labs(
			ggplot.Title("Scatter Plot Golden"),
			ggplot.XLab("X Values"),
			ggplot.YLab("Y Values"),
		)

	got := renderPNG(t, p, 400, 300)
	assertGolden(t, "scatter_plot", got)
}

func TestGolden_BarChart(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", []float64{1, 2, 3, 4, 5}),
		eng.NewFloat64Column("count", []float64{10, 25, 15, 30, 20}),
	)
	if err != nil {
		t.Fatal(err)
	}

	p := ggplot.New(ds, aes.X("x"), aes.Y("count")).
		Layer(geom.Bar(geom.WithFill("#3498DB"), geom.WithWidth(0.7))).
		Labs(
			ggplot.Title("Bar Chart Golden"),
			ggplot.XLab("Category"),
			ggplot.YLab("Count"),
		)

	got := renderPNG(t, p, 400, 300)
	assertGolden(t, "bar_chart", got)
}

func TestGolden_MultiLayer(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}),
		eng.NewFloat64Column("y", []float64{2.1, 4.3, 3.0, 7.8, 5.5, 8.1, 6.9, 9.2, 8.5, 10.0}),
	)
	if err != nil {
		t.Fatal(err)
	}

	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point(geom.WithColor("#E74C3C"), geom.WithSize(5))).
		Layer(geom.Line(geom.WithColor("#2980B9"), geom.WithLineWidth(2))).
		Labs(
			ggplot.Title("Multi-Layer Golden"),
			ggplot.XLab("X"),
			ggplot.YLab("Y"),
		)

	got := renderPNG(t, p, 400, 300)
	assertGolden(t, "multi_layer", got)
}

func TestGolden_GroupedColor(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", []float64{1, 2, 3, 1, 2, 3, 1, 2, 3}),
		eng.NewFloat64Column("y", []float64{1, 4, 9, 2, 5, 8, 3, 6, 7}),
		eng.NewStringColumn("group", []string{"A", "A", "A", "B", "B", "B", "C", "C", "C"}),
	)
	if err != nil {
		t.Fatal(err)
	}

	p := ggplot.New(ds, aes.X("x"), aes.Y("y"), aes.Color("group")).
		Layer(geom.Point(geom.WithSize(6))).
		Labs(ggplot.Title("Grouped Scatter Golden"))

	got := renderPNG(t, p, 500, 400)
	assertGolden(t, "grouped_color", got)
}

func TestGolden_Histogram(t *testing.T) {
	t.Parallel()

	// Deterministic histogram data (not random).
	xs := make([]float64, 200)
	for i := range xs {
		// Simple deterministic distribution: sawtooth pattern.
		xs[i] = float64(i%20) * 0.5
	}

	eng := memory.NewEngine(context.Background())

	ds, err := dataset.NewDataset(eng, eng.NewFloat64Column("x", xs))
	if err != nil {
		t.Fatal(err)
	}

	p := ggplot.New(ds, aes.X("x")).
		Layer(geom.Histogram(geom.WithBins(10), geom.WithFill("#9B59B6"))).
		Labs(ggplot.Title("Histogram Golden"))

	got := renderPNG(t, p, 400, 300)
	assertGolden(t, "histogram", got)
}

func TestGolden_LabelsAndTheme(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", []float64{1, 2, 3, 4, 5}),
		eng.NewFloat64Column("y", []float64{10, 20, 15, 25, 30}),
	)
	if err != nil {
		t.Fatal(err)
	}

	p := ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point(geom.WithSize(6), geom.WithColor("#E67E22"))).
		Layer(geom.Line(geom.WithColor("#2C3E50"), geom.WithLineWidth(2))).
		Labs(
			ggplot.Title("Full Labels Golden"),
			ggplot.Subtitle("Testing all label positions"),
			ggplot.XLab("X Axis Label"),
			ggplot.YLab("Y Axis Label"),
			ggplot.Caption("Source: test data"),
		).
		Theme("minimal").
		XLim(0, 6).
		YLim(0, 35)

	got := renderPNG(t, p, 600, 450)
	assertGolden(t, "labels_theme", got)
}

// TestGolden_Summary prints the golden directory and file count for CI visibility.
func TestGolden_Summary(t *testing.T) {
	t.Parallel()

	dir := goldenDir()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Logf("golden dir %s: %v", dir, err)
		return
	}

	var count int

	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".png" {
			count++
		}
	}

	fmt.Fprintf(os.Stderr, "golden directory: %s (%d files)\n", dir, count)
}
