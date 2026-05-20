package bigquery

import (
	"context"
	"fmt"
	"math"

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

// loessTopK computes the VECTOR_SEARCH top_k parameter for LOESS.
// Adaptive bandwidth: alpha varies by data size (0.75/0.50/0.30),
// k = CEIL(alpha × n), clamped to [3, n].
func loessTopK(n int) int {
	alpha := 0.3 //nolint:mnd // LOESS bandwidth fraction — 30% of data for large datasets.
	if n < 20 {  //nolint:mnd // Adaptive bandwidth for small datasets.
		alpha = 0.75 //nolint:mnd // 75% of data for very small datasets.
	} else if n < 50 { //nolint:mnd // Medium dataset size threshold.
		alpha = 0.5 //nolint:mnd // 50% of data for medium datasets.
	}

	k := int(math.Ceil(alpha * float64(n)))
	k = max(k, 3) //nolint:mnd // Minimum window of 3 data points for stable local fit.

	return min(k, n)
}

// LoessFit computes locally weighted regression (LOESS) entirely server-side.
// Uses VECTOR_SEARCH for k-NN neighbor lookup and SQL aggregation for
// tri-cube weighted least squares — no data download.
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

	// Row count from BQ API metadata — no query needed.
	n := int(xCol.Len())

	if n < 2 { //nolint:mnd // LOESS needs at least 2 points.
		return e.LinearFit(xCol, yCol, nOut)
	}

	k := loessTopK(n)

	if nOut > n {
		nOut = n
	}

	sql := fmt.Sprintf(`
WITH
src AS (
  SELECT
    `+"`%[1]s`"+` AS x_val,
    `+"`%[2]s`"+` AS y_val,
    [CAST(`+"`%[1]s`"+` AS FLOAT64)] AS emb
  FROM %[3]s
  WHERE `+"`%[1]s`"+` IS NOT NULL AND `+"`%[2]s`"+` IS NOT NULL
),
stats AS (
  SELECT MIN(x_val) AS x_min, MAX(x_val) AS x_max FROM src
),
grid AS (
  SELECT val AS x_eval, [val] AS emb
  FROM stats,
  UNNEST(GENERATE_ARRAY(x_min, x_max,
    (x_max - x_min) / GREATEST(%[4]d - 1, 1))) AS val
),
neighbors AS (
  SELECT
    query.x_eval,
    base.x_val AS x,
    base.y_val AS y,
    distance
  FROM VECTOR_SEARCH(
    (SELECT x_val, y_val, emb FROM src),
    'emb',
    (SELECT x_eval, emb FROM grid),
    top_k => %[5]d,
    distance_type => 'EUCLIDEAN'
  )
),
tricube AS (
  SELECT
    x_eval, x, y,
    POW(1 - POW(SAFE_DIVIDE(distance,
      GREATEST(MAX(distance) OVER (PARTITION BY x_eval), 1e-12)), 3), 3) AS w,
    x - x_eval AS dx
  FROM neighbors
)
SELECT
  x_eval AS x,
  CASE
    WHEN SUM(w) < 1e-15 THEN 0
    WHEN ABS(SUM(w)*SUM(w*dx*dx) - POW(SUM(w*dx), 2)) < 1e-15
      THEN SAFE_DIVIDE(SUM(w*y), SUM(w))
    ELSE (SUM(w*dx*dx)*SUM(w*y) - SUM(w*dx)*SUM(w*dx*y))
         / (SUM(w)*SUM(w*dx*dx) - POW(SUM(w*dx), 2))
  END AS y
FROM tricube
GROUP BY x_eval
ORDER BY x
`, xName, yName, xSrc, nOut, k)

	schema := dataset.NewSchema(
		dataset.Field{Name: "x", Dtype: dataset.DTypeFloat64},
		dataset.Field{Name: "y", Dtype: dataset.DTypeFloat64},
	)

	return e.lazyStatDataset(sql, schema), nil
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

// LoessFitSE computes LOESS with approximate 95% confidence bands,
// entirely server-side via VECTOR_SEARCH + SQL aggregation.
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

	n := int(xCol.Len())

	if n < 3 { //nolint:mnd // SE requires at least 3 points.
		return e.LinearFitSE(xCol, yCol, nOut)
	}

	k := loessTopK(n)

	if nOut > n {
		nOut = n
	}

	tCrit := 1.96 //nolint:mnd // z ≈ 1.96 for 95% CI.

	sql := fmt.Sprintf(`
WITH
src AS (
  SELECT
    `+"`%[1]s`"+` AS x_val,
    `+"`%[2]s`"+` AS y_val,
    [CAST(`+"`%[1]s`"+` AS FLOAT64)] AS emb
  FROM %[3]s
  WHERE `+"`%[1]s`"+` IS NOT NULL AND `+"`%[2]s`"+` IS NOT NULL
),
stats AS (
  SELECT MIN(x_val) AS x_min, MAX(x_val) AS x_max FROM src
),
grid AS (
  SELECT val AS x_eval, [val] AS emb
  FROM stats,
  UNNEST(GENERATE_ARRAY(x_min, x_max,
    (x_max - x_min) / GREATEST(%[4]d - 1, 1))) AS val
),
neighbors AS (
  SELECT
    query.x_eval,
    base.x_val AS x,
    base.y_val AS y,
    distance
  FROM VECTOR_SEARCH(
    (SELECT x_val, y_val, emb FROM src),
    'emb',
    (SELECT x_eval, emb FROM grid),
    top_k => %[5]d,
    distance_type => 'EUCLIDEAN'
  )
),
tricube AS (
  SELECT
    x_eval, x, y,
    POW(1 - POW(SAFE_DIVIDE(distance,
      GREATEST(MAX(distance) OVER (PARTITION BY x_eval), 1e-12)), 3), 3) AS w,
    x - x_eval AS dx
  FROM neighbors
),
fit AS (
  SELECT
    x_eval,
    SUM(w) AS sw,
    CASE
      WHEN SUM(w) < 1e-15 THEN 0
      WHEN ABS(SUM(w)*SUM(w*dx*dx) - POW(SUM(w*dx), 2)) < 1e-15
        THEN SAFE_DIVIDE(SUM(w*y), SUM(w))
      ELSE (SUM(w*dx*dx)*SUM(w*y) - SUM(w*dx)*SUM(w*dx*y))
           / (SUM(w)*SUM(w*dx*dx) - POW(SUM(w*dx), 2))
    END AS y_hat
  FROM tricube
  GROUP BY x_eval
),
resid AS (
  SELECT
    f.x_eval,
    f.y_hat,
    f.sw,
    SAFE_DIVIDE(
      SUM(t.w * POW(t.y - f.y_hat, 2)),
      SUM(t.w)
    ) AS mse
  FROM fit f
  JOIN tricube t ON t.x_eval = f.x_eval
  GROUP BY f.x_eval, f.y_hat, f.sw
)
SELECT
  x_eval AS x,
  y_hat AS y,
  y_hat - %[6]f * SQRT(SAFE_DIVIDE(mse, sw)) AS ymin,
  y_hat + %[6]f * SQRT(SAFE_DIVIDE(mse, sw)) AS ymax
FROM resid
ORDER BY x
`, xName, yName, xSrc, nOut, k, tCrit)

	schema := dataset.NewSchema(
		dataset.Field{Name: "x", Dtype: dataset.DTypeFloat64},
		dataset.Field{Name: "y", Dtype: dataset.DTypeFloat64},
		dataset.Field{Name: "ymax", Dtype: dataset.DTypeFloat64},
		dataset.Field{Name: "ymin", Dtype: dataset.DTypeFloat64},
	)

	return e.lazyStatDataset(sql, schema), nil
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
