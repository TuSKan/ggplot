package stat

import (
	"context"
	"fmt"
	"maps"
	"math"
	"sort"

	"github.com/TuSKan/ggplot/dataset"
)

// ViolinOption configures the ViolinY transform.
type ViolinOption func(*violinConfig)

type violinConfig struct {
	Bandwidth float64 // KDE bandwidth; 0 = Silverman auto-select
	Points    int     // number of density evaluation points; 0 = 128
	Scale     string  // "area" (default), "count", or "width"
}

// WithViolinBandwidth sets the KDE bandwidth for violin density.
func WithViolinBandwidth(bw float64) ViolinOption {
	return func(c *violinConfig) { c.Bandwidth = bw }
}

// WithViolinPoints sets the number of density evaluation grid points.
func WithViolinPoints(n int) ViolinOption {
	return func(c *violinConfig) { c.Points = n }
}

// WithViolinScale sets the scaling method for violin widths.
//   - "area" (default): all violins have the same area
//   - "count": widths proportional to group size
//   - "width": all violins have the same maximum width
func WithViolinScale(s string) ViolinOption {
	return func(c *violinConfig) { c.Scale = s }
}

// ViolinY returns a Transform that computes a mirrored kernel density
// estimate on the Y values of each group. The output produces columns
// suitable for symmetric polygon rendering around each group's X center.
//
// Output columns: "x" (group center), "y" (density grid), "violinwidth"
// (scaled half-width at each y), "xmin" (x - violinwidth), "xmax" (x + violinwidth).
func ViolinY(opts ...ViolinOption) Transform {
	cfg := violinConfig{Points: 128, Scale: "area"} //nolint:mnd // 128 is a reasonable default grid size for violin KDE.
	for _, o := range opts {
		o(&cfg)
	}

	return &violinTransform{cfg: cfg}
}

type violinTransform struct {
	cfg violinConfig
}

type violinGroupResult struct {
	pos     float64
	grid    []float64
	density []float64
	count   int
}

func (v *violinTransform) Name() string { return "violinY" }

func (v *violinTransform) OutputSchema() []string {
	return []string{"x", "y", "violinwidth", "xmin", "xmax"}
}

func (v *violinTransform) OutputMapping() map[string]string {
	return map[string]string{"x": "x", "y": "y"}
}

func (v *violinTransform) OutputHints() map[string]ChannelHint { return nil }

func (v *violinTransform) Apply(ctx context.Context, in TransformInput) (TransformResult, error) {
	const label = "violinY"

	yCol := in.Mapping["y"]
	if yCol == "" {
		return TransformResult{}, fmt.Errorf("%s: missing 'y' aesthetic: %w", label, ErrMissingColumn)
	}

	xCol := in.Mapping["x"]
	if xCol == "" {
		return TransformResult{}, fmt.Errorf("%s: missing 'x' aesthetic: %w", label, ErrMissingColumn)
	}

	eng := dataset.GetEngine(in.Data.Table())
	if eng == nil {
		return TransformResult{}, fmt.Errorf("%s: no engine: %w", label, ErrUnsupportedType)
	}

	cf, ok := eng.(dataset.ColumnFactory)
	if !ok {
		return TransformResult{}, fmt.Errorf("%s: engine %q: ColumnFactory: %w", label, eng.Name(), ErrUnsupportedType)
	}

	sk, ok := eng.(dataset.StatKernel)
	if !ok {
		return TransformResult{}, fmt.Errorf("%s: engine %q: StatKernel: %w", label, eng.Name(), ErrUnsupportedType)
	}

	// Get raw data.
	xVals, err := in.Data.Float64(xCol)
	if err != nil {
		// X might be categorical/string — try string and assign numeric positions.
		xStrVals, serr := in.Data.Strings(xCol)
		if serr != nil {
			return TransformResult{}, fmt.Errorf("%s: x column: %w", label, err)
		}

		return v.applyGrouped(ctx, in, xStrVals, yCol, cf, sk)
	}

	// Continuous X — group by distinct X values.
	groups := groupByFloat64(xVals)

	return v.computeViolin(ctx, in, groups, cf, sk, yCol)
}

// groupByFloat64 groups row indices by distinct float64 values.
func groupByFloat64(xs []float64) map[float64][]int {
	groups := make(map[float64][]int)
	for i, x := range xs {
		groups[x] = append(groups[x], i)
	}

	return groups
}

func (v *violinTransform) applyGrouped(
	ctx context.Context,
	in TransformInput,
	xLabels []string,
	yCol string,
	cf dataset.ColumnFactory,
	sk dataset.StatKernel,
) (TransformResult, error) {
	const label = "violinY"

	yVals, err := in.Data.Float64(yCol)
	if err != nil {
		return TransformResult{}, fmt.Errorf("%s: y column: %w", label, err)
	}

	// Assign numeric positions to categories.
	catOrder := make([]string, 0)
	catPos := make(map[string]float64)

	for _, s := range xLabels {
		if _, ok := catPos[s]; !ok {
			catPos[s] = float64(len(catOrder))
			catOrder = append(catOrder, s)
		}
	}

	// Build float groups.
	groups := make(map[float64][]int)

	for i, s := range xLabels {
		pos := catPos[s]
		groups[pos] = append(groups[pos], i)
	}

	// Build y values indexed.
	indexedY := make(map[float64][]float64, len(groups))
	for pos, indices := range groups {
		vals := make([]float64, len(indices))
		for j, idx := range indices {
			vals[j] = yVals[idx]
		}

		indexedY[pos] = vals
	}

	return v.computeViolinFromValues(ctx, in, indexedY, cf, sk)
}

func (v *violinTransform) computeViolin(
	ctx context.Context,
	in TransformInput,
	groups map[float64][]int,
	cf dataset.ColumnFactory,
	sk dataset.StatKernel,
	yCol string,
) (TransformResult, error) {
	const label = "violinY"

	yVals, err := in.Data.Float64(yCol)
	if err != nil {
		return TransformResult{}, fmt.Errorf("%s: y column: %w", label, err)
	}

	indexedY := make(map[float64][]float64, len(groups))
	for pos, indices := range groups {
		vals := make([]float64, len(indices))
		for j, idx := range indices {
			vals[j] = yVals[idx]
		}

		indexedY[pos] = vals
	}

	return v.computeViolinFromValues(ctx, in, indexedY, cf, sk)
}

func (v *violinTransform) computeViolinFromValues(
	ctx context.Context,
	in TransformInput,
	indexedY map[float64][]float64,
	cf dataset.ColumnFactory,
	sk dataset.StatKernel,
) (TransformResult, error) {
	const label = "violinY"

	nPts := v.cfg.Points

	// Sort group positions for deterministic output.
	positions := make([]float64, 0, len(indexedY))
	for pos := range indexedY {
		positions = append(positions, pos)
	}

	sort.Float64s(positions)

	results := make([]violinGroupResult, 0, len(positions))

	for _, pos := range positions {
		vals := indexedY[pos]
		if len(vals) < 2 { //nolint:mnd // Need at least 2 points for KDE.
			continue
		}

		col := cf.NewFloat64Column("_violin_y", vals)

		kdeTbl, err := sk.KDE(ctx, col, v.cfg.Bandwidth, nPts)
		if err != nil {
			return TransformResult{}, fmt.Errorf("%s: KDE for group %.0f: %w", label, pos, err)
		}

		kdeDS := dataset.From(kdeTbl)

		grid, err := kdeDS.Float64("x")
		if err != nil {
			return TransformResult{}, fmt.Errorf("%s: KDE grid: %w", label, err)
		}

		density, err := kdeDS.Float64("density")
		if err != nil {
			return TransformResult{}, fmt.Errorf("%s: KDE density: %w", label, err)
		}

		results = append(results, violinGroupResult{
			pos:     pos,
			grid:    grid,
			density: density,
			count:   len(vals),
		})
	}

	if len(results) == 0 {
		return TransformResult(in), nil
	}

	// Scale densities according to v.cfg.Scale.
	v.scaleDensities(results)

	// Assemble output columns.
	totalRows := 0
	for _, r := range results {
		totalRows += len(r.grid)
	}

	outX := make([]float64, 0, totalRows)
	outY := make([]float64, 0, totalRows)
	outWidth := make([]float64, 0, totalRows)
	outXmin := make([]float64, 0, totalRows)
	outXmax := make([]float64, 0, totalRows)

	for _, r := range results {
		for i, y := range r.grid {
			w := r.density[i]
			outX = append(outX, r.pos)
			outY = append(outY, y)
			outWidth = append(outWidth, w)
			outXmin = append(outXmin, r.pos-w)
			outXmax = append(outXmax, r.pos+w)
		}
	}

	schema := dataset.NewSchema(
		dataset.FloatCol("x"),
		dataset.FloatCol("y"),
		dataset.FloatCol("violinwidth"),
		dataset.FloatCol("xmin"),
		dataset.FloatCol("xmax"),
	)

	tbl, err := cf.FromColumns(schema,
		cf.NewFloat64Column("x", outX),
		cf.NewFloat64Column("y", outY),
		cf.NewFloat64Column("violinwidth", outWidth),
		cf.NewFloat64Column("xmin", outXmin),
		cf.NewFloat64Column("xmax", outXmax),
	)
	if err != nil {
		return TransformResult{}, fmt.Errorf("%s: assemble: %w", label, err)
	}

	outMapping := make(map[string]string, len(in.Mapping)+len(v.OutputMapping()))
	maps.Copy(outMapping, in.Mapping)
	maps.Copy(outMapping, v.OutputMapping())

	return TransformResult{Data: dataset.From(tbl), Mapping: outMapping}, nil
}

func (v *violinTransform) scaleDensities(results []violinGroupResult) {
	switch v.cfg.Scale {
	case "width":
		// Normalize each group's max density to 1.
		for i := range results {
			maxD := 0.0
			for _, d := range results[i].density {
				maxD = math.Max(maxD, d)
			}

			if maxD > 0 {
				for j := range results[i].density {
					results[i].density[j] /= maxD
				}
			}
		}

	case "count":
		// Scale by group count relative to largest group.
		maxCount := 0
		for _, r := range results {
			if r.count > maxCount {
				maxCount = r.count
			}
		}

		for i := range results {
			// First normalize to width, then scale by count ratio.
			maxD := 0.0
			for _, d := range results[i].density {
				maxD = math.Max(maxD, d)
			}

			if maxD > 0 {
				scale := float64(results[i].count) / float64(maxCount)
				for j := range results[i].density {
					results[i].density[j] = results[i].density[j] / maxD * scale
				}
			}
		}

	default: // "area" — normalize so all violins have equal area.
		// The KDE from StatKernel is already area-normalized.
		// Just normalize each to unit max for consistent visual width.
		globalMax := 0.0

		for _, r := range results {
			for _, d := range r.density {
				globalMax = math.Max(globalMax, d)
			}
		}

		if globalMax > 0 {
			for i := range results {
				for j := range results[i].density {
					results[i].density[j] /= globalMax
				}
			}
		}
	}

	// Scale all densities by 0.5 so they represent half-widths.
	halfWidth := 0.5 //nolint:mnd // Half-width: density represents one side of the mirrored violin.

	for i := range results {
		for j := range results[i].density {
			results[i].density[j] *= halfWidth
		}
	}
}
