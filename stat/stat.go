// Package stat provides statistical transformations for the Grammar of Graphics.
// Stats compute derived data (bins, counts, densities, smooths) from raw data,
// producing new datasets that are then rendered by geometries.
//
// Each stat implements the [Stat] interface and is registered with a typed
// [Name] for lookup during pipeline compilation.
package stat

import (
	"context"
	"fmt"
	"math"
	"sort"

	"github.com/TuSKan/ggplot/dataset"
)

// Name identifies a statistical transformation.
type Name string

const (
	Identity Name = "identity"
	Bin      Name = "bin"
	Count    Name = "count"
	Density  Name = "density"
	Smooth   Name = "smooth"
	Summary  Name = "summary"
	Boxplot  Name = "boxplot"
)

// Options holds typed parameters for stat computations,
// replacing magic string keys like "__bins".
type Options struct {
	Bins   int    // number of bins (histogram/bin stat)
	Method string // smoothing method: "lm", "loess"
	Points int    // number of output grid points
}

// Stat computes a statistical transformation on a dataset, producing a
// new (possibly differently shaped) dataset. The transformed dataset's
// columns depend on the stat type.
type Stat interface {
	// Name returns the stat's typed identifier.
	Name() Name

	// RequiredAes returns the aesthetic channels this stat needs.
	RequiredAes() []string

	// OutputSchema returns the column names this stat produces.
	OutputSchema() []string

	// Compute performs the transformation.
	Compute(ctx context.Context, ds dataset.Dataset, mapping map[string]string, opts Options) (dataset.Dataset, error)
}

// --- Registry ---

var registry = map[Name]Stat{
	Identity: identityStat{},
}

// Register adds a stat to the global registry.
func Register(s Stat) { registry[s.Name()] = s }

// Lookup returns a stat by name. Returns an error for unknown names.
func Lookup(name Name) (Stat, error) {
	if s, ok := registry[name]; ok {
		return s, nil
	}
	return nil, fmt.Errorf("stat: unknown stat %q", name)
}

func init() {
	Register(binStat{})
	Register(countStat{})
	Register(densityStat{})
	Register(smoothStat{})
	Register(summaryStat{})
	Register(boxplotStat{})
}

// --- Identity ---

type identityStat struct{}

func (identityStat) Name() Name             { return Identity }
func (identityStat) RequiredAes() []string  { return nil }
func (identityStat) OutputSchema() []string { return nil }
func (identityStat) Compute(_ context.Context, ds dataset.Dataset, _ map[string]string, _ Options) (dataset.Dataset, error) {
	return ds, nil
}

// --- Bin ---

type binStat struct{}

func (binStat) Name() Name             { return Bin }
func (binStat) RequiredAes() []string  { return []string{"x"} }
func (binStat) OutputSchema() []string { return []string{"x", "count", "xmin", "xmax"} }

func (binStat) Compute(_ context.Context, ds dataset.Dataset, mapping map[string]string, opts Options) (dataset.Dataset, error) {
	xCol := mapping["x"]
	if xCol == "" {
		return dataset.Dataset{}, fmt.Errorf("stat_bin: missing 'x' aesthetic")
	}

	vals, err := collectFloat64Column(ds, xCol)
	if err != nil {
		return dataset.Dataset{}, err
	}

	// Determine number of bins: explicit opts.Bins > Sturges' rule.
	nBins := opts.Bins
	if nBins <= 0 {
		// Sturges' rule: k = ceil(1 + log2(n))
		nBins = int(math.Ceil(1.0 + math.Log2(float64(len(vals)))))
	}
	if nBins <= 0 {
		nBins = 1
	}
	if nBins > len(vals) {
		nBins = len(vals)
	}

	min, max := vals[0], vals[0]
	for _, v := range vals {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	binWidth := (max - min) / float64(nBins)
	if binWidth <= 0 {
		binWidth = 1
	}

	centers := make([]float64, nBins)
	counts := make([]float64, nBins)
	for i := range centers {
		centers[i] = min + binWidth*(float64(i)+0.5)
	}

	for _, v := range vals {
		idx := int((v - min) / binWidth)
		if idx >= nBins {
			idx = nBins - 1
		}
		if idx < 0 {
			idx = 0
		}
		counts[idx]++
	}

	return newFloat64Dataset(ds, map[string][]float64{
		"x":     centers,
		"count": counts,
	})
}

// --- Count ---

type countStat struct{}

func (countStat) Name() Name             { return Count }
func (countStat) RequiredAes() []string  { return []string{"x"} }
func (countStat) OutputSchema() []string { return []string{"x", "count"} }

func (countStat) Compute(_ context.Context, ds dataset.Dataset, mapping map[string]string, _ Options) (dataset.Dataset, error) {
	xCol := mapping["x"]
	if xCol == "" {
		return dataset.Dataset{}, fmt.Errorf("stat_count: missing 'x' aesthetic")
	}

	vals, err := collectFloat64Column(ds, xCol)
	if err != nil {
		return dataset.Dataset{}, err
	}

	// Count unique values.
	unique := make(map[float64]int)
	var order []float64
	for _, v := range vals {
		if _, exists := unique[v]; !exists {
			order = append(order, v)
		}
		unique[v]++
	}
	sort.Float64s(order)

	xs := make([]float64, len(order))
	counts := make([]float64, len(order))
	for i, v := range order {
		xs[i] = v
		counts[i] = float64(unique[v])
	}

	return newFloat64Dataset(ds, map[string][]float64{
		"x":     xs,
		"count": counts,
	})
}

// --- Density ---

type densityStat struct{}

func (densityStat) Name() Name             { return Density }
func (densityStat) RequiredAes() []string  { return []string{"x"} }
func (densityStat) OutputSchema() []string { return []string{"x", "density"} }

func (densityStat) Compute(ctx context.Context, ds dataset.Dataset, mapping map[string]string, opts Options) (dataset.Dataset, error) {
	xCol := mapping["x"]
	if xCol == "" {
		return dataset.Dataset{}, fmt.Errorf("stat_density: missing 'x' aesthetic")
	}

	vals, err := collectFloat64Column(ds, xCol)
	if err != nil {
		return dataset.Dataset{}, err
	}
	sort.Float64s(vals)

	n := len(vals)
	if n == 0 {
		return newFloat64Dataset(ds, map[string][]float64{"x": {}, "density": {}})
	}

	// Silverman's rule-of-thumb bandwidth.
	mean := 0.0
	for _, v := range vals {
		mean += v
	}
	mean /= float64(n)

	variance := 0.0
	for _, v := range vals {
		d := v - mean
		variance += d * d
	}
	variance /= float64(n)
	sd := math.Sqrt(variance)
	if sd == 0 {
		sd = 1
	}
	bandwidth := 1.06 * sd * math.Pow(float64(n), -0.2)

	// Evaluate KDE.
	points := 512
	xmin, xmax := vals[0]-3*bandwidth, vals[n-1]+3*bandwidth
	step := (xmax - xmin) / float64(points-1)

	xs := make([]float64, points)
	ys := make([]float64, points)

	for i := 0; i < points; i++ {
		if i%64 == 0 {
			if err := ctx.Err(); err != nil {
				return dataset.Dataset{}, err
			}
		}
		x := xmin + float64(i)*step
		xs[i] = x
		density := 0.0
		for _, v := range vals {
			z := (x - v) / bandwidth
			density += math.Exp(-0.5*z*z) / (bandwidth * math.Sqrt(2*math.Pi))
		}
		ys[i] = density / float64(n)
	}

	return newFloat64Dataset(ds, map[string][]float64{"x": xs, "density": ys})
}

// --- Smooth ---

type smoothStat struct{}

func (smoothStat) Name() Name             { return Smooth }
func (smoothStat) RequiredAes() []string  { return []string{"x", "y"} }
func (smoothStat) OutputSchema() []string { return []string{"x", "y"} }

func (smoothStat) Compute(ctx context.Context, ds dataset.Dataset, mapping map[string]string, opts Options) (dataset.Dataset, error) {
	xCol := mapping["x"]
	yCol := mapping["y"]
	if xCol == "" || yCol == "" {
		return dataset.Dataset{}, fmt.Errorf("stat_smooth: missing 'x' or 'y' aesthetic")
	}

	xData, err := collectFloat64Column(ds, xCol)
	if err != nil {
		return dataset.Dataset{}, err
	}
	yData, err := collectFloat64Column(ds, yCol)
	if err != nil {
		return dataset.Dataset{}, err
	}
	if len(xData) != len(yData) {
		return dataset.Dataset{}, fmt.Errorf("stat_smooth: x and y columns have different lengths")
	}

	n := len(xData)
	if n < 2 {
		return newFloat64Dataset(ds, map[string][]float64{"x": xData, "y": yData})
	}

	// Sort by X for consistent ordering.
	type xy struct{ x, y float64 }
	pts := make([]xy, n)
	for i := range xData {
		pts[i] = xy{xData[i], yData[i]}
	}
	sort.Slice(pts, func(i, j int) bool { return pts[i].x < pts[j].x })

	xMin, xMax := pts[0].x, pts[n-1].x

	// LOESS parameters.
	alpha := 0.3 // bandwidth: fraction of data used per local fit
	if n < 20 {
		alpha = 0.75
	} else if n < 50 {
		alpha = 0.5
	}

	// Number of output points (from opts or default 80).
	nOut := opts.Points
	if nOut <= 0 {
		nOut = 80
	}
	if nOut > n {
		nOut = n
	}
	step := (xMax - xMin) / float64(nOut-1)

	xs := make([]float64, nOut)
	ys := make([]float64, nOut)

	k := int(math.Ceil(alpha * float64(n)))
	if k < 3 {
		k = 3
	}
	if k > n {
		k = n
	}

	// Sliding window: since pts is sorted and xEval advances monotonically,
	// maintain a [lo, hi) window of size k instead of sorting distances per point.
	lo, hi := 0, k
	for i := 0; i < nOut; i++ {
		if i%32 == 0 {
			if err := ctx.Err(); err != nil {
				return dataset.Dataset{}, err
			}
		}
		xEval := xMin + float64(i)*step
		xs[i] = xEval

		// Advance window: expand hi rightward while it's closer than lo.
		for hi < n && math.Abs(pts[hi].x-xEval) < math.Abs(pts[lo].x-xEval) {
			lo++
			hi++
		}

		maxDist := math.Max(math.Abs(pts[lo].x-xEval), math.Abs(pts[hi-1].x-xEval))
		if maxDist < 1e-12 {
			maxDist = 1e-12
		}

		// Weighted local linear regression over pts[lo:hi].
		var sw, swx, swy, swxx, swxy float64
		for j := lo; j < hi; j++ {
			u := math.Abs(pts[j].x-xEval) / maxDist
			if u >= 1.0 {
				continue
			}
			w := (1 - u*u*u)
			w = w * w * w // tri-cube
			dx := pts[j].x - xEval
			sw += w
			swx += w * dx
			swy += w * pts[j].y
			swxx += w * dx * dx
			swxy += w * dx * pts[j].y
		}

		if sw < 1e-15 {
			ys[i] = 0
			continue
		}
		det := sw*swxx - swx*swx
		if math.Abs(det) < 1e-15 {
			ys[i] = swy / sw
		} else {
			a := (swxx*swy - swx*swxy) / det
			ys[i] = a
		}
	}

	return newFloat64Dataset(ds, map[string][]float64{"x": xs, "y": ys})
}

// --- Summary ---

type summaryStat struct{}

func (summaryStat) Name() Name             { return Summary }
func (summaryStat) RequiredAes() []string  { return []string{"x", "y"} }
func (summaryStat) OutputSchema() []string { return []string{"x", "y"} }

func (summaryStat) Compute(_ context.Context, ds dataset.Dataset, mapping map[string]string, _ Options) (dataset.Dataset, error) {
	// summaryStat computes mean(y) for each distinct x value.
	xCol := mapping["x"]
	yCol := mapping["y"]
	if xCol == "" || yCol == "" {
		return dataset.Dataset{}, fmt.Errorf("stat_summary: missing 'x' or 'y' aesthetic")
	}

	xData, err := collectFloat64Column(ds, xCol)
	if err != nil {
		return dataset.Dataset{}, err
	}
	yData, err := collectFloat64Column(ds, yCol)
	if err != nil {
		return dataset.Dataset{}, err
	}

	groups := make(map[float64][]float64)
	var order []float64
	for i, x := range xData {
		if _, exists := groups[x]; !exists {
			order = append(order, x)
		}
		groups[x] = append(groups[x], yData[i])
	}
	sort.Float64s(order)

	xs := make([]float64, len(order))
	means := make([]float64, len(order))
	for i, x := range order {
		xs[i] = x
		sum := 0.0
		for _, v := range groups[x] {
			sum += v
		}
		means[i] = sum / float64(len(groups[x]))
	}

	return newFloat64Dataset(ds, map[string][]float64{"x": xs, "y": means})
}

// --- Helpers ---

func collectFloat64(col dataset.AnyColumn) ([]float64, error) {
	fc, ok := col.(dataset.Column[float64])
	if !ok {
		return nil, fmt.Errorf("stat: column %q (%s) is not float64", col.Name(), col.DType())
	}
	vals := fc.Values()
	out := make([]float64, 0, len(vals))
	for _, v := range vals {
		if !math.IsNaN(v) {
			out = append(out, v)
		}
	}
	return out, nil
}

func collectFloat64Column(ds dataset.Dataset, name string) ([]float64, error) {
	col, err := ds.Column(name)
	if err != nil {
		return nil, err
	}
	return collectFloat64(col)
}

// newFloat64Dataset creates a Dataset from float64 columns using the engine
// from the source dataset. Never imports a specific engine.
func newFloat64Dataset(ds dataset.Dataset, cols map[string][]float64) (dataset.Dataset, error) {
	eng := dataset.GetEngine(ds.Table())
	if eng == nil {
		return dataset.Dataset{}, fmt.Errorf("stat: source dataset has no engine")
	}
	factory, ok := eng.(dataset.ColumnFactory)
	if !ok {
		return dataset.Dataset{}, fmt.Errorf("stat: engine %q does not support ColumnFactory", eng.Name())
	}
	var anyCols []dataset.AnyColumn
	// Deterministic column order: sort keys
	keys := make([]string, 0, len(cols))
	for k := range cols {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, name := range keys {
		anyCols = append(anyCols, factory.NewFloat64Column(name, cols[name]))
	}
	return dataset.NewDataset(eng, anyCols...)
}

// --- Boxplot ---

type boxplotStat struct{}

func (boxplotStat) Name() Name            { return Boxplot }
func (boxplotStat) RequiredAes() []string { return []string{"y"} }
func (boxplotStat) OutputSchema() []string {
	return []string{"x", "lower", "q1", "middle", "q3", "upper"}
}

// Compute produces the five-number summary for each unique X value (group).
// Output columns: x, lower, q1, middle, q3, upper.
func (boxplotStat) Compute(_ context.Context, ds dataset.Dataset, mapping map[string]string, _ Options) (dataset.Dataset, error) {
	yCol := mapping["y"]
	if yCol == "" {
		return dataset.Dataset{}, fmt.Errorf("stat_boxplot: missing 'y' aesthetic")
	}

	// Collect Y values, optionally grouped by X.
	yVals, err := collectFloat64Column(ds, yCol)
	if err != nil {
		return dataset.Dataset{}, err
	}

	// Collect X values for grouping (if they exist).
	xCol := mapping["x"]
	var xVals []float64
	if xCol != "" {
		xVals, _ = collectFloat64Column(ds, xCol)
	}

	// Group Y values by X.
	type group struct {
		x    float64
		yAll []float64
	}

	var groups []group
	if len(xVals) == len(yVals) {
		groupMap := make(map[float64]*group)
		var order []float64
		for i, xv := range xVals {
			g, exists := groupMap[xv]
			if !exists {
				g = &group{x: xv}
				groupMap[xv] = g
				order = append(order, xv)
			}
			g.yAll = append(g.yAll, yVals[i])
		}
		sort.Float64s(order)
		for _, xv := range order {
			groups = append(groups, *groupMap[xv])
		}
	} else {
		// Single group at x=0.
		groups = []group{{x: 0, yAll: yVals}}
	}

	// Compute five-number summary for each group.
	xs := make([]float64, len(groups))
	lower := make([]float64, len(groups))
	q1 := make([]float64, len(groups))
	median := make([]float64, len(groups))
	q3 := make([]float64, len(groups))
	upper := make([]float64, len(groups))

	for i, g := range groups {
		sort.Float64s(g.yAll)
		n := len(g.yAll)
		xs[i] = g.x
		median[i] = quantile(g.yAll, 0.5)
		q1[i] = quantile(g.yAll, 0.25)
		q3[i] = quantile(g.yAll, 0.75)

		// Whiskers: extend to most extreme data point within 1.5*IQR.
		iqr := q3[i] - q1[i]
		lowerFence := q1[i] - 1.5*iqr
		upperFence := q3[i] + 1.5*iqr

		lower[i] = q1[i] // start at Q1, search downward
		for j := 0; j < n; j++ {
			if g.yAll[j] >= lowerFence {
				lower[i] = g.yAll[j]
				break
			}
		}
		upper[i] = q3[i] // start at Q3, search upward
		for j := n - 1; j >= 0; j-- {
			if g.yAll[j] <= upperFence {
				upper[i] = g.yAll[j]
				break
			}
		}
	}

	return newFloat64Dataset(ds, map[string][]float64{
		"x":      xs,
		"lower":  lower,
		"q1":     q1,
		"middle": median,
		"q3":     q3,
		"upper":  upper,
	})
}

// quantile returns the p-th quantile of sorted data using linear interpolation.
func quantile(sorted []float64, p float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n == 1 {
		return sorted[0]
	}
	idx := p * float64(n-1)
	lo := int(math.Floor(idx))
	hi := int(math.Ceil(idx))
	if lo == hi || hi >= n {
		return sorted[lo]
	}
	frac := idx - float64(lo)
	return sorted[lo]*(1-frac) + sorted[hi]*frac
}
