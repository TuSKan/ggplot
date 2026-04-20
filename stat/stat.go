// Package stat provides statistical transformations for the Grammar of Graphics.
// Stats compute derived data (bins, counts, densities, smooths) from raw data,
// producing new datasets that are then rendered by geometries.
//
// Each stat implements the [Stat] interface and is registered with a string name
// for lookup during pipeline compilation.
package stat

import (
	"fmt"
	"math"
	"sort"

	"github.com/TuSKan/ggplot/dataset"
)

// Stat computes a statistical transformation on a dataset, producing a
// new (possibly differently shaped) dataset. The transformed dataset's
// columns depend on the stat type.
type Stat interface {
	// Name returns the stat's identifier (e.g., "identity", "bin", "count").
	Name() string

	// RequiredAes returns the aesthetic channels this stat needs.
	RequiredAes() []string

	// Compute performs the transformation.
	Compute(ds dataset.Dataset, mapping map[string]string) (dataset.Dataset, error)
}

// --- Registry ---

var registry = map[string]Stat{
	"identity": identityStat{},
}

// Register adds a stat to the global registry.
func Register(s Stat) { registry[s.Name()] = s }

// Lookup retrieves a stat by name. Returns the identity stat if not found.
func Lookup(name string) Stat {
	if s, ok := registry[name]; ok {
		return s
	}
	return identityStat{}
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

func (identityStat) Name() string          { return "identity" }
func (identityStat) RequiredAes() []string { return nil }
func (identityStat) Compute(ds dataset.Dataset, _ map[string]string) (dataset.Dataset, error) {
	return ds, nil
}

// --- Bin ---

type binStat struct{}

func (binStat) Name() string          { return "bin" }
func (binStat) RequiredAes() []string { return []string{"x"} }

func (binStat) Compute(ds dataset.Dataset, mapping map[string]string) (dataset.Dataset, error) {
	xCol := mapping["x"]
	if xCol == "" {
		return nil, fmt.Errorf("stat_bin: missing 'x' aesthetic")
	}

	col, err := ds.Column(xCol)
	if err != nil {
		return nil, err
	}

	vals, err := collectFloat64(col)
	if err != nil {
		return nil, err
	}

	// Determine number of bins: explicit __bins param > Sturges' rule.
	nBins := 0
	if binsStr, ok := mapping["__bins"]; ok {
		if _, err := fmt.Sscanf(binsStr, "%d", &nBins); err != nil {
			nBins = 0 // Fall through to Sturges' rule below.
		}
	}
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

	return dataset.NewDataFrame(map[string][]float64{
		"x":     centers,
		"count": counts,
	})
}

// --- Count ---

type countStat struct{}

func (countStat) Name() string          { return "count" }
func (countStat) RequiredAes() []string { return []string{"x"} }

func (countStat) Compute(ds dataset.Dataset, mapping map[string]string) (dataset.Dataset, error) {
	xCol := mapping["x"]
	if xCol == "" {
		return nil, fmt.Errorf("stat_count: missing 'x' aesthetic")
	}

	col, err := ds.Column(xCol)
	if err != nil {
		return nil, err
	}

	vals, err := collectFloat64(col)
	if err != nil {
		return nil, err
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

	return dataset.NewDataFrame(map[string][]float64{
		"x":     xs,
		"count": counts,
	})
}

// --- Density ---

type densityStat struct{}

func (densityStat) Name() string          { return "density" }
func (densityStat) RequiredAes() []string { return []string{"x"} }

func (densityStat) Compute(ds dataset.Dataset, mapping map[string]string) (dataset.Dataset, error) {
	xCol := mapping["x"]
	if xCol == "" {
		return nil, fmt.Errorf("stat_density: missing 'x' aesthetic")
	}

	col, err := ds.Column(xCol)
	if err != nil {
		return nil, err
	}

	vals, err := collectFloat64(col)
	if err != nil {
		return nil, err
	}
	sort.Float64s(vals)

	n := len(vals)
	if n == 0 {
		return dataset.NewDataFrame(map[string][]float64{"x": {}, "density": {}})
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
		x := xmin + float64(i)*step
		xs[i] = x
		density := 0.0
		for _, v := range vals {
			z := (x - v) / bandwidth
			density += math.Exp(-0.5*z*z) / (bandwidth * math.Sqrt(2*math.Pi))
		}
		ys[i] = density / float64(n)
	}

	return dataset.NewDataFrame(map[string][]float64{"x": xs, "density": ys})
}

// --- Smooth ---

type smoothStat struct{}

func (smoothStat) Name() string          { return "smooth" }
func (smoothStat) RequiredAes() []string { return []string{"x", "y"} }

func (smoothStat) Compute(ds dataset.Dataset, mapping map[string]string) (dataset.Dataset, error) {
	xCol := mapping["x"]
	yCol := mapping["y"]
	if xCol == "" || yCol == "" {
		return nil, fmt.Errorf("stat_smooth: missing 'x' or 'y' aesthetic")
	}

	xData, err := collectFloat64Column(ds, xCol)
	if err != nil {
		return nil, err
	}
	yData, err := collectFloat64Column(ds, yCol)
	if err != nil {
		return nil, err
	}
	if len(xData) != len(yData) {
		return nil, fmt.Errorf("stat_smooth: x and y columns have different lengths")
	}

	n := len(xData)
	if n < 2 {
		return dataset.NewDataFrame(map[string][]float64{"x": xData, "y": yData})
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
		alpha = 0.75 // use wider bandwidth for small datasets
	} else if n < 50 {
		alpha = 0.5
	}

	// Number of output points.
	nOut := 80
	if nOut > n {
		nOut = n
	}
	step := (xMax - xMin) / float64(nOut-1)

	xs := make([]float64, nOut)
	ys := make([]float64, nOut)

	k := int(math.Ceil(alpha * float64(n))) // number of neighbors
	if k < 3 {
		k = 3
	}
	if k > n {
		k = n
	}

	for i := 0; i < nOut; i++ {
		xEval := xMin + float64(i)*step
		xs[i] = xEval

		// Find k nearest neighbors by distance to xEval.
		dists := make([]float64, n)
		for j := range pts {
			dists[j] = math.Abs(pts[j].x - xEval)
		}
		// Find the k-th smallest distance (max bandwidth).
		sortedDists := make([]float64, n)
		copy(sortedDists, dists)
		sort.Float64s(sortedDists)
		maxDist := sortedDists[k-1]
		if maxDist < 1e-12 {
			maxDist = 1e-12
		}

		// Weighted local linear regression: y = a + b*(x - xEval)
		// using tri-cube kernel: w = (1 - (d/maxDist)^3)^3
		var sw, swx, swy, swxx, swxy float64
		for j := range pts {
			u := dists[j] / maxDist
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
		// Solve 2x2 WLS system: [sw swx; swx swxx] [a; b] = [swy; swxy]
		det := sw*swxx - swx*swx
		if math.Abs(det) < 1e-15 {
			ys[i] = swy / sw
		} else {
			a := (swxx*swy - swx*swxy) / det
			ys[i] = a // evaluate at xEval where dx=0, so y = a
		}
	}

	return dataset.NewDataFrame(map[string][]float64{"x": xs, "y": ys})
}

// --- Summary ---

type summaryStat struct{}

func (summaryStat) Name() string          { return "summary" }
func (summaryStat) RequiredAes() []string { return []string{"x", "y"} }

func (summaryStat) Compute(ds dataset.Dataset, mapping map[string]string) (dataset.Dataset, error) {
	// summaryStat computes mean(y) for each distinct x value.
	xCol := mapping["x"]
	yCol := mapping["y"]
	if xCol == "" || yCol == "" {
		return nil, fmt.Errorf("stat_summary: missing 'x' or 'y' aesthetic")
	}

	xData, err := collectFloat64Column(ds, xCol)
	if err != nil {
		return nil, err
	}
	yData, err := collectFloat64Column(ds, yCol)
	if err != nil {
		return nil, err
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

	return dataset.NewDataFrame(map[string][]float64{"x": xs, "y": means})
}

// --- Helpers ---

func collectFloat64(col dataset.Column) ([]float64, error) {
	iter, ok := col.(dataset.IterableColumn)
	if !ok {
		return nil, fmt.Errorf("stat: column does not support float64 iteration")
	}
	flt, err := iter.Float64s()
	if err != nil {
		return nil, err
	}
	var vals []float64
	for {
		v, isNull, ok := flt.Next()
		if !ok {
			break
		}
		if !isNull {
			vals = append(vals, v)
		}
	}
	return vals, nil
}

func collectFloat64Column(ds dataset.Dataset, name string) ([]float64, error) {
	col, err := ds.Column(name)
	if err != nil {
		return nil, err
	}
	return collectFloat64(col)
}

// --- Boxplot ---

type boxplotStat struct{}

func (boxplotStat) Name() string          { return "boxplot" }
func (boxplotStat) RequiredAes() []string { return []string{"y"} }

// Compute produces the five-number summary for each unique X value (group).
// Output columns: x, lower, q1, middle, q3, upper.
func (boxplotStat) Compute(ds dataset.Dataset, mapping map[string]string) (dataset.Dataset, error) {
	yCol := mapping["y"]
	if yCol == "" {
		return nil, fmt.Errorf("stat_boxplot: missing 'y' aesthetic")
	}

	// Collect Y values, optionally grouped by X.
	yVals, err := collectFloat64Column(ds, yCol)
	if err != nil {
		return nil, err
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

	return dataset.NewDataFrame(map[string][]float64{
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
