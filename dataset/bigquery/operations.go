package bigquery

import (
	"errors"
	"fmt"
	"strings"

	"github.com/TuSKan/ggplot/dataset"
)

// --- Aggregator ---
//
// All aggregation returns lazy bqDatasets/bqColumns backed by pending SQL.
// No SQL Job executes until Values() is called.

// Sum returns a single-element column containing the SQL SUM.
func (e *Engine) Sum(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.lazyAgg("SUM", col)
}

// Mean returns a single-element column containing the SQL AVG.
func (e *Engine) Mean(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.lazyAgg("AVG", col)
}

// Count returns a single-element int64 column with the SQL COUNT.
func (e *Engine) Count(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.lazyAgg("COUNT", col)
}

// Median returns the approximate median via BQ APPROX_QUANTILES.
func (e *Engine) Median(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	bqCol, ok := col.(*bqColumn)
	if !ok {
		result, medErr := e.localEngine().Median(col)
		if medErr != nil {
			return nil, fmt.Errorf("bigquery: %w", medErr)
		}

		return result, nil
	}
	// BQ: APPROX_QUANTILES(col, 2)[OFFSET(1)]
	sql := fmt.Sprintf(
		"SELECT APPROX_QUANTILES(`%s`, 2)[OFFSET(1)] AS `%s` FROM %s",
		bqCol.name, bqCol.name, bqCol.ds.sourceRef(),
	)
	ds := bqCol.ds.withSQL(sql,
		dataset.NewSchema(dataset.FloatCol(bqCol.name)),
		1,
	)

	return &bqColumn{ds: ds, name: bqCol.name, dtype: dataset.DTypeFloat64}, nil
}

// Variance returns a single-element column containing the SQL VARIANCE.
func (e *Engine) Variance(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.lazyAgg("VARIANCE", col)
}

// MinMax returns two single-element columns containing SQL MIN and MAX.
func (e *Engine) MinMax(col dataset.AnyColumn) (dataset.AnyColumn, dataset.AnyColumn, error) {
	bqCol, ok := col.(*bqColumn)
	if !ok {
		lo, hi, mmErr := e.localEngine().MinMax(col)
		if mmErr != nil {
			return nil, nil, fmt.Errorf("bigquery: %w", mmErr)
		}

		return lo, hi, nil
	}

	minName := "min_" + bqCol.name
	maxName := "max_" + bqCol.name
	sql := fmt.Sprintf(
		"SELECT MIN(`%s`) AS `%s`, MAX(`%s`) AS `%s` FROM %s",
		bqCol.name, minName, bqCol.name, maxName, bqCol.ds.sourceRef(),
	)

	f := dataset.Field{Name: minName, Dtype: bqCol.dtype}
	f2 := dataset.Field{Name: maxName, Dtype: bqCol.dtype}
	schema := dataset.NewSchema(f, f2)

	ds := bqCol.ds.withSQL(sql, schema, 1)
	minCol := &bqColumn{ds: ds, name: minName, dtype: bqCol.dtype}
	maxCol := &bqColumn{ds: ds, name: maxName, dtype: bqCol.dtype}

	return minCol, maxCol, nil
}

// lazyAgg creates a lazy bqColumn backed by a SELECT fn(col) SQL.
func (e *Engine) lazyAgg(fn string, col dataset.AnyColumn) (dataset.AnyColumn, error) {
	bqCol, ok := col.(*bqColumn)
	if !ok {
		var (
			result dataset.AnyColumn
			aggErr error
		)

		switch fn {
		case "SUM":
			result, aggErr = e.localEngine().Sum(col)
		case "AVG":
			result, aggErr = e.localEngine().Mean(col)
		case "COUNT":
			result, aggErr = e.localEngine().Count(col)
		case "VARIANCE":
			result, aggErr = e.localEngine().Variance(col)
		default:
			return nil, fmt.Errorf("bigquery: unknown agg function %q", fn)
		}

		if aggErr != nil {
			return nil, fmt.Errorf("bigquery: %w", aggErr)
		}

		return result, nil
	}

	sql := fmt.Sprintf(
		"SELECT %s(`%s`) AS `%s` FROM %s",
		fn, bqCol.name, bqCol.name, bqCol.ds.sourceRef(),
	)

	dtype := bqCol.dtype
	if fn == "AVG" || fn == "VARIANCE" {
		dtype = dataset.DTypeFloat64
	}

	if fn == "COUNT" {
		dtype = dataset.DTypeInt64
	}

	ds := bqCol.ds.withSQL(sql,
		dataset.NewSchema(dataset.Field{Name: bqCol.name, Dtype: dtype}),
		1,
	)

	return &bqColumn{ds: ds, name: bqCol.name, dtype: dtype}, nil
}

// --- Joiner ---

// Join creates a lazy SQL JOIN between two BigQuery datasets.
func (e *Engine) Join(left, right dataset.Table, spec dataset.JoinSpec) (dataset.Table, error) {
	leftBQ, leftOK := left.(*bqDataset)
	rightBQ, rightOK := right.(*bqDataset)

	if !leftOK || !rightOK {
		return nil, errors.New("bigquery: Join requires both datasets to be BigQuery datasets")
	}

	// Map JoinType → SQL
	switch spec.Type {
	case dataset.JoinSemi:
		return e.lazySemiAntiJoin(leftBQ, rightBQ, spec, true)
	case dataset.JoinAnti:
		return e.lazySemiAntiJoin(leftBQ, rightBQ, spec, false)
	}

	joinSQL := "INNER JOIN"

	switch spec.Type {
	case dataset.JoinLeft:
		joinSQL = "LEFT JOIN"
	case dataset.JoinRight:
		joinSQL = "RIGHT JOIN"
	case dataset.JoinFull:
		joinSQL = "FULL OUTER JOIN"
	}

	// Build ON clause
	onParts := make([]string, len(spec.LeftCols))
	for i := range spec.LeftCols {
		onParts[i] = fmt.Sprintf("L.`%s` = R.`%s`", spec.LeftCols[i], spec.RightCols[i])
	}

	onClause := strings.Join(onParts, " AND ")

	sql := fmt.Sprintf(
		"SELECT * FROM %s AS L %s %s AS R ON %s",
		leftBQ.sourceRef(), joinSQL, rightBQ.sourceRef(), onClause,
	)

	// Build combined schema (left + right)
	leftFields := leftBQ.schema.Fields()
	rightFields := rightBQ.schema.Fields()
	allFields := make([]dataset.Field, 0, len(leftFields)+len(rightFields))
	allFields = append(allFields, leftFields...)
	allFields = append(allFields, rightFields...)
	schema := dataset.NewSchema(allFields...)

	return leftBQ.withSQL(sql, schema, leftBQ.numRows), nil // estimate
}

// lazySemiAntiJoin creates a lazy SEMI/ANTI join via WHERE [NOT] EXISTS.
func (e *Engine) lazySemiAntiJoin(left, right *bqDataset, spec dataset.JoinSpec, semi bool) (dataset.Table, error) {
	onParts := make([]string, len(spec.LeftCols))
	for i := range spec.LeftCols {
		onParts[i] = fmt.Sprintf("L.`%s` = R.`%s`", spec.LeftCols[i], spec.RightCols[i])
	}

	onClause := strings.Join(onParts, " AND ")

	exists := "EXISTS"
	if !semi {
		exists = "NOT EXISTS"
	}

	sql := fmt.Sprintf(
		"SELECT L.* FROM %s AS L WHERE %s (SELECT 1 FROM %s AS R WHERE %s)",
		left.sourceRef(), exists, right.sourceRef(), onClause,
	)

	return left.withSQL(sql, left.schema, left.numRows), nil
}

// --- Composer ---

// Stack vertically concatenates BigQuery datasets via UNION ALL.
func (e *Engine) Stack(datasets ...dataset.Table) (dataset.Table, error) {
	if len(datasets) == 0 {
		return nil, errors.New("bigquery: Stack requires at least one dataset")
	}

	parts := make([]string, len(datasets))
	for i, ds := range datasets {
		bq, ok := ds.(*bqDataset)
		if !ok {
			return nil, errors.New("bigquery: Stack requires all BigQuery datasets")
		}

		parts[i] = "SELECT * FROM " + bq.sourceRef()
	}

	sql := strings.Join(parts, " UNION ALL ")
	first := datasets[0].(*bqDataset)

	var totalRows int64
	for _, ds := range datasets {
		totalRows += ds.NumRows()
	}

	return first.withSQL(sql, first.schema, totalRows), nil
}

// Combine horizontally concatenates datasets (downloads then delegates).
func (e *Engine) Combine(datasets ...dataset.Table) (dataset.Table, error) {
	if len(datasets) == 0 {
		return nil, errors.New("bigquery: Combine requires at least one dataset")
	}

	// Column bind — download and delegate to arrow engine
	localDS := make([]dataset.Table, len(datasets))
	for i, ds := range datasets {
		bq, ok := ds.(*bqDataset)
		if ok {
			var err error

			localDS[i], err = bq.download()
			if err != nil {
				return nil, err
			}
		} else {
			localDS[i] = ds
		}
	}

	var eng dataset.Engine = e.localEngine()

	composer, ok := eng.(dataset.Composer)
	if !ok {
		return nil, errors.New("bigquery: local engine does not support Composer")
	}

	result, combErr := composer.Combine(localDS...)
	if combErr != nil {
		return nil, fmt.Errorf("bigquery: %w", combErr)
	}

	return result, nil
}

// --- Filler ---

// Fill forward- or backward-fills null values (downloads then delegates).
func (e *Engine) Fill(col dataset.AnyColumn, dir dataset.FillDirection) (dataset.AnyColumn, error) {
	// Window-based fill is complex — download and delegate
	if bqCol, ok := col.(*bqColumn); ok {
		ds, err := bqCol.ds.download()
		if err != nil {
			return nil, err
		}

		localCol, err := ds.Column(bqCol.name)
		if err != nil {
			return nil, fmt.Errorf("bigquery: %w", err)
		}

		result, fillErr := e.localEngine().Fill(localCol, dir)
		if fillErr != nil {
			return nil, fmt.Errorf("bigquery: %w", fillErr)
		}

		return result, nil
	}

	result, fillErr := e.localEngine().Fill(col, dir)
	if fillErr != nil {
		return nil, fmt.Errorf("bigquery: %w", fillErr)
	}

	return result, nil
}

// DropNA returns a dataset with rows filtered by IS NOT NULL.
func (e *Engine) DropNA(ds dataset.Table, cols ...string) (dataset.Table, error) {
	if bq, ok := ds.(*bqDataset); ok {
		// Pure RowRestriction — fully lazy
		parts := make([]string, len(cols))
		for i, c := range cols {
			parts[i] = fmt.Sprintf("`%s` IS NOT NULL", c)
		}

		return bq.withRestriction(strings.Join(parts, " AND ")), nil
	}

	result, dropErr := e.localEngine().DropNA(ds, cols...)
	if dropErr != nil {
		return nil, fmt.Errorf("bigquery: %w", dropErr)
	}

	return result, nil
}

// ReplaceNA replaces null values with a default via SQL COALESCE.
func (e *Engine) ReplaceNA(col dataset.AnyColumn, defaultVal float64) (dataset.AnyColumn, error) {
	if bqCol, ok := col.(*bqColumn); ok {
		// COALESCE — lazy SQL
		sql := fmt.Sprintf(
			"SELECT COALESCE(`%s`, %v) AS `%s` FROM %s",
			bqCol.name, defaultVal, bqCol.name, bqCol.ds.sourceRef(),
		)
		ds := bqCol.ds.withSQL(sql,
			dataset.NewSchema(dataset.Field{Name: bqCol.name, Dtype: bqCol.dtype}),
			bqCol.ds.numRows,
		)

		return &bqColumn{ds: ds, name: bqCol.name, dtype: bqCol.dtype}, nil
	}

	result, repErr := e.localEngine().ReplaceNA(col, defaultVal)
	if repErr != nil {
		return nil, fmt.Errorf("bigquery: %w", repErr)
	}

	return result, nil
}
