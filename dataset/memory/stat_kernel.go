package memory

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"sort"
	"sync"

	"github.com/TuSKan/ggplot/dataset"
)

// --- StatKernel: statistical compute kernels ---
//
// These methods move the core algorithms from stat/ into engine-native
// operations. Each produces a new Table (no lazy materialization).

// buildTable creates a Table from named float64 columns, sorted by key
// for deterministic column order.
func (e *Engine) buildTable(cols map[string][]float64) (dataset.Table, error) {
	keys := make([]string, 0, len(cols))
	for k := range cols {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	anyCols := make([]dataset.AnyColumn, len(keys))
	fields := make([]dataset.Field, len(keys))

	for i, name := range keys {
		anyCols[i] = e.NewFloat64Column(name, cols[name])
		fields[i] = dataset.Field{Name: name, Dtype: dataset.DTypeFloat64}
	}

	return e.FromColumns(dataset.NewSchema(fields...), anyCols...)
}

// --- Shared helpers (ported from stat/stat.go) ---

// statSilvermanBandwidth computes Silverman's rule-of-thumb bandwidth.
func statSilvermanBandwidth(vals []float64) float64 {
	n := len(vals)

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

	return 1.06 * sd * math.Pow(float64(n), -0.2) //nolint:mnd // Silverman's rule-of-thumb constants.
}

// statQuantile returns the p-th quantile of sorted data using linear interpolation.
func statQuantile(sorted []float64, p float64) float64 {
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

// statAutoBins selects the number of bins using Sturges' rule.
func statAutoBins(vals []float64) int {
	n := len(vals)
	if n <= 1 {
		return 1
	}

	sorted := make([]float64, n)
	copy(sorted, vals)
	sort.Float64s(sorted)

	vMin, vMax := sorted[0], sorted[n-1]

	span := vMax - vMin
	if span <= 0 {
		return 1
	}

	// Sturges' rule: k = ceil(1 + log2(n))
	return int(math.Ceil(1.0 + math.Log2(float64(n))))
}

// --- Histogram ---

// Histogram bins a numeric column into equal-width bins.
func (e *Engine) Histogram(col dataset.AnyColumn, nBins int) (dataset.Table, error) {
	c, err := requireFloat64(col)
	if err != nil {
		return nil, fmt.Errorf("Histogram: %w", err)
	}

	vals := c.data
	if len(vals) == 0 {
		return e.buildTable(map[string][]float64{"x": {}, "count": {}})
	}

	if nBins <= 0 {
		nBins = statAutoBins(vals)
	}

	if nBins <= 0 {
		nBins = 1
	}

	if nBins > len(vals) {
		nBins = len(vals)
	}

	lo, hi := vals[0], vals[0]
	for _, v := range vals {
		if v < lo {
			lo = v
		}

		if v > hi {
			hi = v
		}
	}

	binWidth := (hi - lo) / float64(nBins)
	if binWidth <= 0 {
		binWidth = 1
	}

	centers := make([]float64, nBins)
	counts := make([]float64, nBins)

	for i := range centers {
		centers[i] = lo + binWidth*(float64(i)+0.5) //nolint:mnd // Half-bin offset for center calculation.
	}

	for _, v := range vals {
		idx := int((v - lo) / binWidth)
		if idx >= nBins {
			idx = nBins - 1
		}

		if idx < 0 {
			idx = 0
		}

		counts[idx]++
	}

	return e.buildTable(map[string][]float64{"x": centers, "count": counts})
}

// --- KDE ---

// KDE computes kernel density estimation over a numeric column.
func (e *Engine) KDE(ctx context.Context, col dataset.AnyColumn, bandwidth float64, points int) (dataset.Table, error) {
	c, err := requireFloat64(col)
	if err != nil {
		return nil, fmt.Errorf("KDE: %w", err)
	}

	vals := make([]float64, len(c.data))
	copy(vals, c.data)
	sort.Float64s(vals)

	n := len(vals)
	if n == 0 {
		return e.buildTable(map[string][]float64{"x": {}, "density": {}})
	}

	if bandwidth <= 0 {
		bandwidth = statSilvermanBandwidth(vals)
	}

	if points <= 0 {
		points = 512 //nolint:mnd // Default KDE grid size, standard in density estimation.
	}

	xmin, xmax := vals[0]-3*bandwidth, vals[n-1]+3*bandwidth //nolint:mnd // 3σ padding is standard for Gaussian KDE.
	step := (xmax - xmin) / float64(points-1)

	xs := make([]float64, points)
	ys := make([]float64, points)

	for i := range xs {
		xs[i] = xmin + float64(i)*step
	}

	// Parallel KDE evaluation: chunk by CPU count.
	nCPU := min(runtime.NumCPU(), points)
	chunk := (points + nCPU - 1) / nCPU

	var wg sync.WaitGroup

	errCh := make(chan error, 1)

	for ci := range nCPU {
		start := ci * chunk
		end := min(start+chunk, points)

		wg.Add(1)

		go func(start, end int) {
			defer wg.Done()

			bwInv := 1.0 / bandwidth
			norm := 1.0 / (bandwidth * math.Sqrt(2*math.Pi) * float64(n))

			for i := start; i < end; i++ {
				if i%64 == 0 { //nolint:mnd // Check cancellation every 64 iterations to amortize ctx.Err() cost.
					if err := ctx.Err(); err != nil {
						select {
						case errCh <- err:
						default:
						}

						return
					}
				}

				x := xs[i]
				density := 0.0

				for _, v := range vals {
					z := (x - v) * bwInv
					density += math.Exp(-0.5 * z * z) //nolint:mnd // Gaussian kernel exponent coefficient.
				}

				ys[i] = density * norm
			}
		}(start, end)
	}

	wg.Wait()

	select {
	case err := <-errCh:
		return nil, err
	default:
	}

	return e.buildTable(map[string][]float64{"x": xs, "density": ys})
}

// --- LinearFit ---

// xyPair is a sorted (x, y) data point used by smooth methods.
type xyPair struct{ x, y float64 }

// LinearFit computes OLS linear regression y = a + b*x.
func (e *Engine) LinearFit(xCol, yCol dataset.AnyColumn, nOut int) (dataset.Table, error) {
	cx, err := requireFloat64(xCol)
	if err != nil {
		return nil, fmt.Errorf("LinearFit: x: %w", err)
	}

	cy, err := requireFloat64(yCol)
	if err != nil {
		return nil, fmt.Errorf("LinearFit: y: %w", err)
	}

	if len(cx.data) != len(cy.data) {
		return nil, fmt.Errorf("LinearFit: x and y columns have different lengths (%d vs %d): %w",
			len(cx.data), len(cy.data), ErrLengthMismatch)
	}

	n := len(cx.data)
	if n < 2 { //nolint:mnd // OLS requires at least 2 data points.
		return e.buildTable(map[string][]float64{"x": cx.data, "y": cy.data})
	}

	// Sort by X for consistent ordering.
	pts := make([]xyPair, n)
	for i := range cx.data {
		pts[i] = xyPair{cx.data[i], cy.data[i]}
	}

	sort.Slice(pts, func(i, j int) bool { return pts[i].x < pts[j].x })

	xMin, xMax := pts[0].x, pts[n-1].x

	if nOut <= 0 {
		nOut = 80 //nolint:mnd // Default output grid size for smooth methods.
	}

	if nOut > n {
		nOut = n
	}

	// OLS: y = a + b*x
	nf := float64(n)

	var sx, sy, sxx, sxy float64
	for _, p := range pts {
		sx += p.x
		sy += p.y
		sxx += p.x * p.x
		sxy += p.x * p.y
	}

	det := nf*sxx - sx*sx

	var a, b float64
	if math.Abs(det) < 1e-15 { //nolint:mnd // Near-zero determinant threshold for singular matrix detection.
		a = sy / nf
		b = 0
	} else {
		b = (nf*sxy - sx*sy) / det
		a = (sy - b*sx) / nf
	}

	step := (xMax - xMin) / float64(nOut-1)
	xs := make([]float64, nOut)
	ys := make([]float64, nOut)

	for i := range nOut {
		xs[i] = xMin + float64(i)*step
		ys[i] = a + b*xs[i]
	}

	return e.buildTable(map[string][]float64{"x": xs, "y": ys})
}

// --- LoessFit ---

// LoessFit computes locally weighted regression (LOESS).
func (e *Engine) LoessFit(ctx context.Context, xCol, yCol dataset.AnyColumn, nOut int) (dataset.Table, error) {
	cx, err := requireFloat64(xCol)
	if err != nil {
		return nil, fmt.Errorf("LoessFit: x: %w", err)
	}

	cy, err := requireFloat64(yCol)
	if err != nil {
		return nil, fmt.Errorf("LoessFit: y: %w", err)
	}

	if len(cx.data) != len(cy.data) {
		return nil, fmt.Errorf("LoessFit: x and y columns have different lengths (%d vs %d): %w",
			len(cx.data), len(cy.data), ErrLengthMismatch)
	}

	n := len(cx.data)
	if n < 2 { //nolint:mnd // LOESS requires at least 2 data points.
		return e.buildTable(map[string][]float64{"x": cx.data, "y": cy.data})
	}

	// Sort by X for consistent ordering.
	pts := make([]xyPair, n)
	for i := range cx.data {
		pts[i] = xyPair{cx.data[i], cy.data[i]}
	}

	sort.Slice(pts, func(i, j int) bool { return pts[i].x < pts[j].x })

	xMin, xMax := pts[0].x, pts[n-1].x

	if nOut <= 0 {
		nOut = 80 //nolint:mnd // Default output grid size for smooth methods.
	}

	if nOut > n {
		nOut = n
	}

	alpha := 0.3 //nolint:mnd // LOESS bandwidth fraction — 30% of data for large datasets.
	if n < 20 {  //nolint:mnd // Adaptive bandwidth for small datasets.
		alpha = 0.75 //nolint:mnd // 75% of data for very small datasets.
	} else if n < 50 { //nolint:mnd // Medium dataset size threshold.
		alpha = 0.5 //nolint:mnd // 50% of data for medium datasets.
	}

	step := (xMax - xMin) / float64(nOut-1)

	xs := make([]float64, nOut)
	ys := make([]float64, nOut)

	k := min(max(int(math.Ceil(alpha*float64(n))), 3), n) //nolint:mnd // Minimum window of 3 data points for stable local fit.

	lo, hi := 0, k

	for i := range nOut {
		if i%32 == 0 { //nolint:mnd // Check cancellation every 32 iterations to amortize ctx.Err() cost.
			if err := ctx.Err(); err != nil {
				return nil, fmt.Errorf("LoessFit: %w", err)
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
		if maxDist < 1e-12 { //nolint:mnd // Minimum distance threshold to avoid division by zero.
			maxDist = 1e-12 //nolint:mnd // Minimum distance threshold to avoid division by zero.
		}

		var sw, swx, swy, swxx, swxy float64

		for j := lo; j < hi; j++ {
			u := math.Abs(pts[j].x-xEval) / maxDist
			if u >= 1.0 {
				continue
			}

			w := (1 - u*u*u)
			w = w * w * w // tri-cube kernel

			dx := pts[j].x - xEval
			sw += w
			swx += w * dx
			swy += w * pts[j].y
			swxx += w * dx * dx
			swxy += w * dx * pts[j].y
		}

		if sw < 1e-15 { //nolint:mnd // Near-zero weight sum threshold for degenerate windows.
			ys[i] = 0

			continue
		}

		det := sw*swxx - swx*swx
		if math.Abs(det) < 1e-15 { //nolint:mnd // Near-zero determinant threshold for singular matrix detection.
			ys[i] = swy / sw
		} else {
			a := (swxx*swy - swx*swxy) / det
			ys[i] = a
		}
	}

	return e.buildTable(map[string][]float64{"x": xs, "y": ys})
}

// --- Boxplot ---

// Boxplot computes the five-number summary for a numeric column.
func (e *Engine) Boxplot(yCol, groupCol dataset.AnyColumn, whisker string, notch bool) (dataset.Table, error) {
	cy, err := requireFloat64(yCol)
	if err != nil {
		return nil, fmt.Errorf("Boxplot: y: %w", err)
	}

	yVals := cy.data

	var groups []boxplotGroup

	if groupCol != nil {
		groups, err = boxplotGroups(groupCol, yVals)
		if err != nil {
			return nil, err
		}
	} else {
		groups = []boxplotGroup{{x: 0, yAll: yVals}}
	}

	// Compute five-number summary for each group.
	xs := make([]float64, len(groups))
	lower := make([]float64, len(groups))
	q1Vals := make([]float64, len(groups))
	median := make([]float64, len(groups))
	q3Vals := make([]float64, len(groups))
	upper := make([]float64, len(groups))
	notchLo := make([]float64, len(groups))
	notchHi := make([]float64, len(groups))

	if whisker == "" {
		whisker = "tukey"
	}

	for i, g := range groups {
		sorted := make([]float64, len(g.yAll))
		copy(sorted, g.yAll)
		sort.Float64s(sorted)

		gn := len(sorted)
		xs[i] = g.x
		median[i] = statQuantile(sorted, 0.5)  //nolint:mnd // Median = 50th percentile.
		q1Vals[i] = statQuantile(sorted, 0.25) //nolint:mnd // Q1 = 25th percentile.
		q3Vals[i] = statQuantile(sorted, 0.75) //nolint:mnd // Q3 = 75th percentile.

		iqr := q3Vals[i] - q1Vals[i]

		switch whisker {
		case "range":
			lower[i] = sorted[0]
			upper[i] = sorted[gn-1]
		default: // "tukey"
			lowerFence := q1Vals[i] - 1.5*iqr //nolint:mnd // Tukey's 1.5×IQR whisker rule.
			upperFence := q3Vals[i] + 1.5*iqr //nolint:mnd // Tukey's 1.5×IQR whisker rule.

			lower[i] = q1Vals[i]

			for j := range gn {
				if sorted[j] >= lowerFence {
					lower[i] = sorted[j]

					break
				}
			}

			upper[i] = q3Vals[i]

			for j := gn - 1; j >= 0; j-- {
				if sorted[j] <= upperFence {
					upper[i] = sorted[j]

					break
				}
			}
		}

		// Notch: 95% CI = median ± 1.58 × IQR / √n
		if notch && gn > 0 {
			ci := 1.58 * iqr / math.Sqrt(float64(gn)) //nolint:mnd // 1.58 × IQR/√n gives 95% CI for median comparison.
			notchLo[i] = median[i] - ci
			notchHi[i] = median[i] + ci
		} else {
			notchLo[i] = median[i]
			notchHi[i] = median[i]
		}
	}

	return e.buildTable(map[string][]float64{
		"x":           xs,
		"lower":       lower,
		"q1":          q1Vals,
		"middle":      median,
		"q3":          q3Vals,
		"upper":       upper,
		"notch_lower": notchLo,
		"notch_upper": notchHi,
	})
}

// boxplotGroup holds a single boxplot group's data.
type boxplotGroup struct {
	x    float64
	yAll []float64
}

// boxplotGroups partitions y values by a group column.
// Supports both float64 and string group columns.
func boxplotGroups(groupCol dataset.AnyColumn, yVals []float64) ([]boxplotGroup, error) {
	switch gc := groupCol.(type) {
	case dataset.Column[float64]:
		return boxplotGroupsFloat64(gc.Values(), yVals)
	case dataset.Column[string]:
		return boxplotGroupsString(gc.Values(), yVals)
	default:
		return nil, fmt.Errorf("Boxplot: group column %T: unsupported type: %w", groupCol, ErrRequiresFloat64)
	}
}

func boxplotGroupsFloat64(gVals, yVals []float64) ([]boxplotGroup, error) {
	if len(gVals) != len(yVals) {
		return nil, fmt.Errorf("Boxplot: y and group columns have different lengths (%d vs %d): %w",
			len(yVals), len(gVals), ErrLengthMismatch)
	}

	groupMap := make(map[float64]*boxplotGroup)

	var order []float64

	for i, xv := range gVals {
		g, exists := groupMap[xv]
		if !exists {
			g = &boxplotGroup{x: xv}
			groupMap[xv] = g
			order = append(order, xv)
		}

		g.yAll = append(g.yAll, yVals[i])
	}

	sort.Float64s(order)

	groups := make([]boxplotGroup, len(order))
	for i, xv := range order {
		groups[i] = *groupMap[xv]
	}

	return groups, nil
}

func boxplotGroupsString(gVals []string, yVals []float64) ([]boxplotGroup, error) {
	if len(gVals) != len(yVals) {
		return nil, fmt.Errorf("Boxplot: y and group columns have different lengths (%d vs %d): %w",
			len(yVals), len(gVals), ErrLengthMismatch)
	}

	type strGroup struct {
		label string
		yAll  []float64
	}

	groupMap := make(map[string]*strGroup)

	var order []string

	for i, label := range gVals {
		g, exists := groupMap[label]
		if !exists {
			g = &strGroup{label: label}
			groupMap[label] = g
			order = append(order, label)
		}

		g.yAll = append(g.yAll, yVals[i])
	}

	sort.Strings(order)

	groups := make([]boxplotGroup, len(order))
	for i, label := range order {
		groups[i] = boxplotGroup{x: float64(i), yAll: groupMap[label].yAll}
	}

	return groups, nil
}
