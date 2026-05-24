package stat

import (
	"context"
	"fmt"
	"maps"
	"math"
	"sort"

	"github.com/TuSKan/ggplot/dataset"
)

// DotBinOption configures the DotBin transform.
type DotBinOption func(*dotBinConfig)

type dotBinConfig struct {
	BinWidth float64 // dot bin width; 0 = auto from data range and dot size
	Method   string  // "dotdensity" (Wilkinson-style, default) or "histodot"
}

// WithDotBinWidth sets an explicit bin width for the dot plot.
func WithDotBinWidth(w float64) DotBinOption {
	return func(c *dotBinConfig) { c.BinWidth = w }
}

// WithDotMethod sets the binning method: "dotdensity" (default) or "histodot".
//   - "dotdensity": Wilkinson-style greedy binning (dots packed tightly)
//   - "histodot": regular histogram bins with dots stacked
func WithDotMethod(m string) DotBinOption {
	return func(c *dotBinConfig) { c.Method = m }
}

// DotBin returns a Transform that bins observations for dot plots.
// Each observation becomes one dot, stacked vertically within its bin.
//
// Output columns: "x" (bin center), "y" (stacked position from 0),
// "count" (number of dots in each bin).
func DotBin(opts ...DotBinOption) Transform {
	cfg := dotBinConfig{Method: "dotdensity"}
	for _, o := range opts {
		o(&cfg)
	}

	return &dotBinTransform{cfg: cfg}
}

type dotBinTransform struct {
	cfg dotBinConfig
}

func (d *dotBinTransform) Name() string { return "dotbin" }

func (d *dotBinTransform) OutputSchema() []string {
	return []string{"x", "y", "count"}
}

func (d *dotBinTransform) OutputMapping() map[string]string {
	return map[string]string{"x": "x", "y": "y"}
}

func (d *dotBinTransform) OutputHints() map[string]ChannelHint {
	return map[string]ChannelHint{"y": HintCount}
}

func (d *dotBinTransform) Apply(ctx context.Context, in TransformInput) (TransformResult, error) {
	const label = "dotbin"

	_ = ctx

	srcCol := in.Mapping["x"]
	if srcCol == "" {
		return TransformResult{}, fmt.Errorf("%s: missing 'x' aesthetic: %w", label, ErrMissingColumn)
	}

	xVals, err := in.Data.Float64(srcCol)
	if err != nil {
		return TransformResult{}, fmt.Errorf("%s: %w", label, err)
	}

	if len(xVals) == 0 {
		return TransformResult(in), nil
	}

	eng := dataset.GetEngine(in.Data.Table())
	if eng == nil {
		return TransformResult{}, fmt.Errorf("%s: no engine: %w", label, ErrUnsupportedType)
	}

	cf, ok := eng.(dataset.ColumnFactory)
	if !ok {
		return TransformResult{}, fmt.Errorf("%s: engine %q: ColumnFactory: %w", label, eng.Name(), ErrUnsupportedType)
	}

	var outX, outY []float64

	var outCount []float64

	switch d.cfg.Method {
	case "histodot":
		outX, outY, outCount = d.histodot(xVals)
	default: // "dotdensity" — Wilkinson-style
		outX, outY, outCount = d.wilkinson(xVals)
	}

	schema := dataset.NewSchema(
		dataset.FloatCol("x"),
		dataset.FloatCol("y"),
		dataset.FloatCol("count"),
	)

	tbl, err := cf.FromColumns(schema,
		cf.NewFloat64Column("x", outX),
		cf.NewFloat64Column("y", outY),
		cf.NewFloat64Column("count", outCount),
	)
	if err != nil {
		return TransformResult{}, fmt.Errorf("%s: assemble: %w", label, err)
	}

	outMapping := make(map[string]string, len(in.Mapping)+len(d.OutputMapping()))
	maps.Copy(outMapping, in.Mapping)
	maps.Copy(outMapping, d.OutputMapping())

	return TransformResult{Data: dataset.From(tbl), Mapping: outMapping}, nil
}

// wilkinson implements Wilkinson-style greedy dot binning.
// Data is sorted, then each point is assigned to the nearest existing
// bin if within binWidth, or starts a new bin.
func (d *dotBinTransform) wilkinson(xVals []float64) (outX, outY, outCount []float64) {
	sorted := make([]float64, len(xVals))
	copy(sorted, xVals)
	sort.Float64s(sorted)

	binWidth := d.cfg.BinWidth
	if binWidth <= 0 {
		binWidth = d.autoBinWidth(sorted)
	}

	type dotBin struct {
		center float64
		count  int
	}

	var bins []dotBin

	for _, x := range sorted {
		placed := false

		for i := range bins {
			if math.Abs(x-bins[i].center) <= binWidth/2 { //nolint:mnd // half-width comparison for bin membership
				bins[i].count++

				placed = true

				break
			}
		}

		if !placed {
			bins = append(bins, dotBin{center: x, count: 1})
		}
	}

	// Expand bins into individual dot rows.
	totalDots := 0
	for _, b := range bins {
		totalDots += b.count
	}

	outX = make([]float64, 0, totalDots)
	outY = make([]float64, 0, totalDots)
	outCount = make([]float64, 0, totalDots)

	for _, b := range bins {
		for j := range b.count {
			outX = append(outX, b.center)
			outY = append(outY, float64(j))
			outCount = append(outCount, float64(b.count))
		}
	}

	return outX, outY, outCount
}

// histodot uses regular equal-width bins (like a histogram) but outputs
// one row per dot stacked within each bin.
func (d *dotBinTransform) histodot(xVals []float64) (outX, outY, outCount []float64) {
	sorted := make([]float64, len(xVals))
	copy(sorted, xVals)
	sort.Float64s(sorted)

	binWidth := d.cfg.BinWidth
	if binWidth <= 0 {
		binWidth = d.autoBinWidth(sorted)
	}

	xMin := sorted[0]
	xMax := sorted[len(sorted)-1]

	nBins := max(int(math.Ceil((xMax-xMin)/binWidth)), 1)

	// Count observations per bin.
	counts := make([]int, nBins)
	centers := make([]float64, nBins)

	for i := range nBins {
		centers[i] = xMin + (float64(i)+0.5)*binWidth //nolint:mnd // 0.5 offset to center of bin
	}

	for _, x := range sorted {
		idx := int((x - xMin) / binWidth)
		if idx >= nBins {
			idx = nBins - 1
		}

		counts[idx]++
	}

	// Expand into dot rows.
	totalDots := 0
	for _, c := range counts {
		totalDots += c
	}

	outX = make([]float64, 0, totalDots)
	outY = make([]float64, 0, totalDots)
	outCount = make([]float64, 0, totalDots)

	for i, c := range counts {
		for j := range c {
			outX = append(outX, centers[i])
			outY = append(outY, float64(j))
			outCount = append(outCount, float64(c))
		}
	}

	return outX, outY, outCount
}

// autoBinWidth computes a reasonable bin width from the data range.
// Uses Freedman-Diaconis style heuristic: binWidth ≈ range / sqrt(n).
func (d *dotBinTransform) autoBinWidth(sorted []float64) float64 {
	if len(sorted) < 2 { //nolint:mnd // Need at least 2 values for a range.
		return 1
	}

	dataRange := sorted[len(sorted)-1] - sorted[0]
	if dataRange <= 0 {
		return 1
	}

	nBins := math.Sqrt(float64(len(sorted)))

	return dataRange / nBins
}
