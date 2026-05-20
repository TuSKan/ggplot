package bigquery

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"

	"github.com/TuSKan/ggplot/dataset"
)

// --- StatKernel: SQL-native statistical compute kernels ---
//
// All stat kernels return lazy bqDataset objects with pendingSQL.
// Computation runs server-side in BigQuery. Data only reaches local memory
// when the drawing pipeline calls Column().Values() — never before.

// requireBQCol extracts the column name and source reference from a column.
func requireBQCol(col dataset.AnyColumn) (colName, sourceRef string, err error) {
	bqCol, ok := col.(*bqColumn)
	if !ok {
		return "", "", fmt.Errorf("bigquery: expected bqColumn, got %T: %w", col, ErrUnsupportedType)
	}

	return bqCol.name, bqCol.ds.sourceRef(), nil
}

// lazyStatDataset returns a lazy bqDataset backed by a stat SQL query.
// No download, no materialization — stays in BigQuery until draw.
func (e *Engine) lazyStatDataset(sql string, schema *dataset.Schema) dataset.Table {
	return &bqDataset{
		engine:     e,
		schema:     schema,
		pendingSQL: sql,
	}
}

// --- Histogram ---

// Histogram bins a numeric column into equal-width bins.
// Returns a lazy bqDataset — computation stays in BigQuery.
func (e *Engine) Histogram(col dataset.AnyColumn, nBins int) (dataset.Table, error) {
	colName, src, err := requireBQCol(col)
	if err != nil {
		return nil, fmt.Errorf("bigquery: Histogram: %w", err)
	}

	if nBins <= 0 {
		nBins = 30 //nolint:mnd // Default bin count — Sturges' rule approximation for typical datasets.
	}

	sql := fmt.Sprintf(`
WITH stats AS (
  SELECT MIN(`+"`%[1]s`"+`) AS lo, MAX(`+"`%[1]s`"+`) AS hi
  FROM %[2]s
  WHERE `+"`%[1]s`"+` IS NOT NULL
),
binned AS (
  SELECT
    LEAST(CAST(FLOOR((`+"`%[1]s`"+` - lo) / NULLIF((hi - lo) / %[3]d, 0)) AS INT64), %[3]d - 1) AS bin_idx,
    COUNT(*) AS cnt
  FROM %[2]s CROSS JOIN stats
  WHERE `+"`%[1]s`"+` IS NOT NULL
  GROUP BY bin_idx
)
SELECT
  lo + (CAST(bin_idx AS FLOAT64) + 0.5) * ((hi - lo) / %[3]d) AS x,
  CAST(cnt AS FLOAT64) AS count
FROM binned CROSS JOIN stats
ORDER BY x
`, colName, src, nBins)

	schema := dataset.NewSchema(
		dataset.Field{Name: "count", Dtype: dataset.DTypeFloat64},
		dataset.Field{Name: "x", Dtype: dataset.DTypeFloat64},
	)

	return e.lazyStatDataset(sql, schema), nil
}

// --- KDE ---

// KDE computes kernel density estimation over a numeric column.
// Returns a lazy bqDataset — grid generation and Gaussian kernel
// evaluation all run server-side via CROSS JOIN.
func (e *Engine) KDE(_ context.Context, col dataset.AnyColumn, bandwidth float64, points int) (dataset.Table, error) {
	colName, src, err := requireBQCol(col)
	if err != nil {
		return nil, fmt.Errorf("bigquery: KDE: %w", err)
	}

	if points <= 0 {
		points = 512 //nolint:mnd // Default KDE grid size, standard in density estimation.
	}

	// Bandwidth: if <= 0, use Silverman's rule computed server-side.
	var bwExpr string
	if bandwidth > 0 {
		bwExpr = fmt.Sprintf("%v", bandwidth)
	} else {
		// Silverman: 1.06 * σ * n^(-0.2)
		bwExpr = "GREATEST(1.06 * sd * POW(n, -0.2), 1e-10)" //nolint:mnd // Silverman's rule-of-thumb constants.
	}

	sql := fmt.Sprintf(`
WITH stats AS (
  SELECT
    MIN(`+"`%[1]s`"+`) AS data_min,
    MAX(`+"`%[1]s`"+`) AS data_max,
    STDDEV_POP(`+"`%[1]s`"+`) AS sd,
    COUNT(*) AS n
  FROM %[2]s
  WHERE `+"`%[1]s`"+` IS NOT NULL
),
params AS (
  SELECT
    data_min - 3 * %[4]s AS grid_min,
    data_max + 3 * %[4]s AS grid_max,
    %[4]s AS bw,
    n
  FROM stats
),
grid AS (
  SELECT val AS x
  FROM params,
  UNNEST(GENERATE_ARRAY(grid_min, grid_max, (grid_max - grid_min) / (%[3]d - 1))) AS val
)
SELECT
  g.x,
  SUM(EXP(-0.5 * POW((g.x - t.`+"`%[1]s`"+`) / p.bw, 2)))
    / (p.bw * SQRT(2 * ACOS(-1)) * p.n) AS density
FROM grid g
CROSS JOIN %[2]s t
CROSS JOIN params p
WHERE t.`+"`%[1]s`"+` IS NOT NULL
GROUP BY g.x, p.bw, p.n
ORDER BY g.x
`, colName, src, points, bwExpr)

	schema := dataset.NewSchema(
		dataset.Field{Name: "density", Dtype: dataset.DTypeFloat64},
		dataset.Field{Name: "x", Dtype: dataset.DTypeFloat64},
	)

	return e.lazyStatDataset(sql, schema), nil
}

// --- LinearFit ---

// LinearFit computes OLS linear regression y = a + b*x.
// Returns a lazy bqDataset — coefficients and grid all computed server-side.
func (e *Engine) LinearFit(xCol, yCol dataset.AnyColumn, nOut int) (dataset.Table, error) {
	xName, xSrc, err := requireBQCol(xCol)
	if err != nil {
		return nil, fmt.Errorf("bigquery: LinearFit: x: %w", err)
	}

	yName, _, err := requireBQCol(yCol)
	if err != nil {
		return nil, fmt.Errorf("bigquery: LinearFit: y: %w", err)
	}

	if nOut <= 0 {
		nOut = 80 //nolint:mnd // Default output grid size for smooth methods.
	}

	sql := fmt.Sprintf(`
WITH stats AS (
  SELECT
    CAST(COUNT(*) AS FLOAT64) AS n,
    SUM(`+"`%[1]s`"+`) AS sx,
    SUM(`+"`%[2]s`"+`) AS sy,
    SUM(`+"`%[1]s`"+` * `+"`%[1]s`"+`) AS sxx,
    SUM(`+"`%[1]s`"+` * `+"`%[2]s`"+`) AS sxy,
    MIN(`+"`%[1]s`"+`) AS x_min,
    MAX(`+"`%[1]s`"+`) AS x_max
  FROM %[3]s
  WHERE `+"`%[1]s`"+` IS NOT NULL AND `+"`%[2]s`"+` IS NOT NULL
),
coeffs AS (
  SELECT
    CASE
      WHEN ABS(n * sxx - sx * sx) < 1e-15 THEN sy / n
      ELSE (sxx * sy - sx * sxy) / (n * sxx - sx * sx)
    END AS a,
    CASE
      WHEN ABS(n * sxx - sx * sx) < 1e-15 THEN 0
      ELSE (n * sxy - sx * sy) / (n * sxx - sx * sx)
    END AS b,
    x_min,
    x_max,
    LEAST(CAST(n AS INT64), %[4]d) AS nout
  FROM stats
)
SELECT val AS x, a + b * val AS y
FROM coeffs,
UNNEST(GENERATE_ARRAY(x_min, x_max,
  (x_max - x_min) / GREATEST(nout - 1, 1))) AS val
ORDER BY x
`, xName, yName, xSrc, nOut)

	schema := dataset.NewSchema(
		dataset.Field{Name: "x", Dtype: dataset.DTypeFloat64},
		dataset.Field{Name: "y", Dtype: dataset.DTypeFloat64},
	)

	return e.lazyStatDataset(sql, schema), nil
}

// --- LoessFit ---

// LoessFit computes locally weighted regression (LOESS).
// LOESS requires iterative nearest-neighbor lookups that cannot be expressed
// in SQL. Strategy: return a lazy Table that, when drawn, samples points
// server-side and computes LOESS on the small sample.
func (e *Engine) LoessFit(_ context.Context, xCol, yCol dataset.AnyColumn, nOut int) (dataset.Table, error) {
	xName, xSrc, err := requireBQCol(xCol)
	if err != nil {
		return nil, fmt.Errorf("bigquery: LoessFit: x: %w", err)
	}

	yName, _, err := requireBQCol(yCol)
	if err != nil {
		return nil, fmt.Errorf("bigquery: LoessFit: y: %w", err)
	}

	if nOut <= 0 {
		nOut = 80 //nolint:mnd // Default output grid size for smooth methods.
	}

	return &loessLazyTable{
		engine: e,
		xName:  xName,
		yName:  yName,
		src:    xSrc,
		nOut:   nOut,
	}, nil
}

// loessLazyTable is a lazy Table that defers LOESS computation until
// Column() is called (i.e., draw time). On first access it:
//  1. Samples 10 000 points via server-side ORDER BY RAND() LIMIT
//  2. Downloads the tiny sample
//  3. Computes LOESS locally on the sample
//  4. Caches the result
type loessLazyTable struct {
	engine *Engine
	xName  string
	yName  string
	src    string
	nOut   int

	once   sync.Once
	result dataset.Table
	err    error
}

func (t *loessLazyTable) Schema() *dataset.Schema {
	return dataset.NewSchema(
		dataset.Field{Name: "x", Dtype: dataset.DTypeFloat64},
		dataset.Field{Name: "y", Dtype: dataset.DTypeFloat64},
	)
}

func (t *loessLazyTable) NumRows() int64 { return int64(t.nOut) }
func (t *loessLazyTable) NumCols() int64 { return 2 } //nolint:mnd // LOESS output has exactly 2 columns: x and y.

func (t *loessLazyTable) Column(name string) (dataset.AnyColumn, error) {
	t.once.Do(t.compute)

	if t.err != nil {
		return nil, t.err
	}

	col, err := t.result.Column(name)
	if err != nil {
		return nil, fmt.Errorf("bigquery: loessLazyTable.Column: %w", err)
	}

	return col, nil
}

func (t *loessLazyTable) compute() {
	// 1. Sample server-side — only 10 000 rows downloaded.
	sampleSize := 10_000 //nolint:mnd // Sample size for LOESS — balances accuracy vs. download cost.

	sampleSQL := fmt.Sprintf(`
SELECT `+"`%[1]s`"+` AS x, `+"`%[2]s`"+` AS y
FROM %[3]s
WHERE `+"`%[1]s`"+` IS NOT NULL AND `+"`%[2]s`"+` IS NOT NULL
ORDER BY RAND()
LIMIT %[4]d
`, t.xName, t.yName, t.src, sampleSize)

	sampleSchema := dataset.NewSchema(
		dataset.Field{Name: "x", Dtype: dataset.DTypeFloat64},
		dataset.Field{Name: "y", Dtype: dataset.DTypeFloat64},
	)

	sampleDS := &bqDataset{
		engine:     t.engine,
		schema:     sampleSchema,
		pendingSQL: sampleSQL,
	}

	tbl, err := sampleDS.download()
	if err != nil {
		t.err = fmt.Errorf("bigquery: LoessFit: sample: %w", err)
		return
	}

	// 2. Extract the downloaded sample.
	xColLocal, err := tbl.Column("x")
	if err != nil {
		t.err = fmt.Errorf("bigquery: LoessFit: %w", err)
		return
	}

	yColLocal, err := tbl.Column("y")
	if err != nil {
		t.err = fmt.Errorf("bigquery: LoessFit: %w", err)
		return
	}

	xTyped, ok := xColLocal.(dataset.Column[float64])
	if !ok {
		t.err = fmt.Errorf("bigquery: LoessFit: x: expected float64, got %T: %w", xColLocal, ErrUnsupportedType)
		return
	}

	yTyped, ok := yColLocal.(dataset.Column[float64])
	if !ok {
		t.err = fmt.Errorf("bigquery: LoessFit: y: expected float64, got %T: %w", yColLocal, ErrUnsupportedType)
		return
	}

	// 3. Compute LOESS on the small sample.
	t.result, t.err = loessCompute(t.engine, xTyped.Values(), yTyped.Values(), t.nOut)
}

// loessCompute runs LOESS on local sample data.
func loessCompute(e *Engine, xData, yData []float64, nOut int) (dataset.Table, error) {
	n := len(xData)
	if n < 2 { //nolint:mnd // LOESS requires at least 2 data points.
		return buildStatTableLocal(e, map[string][]float64{"x": xData, "y": yData})
	}

	type xyPair struct{ x, y float64 }

	pts := make([]xyPair, n)
	for i := range xData {
		pts[i] = xyPair{xData[i], yData[i]}
	}

	sort.Slice(pts, func(i, j int) bool { return pts[i].x < pts[j].x })

	xMin, xMax := pts[0].x, pts[n-1].x

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
		xEval := xMin + float64(i)*step
		xs[i] = xEval

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

	return buildStatTableLocal(e, map[string][]float64{"x": xs, "y": ys})
}

// buildStatTableLocal creates a local Table from named float64 columns.
// Used only for LoessFit result (computed on a tiny sample).
func buildStatTableLocal(e *Engine, cols map[string][]float64) (dataset.Table, error) {
	local := e.localEngine()
	keys := make([]string, 0, len(cols))

	for k := range cols {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	anyCols := make([]dataset.AnyColumn, len(keys))
	fields := make([]dataset.Field, len(keys))

	for i, name := range keys {
		anyCols[i] = local.NewFloat64Column(name, cols[name])
		fields[i] = dataset.Field{Name: name, Dtype: dataset.DTypeFloat64}
	}

	result, err := local.FromColumns(dataset.NewSchema(fields...), anyCols...)
	if err != nil {
		return nil, fmt.Errorf("bigquery: buildStatTableLocal: %w", err)
	}

	return result, nil
}

// --- LinearFitSE ---

// LinearFitSE computes OLS regression with 95% confidence bands.
// Coefficients and residual SE all computed server-side in BigQuery.
func (e *Engine) LinearFitSE(xCol, yCol dataset.AnyColumn, nOut int) (dataset.Table, error) {
	xName, xSrc, err := requireBQCol(xCol)
	if err != nil {
		return nil, fmt.Errorf("bigquery: LinearFitSE: x: %w", err)
	}

	yName, _, err := requireBQCol(yCol)
	if err != nil {
		return nil, fmt.Errorf("bigquery: LinearFitSE: y: %w", err)
	}

	if nOut <= 0 {
		nOut = 80 //nolint:mnd // Default output grid size.
	}

	sql := fmt.Sprintf(`
WITH stats AS (
  SELECT
    CAST(COUNT(*) AS FLOAT64) AS n,
    SUM(`+"`%[1]s`"+`) AS sx,
    SUM(`+"`%[2]s`"+`) AS sy,
    SUM(`+"`%[1]s`"+` * `+"`%[1]s`"+`) AS sxx,
    SUM(`+"`%[1]s`"+` * `+"`%[2]s`"+`) AS sxy,
    MIN(`+"`%[1]s`"+`) AS x_min,
    MAX(`+"`%[1]s`"+`) AS x_max
  FROM %[3]s
  WHERE `+"`%[1]s`"+` IS NOT NULL AND `+"`%[2]s`"+` IS NOT NULL
),
coeffs AS (
  SELECT
    n, sx, sxx, x_min, x_max,
    CASE
      WHEN ABS(n * sxx - sx * sx) < 1e-15 THEN sy / n
      ELSE (sxx * sy - sx * sxy) / (n * sxx - sx * sx)
    END AS a,
    CASE
      WHEN ABS(n * sxx - sx * sx) < 1e-15 THEN 0
      ELSE (n * sxy - sx * sy) / (n * sxx - sx * sx)
    END AS b,
    LEAST(CAST(n AS INT64), %[4]d) AS nout
  FROM stats
),
residuals AS (
  SELECT
    SUM(POW(`+"`%[2]s`"+` - (c.a + c.b * `+"`%[1]s`"+`), 2)) / GREATEST(c.n - 2, 1) AS mse,
    c.*
  FROM %[3]s t CROSS JOIN coeffs c
  WHERE t.`+"`%[1]s`"+` IS NOT NULL AND t.`+"`%[2]s`"+` IS NOT NULL
  GROUP BY c.n, c.sx, c.sxx, c.x_min, c.x_max, c.a, c.b, c.nout
)
SELECT
  val AS x,
  r.a + r.b * val AS y,
  (r.a + r.b * val) - 1.96 * SQRT(r.mse * (1/r.n + POW(val - r.sx/r.n, 2) / (r.sxx - r.sx*r.sx/r.n))) AS ymin,
  (r.a + r.b * val) + 1.96 * SQRT(r.mse * (1/r.n + POW(val - r.sx/r.n, 2) / (r.sxx - r.sx*r.sx/r.n))) AS ymax
FROM residuals r,
UNNEST(GENERATE_ARRAY(r.x_min, r.x_max,
  (r.x_max - r.x_min) / GREATEST(r.nout - 1, 1))) AS val
ORDER BY x
`, xName, yName, xSrc, nOut)

	schema := dataset.NewSchema(
		dataset.Field{Name: "x", Dtype: dataset.DTypeFloat64},
		dataset.Field{Name: "y", Dtype: dataset.DTypeFloat64},
		dataset.Field{Name: "ymax", Dtype: dataset.DTypeFloat64},
		dataset.Field{Name: "ymin", Dtype: dataset.DTypeFloat64},
	)

	return e.lazyStatDataset(sql, schema), nil
}

// --- LoessFitSE ---

// LoessFitSE computes LOESS with approximate 95% confidence bands.
// Like LoessFit, it samples server-side and computes locally.
func (e *Engine) LoessFitSE(_ context.Context, xCol, yCol dataset.AnyColumn, nOut int) (dataset.Table, error) {
	xName, xSrc, err := requireBQCol(xCol)
	if err != nil {
		return nil, fmt.Errorf("bigquery: LoessFitSE: x: %w", err)
	}

	yName, _, err := requireBQCol(yCol)
	if err != nil {
		return nil, fmt.Errorf("bigquery: LoessFitSE: y: %w", err)
	}

	if nOut <= 0 {
		nOut = 80 //nolint:mnd // Default output grid size.
	}

	return &loessSELazyTable{
		engine: e,
		xName:  xName,
		yName:  yName,
		src:    xSrc,
		nOut:   nOut,
	}, nil
}

// loessSELazyTable is a lazy Table for LoessFitSE — same pattern as loessLazyTable
// but produces ymin/ymax bands using the local engine's LoessFitSE.
type loessSELazyTable struct {
	engine *Engine
	xName  string
	yName  string
	src    string
	nOut   int

	once   sync.Once
	result dataset.Table
	err    error
}

func (t *loessSELazyTable) Schema() *dataset.Schema {
	return dataset.NewSchema(
		dataset.Field{Name: "x", Dtype: dataset.DTypeFloat64},
		dataset.Field{Name: "y", Dtype: dataset.DTypeFloat64},
		dataset.Field{Name: "ymax", Dtype: dataset.DTypeFloat64},
		dataset.Field{Name: "ymin", Dtype: dataset.DTypeFloat64},
	)
}

func (t *loessSELazyTable) NumRows() int64 { return int64(t.nOut) }
func (t *loessSELazyTable) NumCols() int64 { return 4 } //nolint:mnd // x, y, ymin, ymax.

func (t *loessSELazyTable) Column(name string) (dataset.AnyColumn, error) {
	t.once.Do(t.compute)

	if t.err != nil {
		return nil, t.err
	}

	col, err := t.result.Column(name)
	if err != nil {
		return nil, fmt.Errorf("bigquery: loessSELazyTable.Column: %w", err)
	}

	return col, nil
}

func (t *loessSELazyTable) compute() {
	sampleSize := 10_000 //nolint:mnd // Sample size for LOESS.

	sampleSQL := fmt.Sprintf(`
SELECT `+"`%[1]s`"+` AS x, `+"`%[2]s`"+` AS y
FROM %[3]s
WHERE `+"`%[1]s`"+` IS NOT NULL AND `+"`%[2]s`"+` IS NOT NULL
ORDER BY RAND()
LIMIT %[4]d
`, t.xName, t.yName, t.src, sampleSize)

	sampleSchema := dataset.NewSchema(
		dataset.Field{Name: "x", Dtype: dataset.DTypeFloat64},
		dataset.Field{Name: "y", Dtype: dataset.DTypeFloat64},
	)

	sampleDS := &bqDataset{
		engine:     t.engine,
		schema:     sampleSchema,
		pendingSQL: sampleSQL,
	}

	tbl, err := sampleDS.download()
	if err != nil {
		t.err = fmt.Errorf("bigquery: LoessFitSE: sample: %w", err)
		return
	}

	xColLocal, err := tbl.Column("x")
	if err != nil {
		t.err = fmt.Errorf("bigquery: LoessFitSE: %w", err)
		return
	}

	yColLocal, err := tbl.Column("y")
	if err != nil {
		t.err = fmt.Errorf("bigquery: LoessFitSE: %w", err)
		return
	}

	xTyped, ok := xColLocal.(dataset.Column[float64])
	if !ok {
		t.err = fmt.Errorf("bigquery: LoessFitSE: x: expected float64, got %T: %w", xColLocal, ErrUnsupportedType)
		return
	}

	yTyped, ok := yColLocal.(dataset.Column[float64])
	if !ok {
		t.err = fmt.Errorf("bigquery: LoessFitSE: y: expected float64, got %T: %w", yColLocal, ErrUnsupportedType)
		return
	}

	t.result, t.err = loessComputeSE(t.engine, xTyped.Values(), yTyped.Values(), t.nOut)
}

// loessComputeSE runs LOESS with SE bands on local sample data.
func loessComputeSE(e *Engine, xData, yData []float64, nOut int) (dataset.Table, error) {
	n := len(xData)
	if n < 3 { //nolint:mnd // SE requires at least 3 points.
		return buildStatTableLocal(e, map[string][]float64{
			"x": xData, "y": yData,
			"ymin": yData, "ymax": yData,
		})
	}

	type xyPair struct{ x, y float64 }

	pts := make([]xyPair, n)
	for i := range xData {
		pts[i] = xyPair{xData[i], yData[i]}
	}

	sort.Slice(pts, func(i, j int) bool { return pts[i].x < pts[j].x })

	xMin, xMax := pts[0].x, pts[n-1].x

	if nOut > n {
		nOut = n
	}

	alpha := 0.3 //nolint:mnd // LOESS bandwidth fraction.
	if n < 20 {  //nolint:mnd // Adaptive bandwidth for small datasets.
		alpha = 0.75 //nolint:mnd // 75% for very small datasets.
	} else if n < 50 { //nolint:mnd // Medium dataset size threshold.
		alpha = 0.5 //nolint:mnd // 50% for medium datasets.
	}

	step := (xMax - xMin) / float64(nOut-1)

	xs := make([]float64, nOut)
	ys := make([]float64, nOut)
	ymin := make([]float64, nOut)
	ymax := make([]float64, nOut)

	k := min(max(int(math.Ceil(alpha*float64(n))), 3), n) //nolint:mnd // Minimum window of 3.
	tCrit := 1.96                                          //nolint:mnd // z ≈ 1.96 for 95% CI.

	lo, hi := 0, k

	for i := range nOut {
		xEval := xMin + float64(i)*step
		xs[i] = xEval

		for hi < n && math.Abs(pts[hi].x-xEval) < math.Abs(pts[lo].x-xEval) {
			lo++
			hi++
		}

		maxDist := math.Max(math.Abs(pts[lo].x-xEval), math.Abs(pts[hi-1].x-xEval))
		if maxDist < 1e-12 { //nolint:mnd // Minimum distance threshold.
			maxDist = 1e-12 //nolint:mnd // Minimum distance threshold.
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

		if sw < 1e-15 { //nolint:mnd // Near-zero weight sum.
			ys[i] = 0
			ymin[i] = 0
			ymax[i] = 0

			continue
		}

		det := sw*swxx - swx*swx
		if math.Abs(det) < 1e-15 { //nolint:mnd // Singular matrix.
			ys[i] = swy / sw
		} else {
			a := (swxx*swy - swx*swxy) / det
			ys[i] = a
		}

		// Approximate SE from weighted residuals in the local window.
		var wsse, wn float64

		for j := lo; j < hi; j++ {
			u := math.Abs(pts[j].x-xEval) / maxDist
			if u >= 1.0 {
				continue
			}

			w := (1 - u*u*u)
			w = w * w * w

			r := pts[j].y - ys[i]
			wsse += w * r * r
			wn += w
		}

		if wn > 0 {
			se := math.Sqrt(wsse / wn / sw)
			ymin[i] = ys[i] - tCrit*se
			ymax[i] = ys[i] + tCrit*se
		} else {
			ymin[i] = ys[i]
			ymax[i] = ys[i]
		}
	}

	return buildStatTableLocal(e, map[string][]float64{
		"x": xs, "y": ys, "ymin": ymin, "ymax": ymax,
	})
}

// --- Boxplot ---

// Boxplot computes the five-number summary for a numeric column,
// optionally grouped by a categorical column.
// Returns a lazy bqDataset — APPROX_QUANTILES, whisker, and notch
// computation all run server-side in a single SQL query.
func (e *Engine) Boxplot(yCol, groupCol dataset.AnyColumn, whisker string, notch bool) (dataset.Table, error) {
	yName, ySrc, err := requireBQCol(yCol)
	if err != nil {
		return nil, fmt.Errorf("bigquery: Boxplot: y: %w", err)
	}

	if whisker == "" {
		whisker = "tukey"
	}

	// Build the GROUP BY and x-value expressions.
	var groupExpr, groupByClause, orderByClause string

	if groupCol != nil {
		gName, _, gErr := requireBQCol(groupCol)
		if gErr != nil {
			return nil, fmt.Errorf("bigquery: Boxplot: group: %w", gErr)
		}

		groupExpr = fmt.Sprintf("CAST(`%s` AS FLOAT64)", gName)
		groupByClause = fmt.Sprintf("GROUP BY `%s`", gName)
		orderByClause = fmt.Sprintf("ORDER BY `%s`", gName)
	} else {
		groupExpr = "0"
		groupByClause = ""
		orderByClause = ""
	}

	// Whisker SQL expression.
	var lowerExpr, upperExpr string

	switch whisker {
	case "range":
		lowerExpr = "data_min"
		upperExpr = "data_max"
	default: // "tukey"
		lowerExpr = "GREATEST(q1 - 1.5 * (q3 - q1), data_min)" //nolint:mnd // Tukey's 1.5×IQR whisker rule.
		upperExpr = "LEAST(q3 + 1.5 * (q3 - q1), data_max)"    //nolint:mnd // Tukey's 1.5×IQR whisker rule.
	}

	// Notch SQL expression.
	var notchLoExpr, notchHiExpr string
	if notch {
		// 1.58 × IQR / √n gives 95% CI for median comparison.
		notchLoExpr = "middle - 1.58 * (q3 - q1) / SQRT(gn)" //nolint:mnd // 95% CI for median comparison.
		notchHiExpr = "middle + 1.58 * (q3 - q1) / SQRT(gn)" //nolint:mnd // 95% CI for median comparison.
	} else {
		notchLoExpr = "middle"
		notchHiExpr = "middle"
	}

	sql := fmt.Sprintf(`
WITH summary AS (
  SELECT
    %[3]s AS x,
    APPROX_QUANTILES(`+"`%[1]s`"+`, 100)[OFFSET(25)] AS q1,
    APPROX_QUANTILES(`+"`%[1]s`"+`, 100)[OFFSET(50)] AS middle,
    APPROX_QUANTILES(`+"`%[1]s`"+`, 100)[OFFSET(75)] AS q3,
    MIN(`+"`%[1]s`"+`) AS data_min,
    MAX(`+"`%[1]s`"+`) AS data_max,
    CAST(COUNT(*) AS FLOAT64) AS gn
  FROM %[2]s
  WHERE `+"`%[1]s`"+` IS NOT NULL
  %[4]s
)
SELECT
  x,
  %[6]s AS lower,
  q1,
  middle,
  q3,
  %[7]s AS upper,
  %[8]s AS notch_lower,
  %[9]s AS notch_upper
FROM summary
%[5]s
`, yName, ySrc, groupExpr, groupByClause, orderByClause,
		lowerExpr, upperExpr, notchLoExpr, notchHiExpr)

	schema := dataset.NewSchema(
		dataset.Field{Name: "lower", Dtype: dataset.DTypeFloat64},
		dataset.Field{Name: "middle", Dtype: dataset.DTypeFloat64},
		dataset.Field{Name: "notch_lower", Dtype: dataset.DTypeFloat64},
		dataset.Field{Name: "notch_upper", Dtype: dataset.DTypeFloat64},
		dataset.Field{Name: "q1", Dtype: dataset.DTypeFloat64},
		dataset.Field{Name: "q3", Dtype: dataset.DTypeFloat64},
		dataset.Field{Name: "upper", Dtype: dataset.DTypeFloat64},
		dataset.Field{Name: "x", Dtype: dataset.DTypeFloat64},
	)

	return e.lazyStatDataset(sql, schema), nil
}
