package bigquery

import (
	"fmt"

	"github.com/TuSKan/ggplot/dataset"
)

// --- Windower ---
//
// All window functions return lazy bqColumns backed by pending SQL.
// No execution until Values() is called.

// Lag shifts column values down by n positions (SQL LAG).
func (e *Engine) Lag(col dataset.AnyColumn, n int) (dataset.AnyColumn, error) {
	return e.lazyWindowFn("LAG", col, n)
}

// Lead shifts column values up by n positions (SQL LEAD).
func (e *Engine) Lead(col dataset.AnyColumn, n int) (dataset.AnyColumn, error) {
	return e.lazyWindowFn("LEAD", col, n)
}

// CumSum returns the cumulative sum (SQL SUM OVER window).
func (e *Engine) CumSum(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.lazyCumulativeWindowFn("SUM", col)
}

// CumMax returns the cumulative maximum (SQL MAX OVER window).
func (e *Engine) CumMax(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.lazyCumulativeWindowFn("MAX", col)
}

// CumMin returns the cumulative minimum (SQL MIN OVER window).
func (e *Engine) CumMin(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.lazyCumulativeWindowFn("MIN", col)
}

// Rank returns the 1-based rank (SQL RANK OVER ORDER BY).
func (e *Engine) Rank(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.lazyRankWindowFn("RANK", col)
}

// DenseRank returns the dense rank (SQL DENSE_RANK OVER ORDER BY).
func (e *Engine) DenseRank(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.lazyRankWindowFn("DENSE_RANK", col)
}

// PercentRank returns the percent rank (SQL PERCENT_RANK).
func (e *Engine) PercentRank(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.lazyRankWindowFn("PERCENT_RANK", col)
}

// RowNumber returns a 1-based sequential row-number column (delegates locally).
func (e *Engine) RowNumber(n int) (dataset.AnyColumn, error) {
	return e.localEngine().RowNumber(n)
}

// lazyWindowFn creates a lazy LAG/LEAD bqColumn.
func (e *Engine) lazyWindowFn(fn string, col dataset.AnyColumn, n int) (dataset.AnyColumn, error) {
	bqCol, ok := col.(*bqColumn)
	if !ok {
		switch fn {
		case "LAG":
			return e.localEngine().Lag(col, n)
		case "LEAD":
			return e.localEngine().Lead(col, n)
		default:
			return nil, fmt.Errorf("bigquery: unknown window function %q", fn)
		}
	}

	resultName := fmt.Sprintf("%s_%s_%d", fn, bqCol.name, n)
	sql := fmt.Sprintf(
		"SELECT *, %s(`%s`, %d) OVER() AS `%s` FROM %s",
		fn, bqCol.name, n, resultName, bqCol.ds.sourceRef(),
	)

	// Build schema: original + new column
	origFields := bqCol.ds.schema.Fields()
	newFields := make([]dataset.Field, len(origFields)+1)
	copy(newFields, origFields)
	newFields[len(origFields)] = dataset.Field{Name: resultName, Dtype: bqCol.dtype}
	schema := dataset.NewSchema(newFields...)

	ds := bqCol.ds.withSQL(sql, schema, bqCol.ds.numRows)

	return &bqColumn{ds: ds, name: resultName, dtype: bqCol.dtype}, nil
}

// lazyCumulativeWindowFn creates a lazy cumulative window bqColumn.
func (e *Engine) lazyCumulativeWindowFn(fn string, col dataset.AnyColumn) (dataset.AnyColumn, error) {
	bqCol, ok := col.(*bqColumn)
	if !ok {
		switch fn {
		case "SUM":
			return e.localEngine().CumSum(col)
		case "MAX":
			return e.localEngine().CumMax(col)
		case "MIN":
			return e.localEngine().CumMin(col)
		default:
			return nil, fmt.Errorf("bigquery: unknown cumulative function %q", fn)
		}
	}

	resultName := fmt.Sprintf("cum_%s_%s", fn, bqCol.name)
	sql := fmt.Sprintf(
		"SELECT *, %s(`%s`) OVER(ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) AS `%s` FROM %s",
		fn, bqCol.name, resultName, bqCol.ds.sourceRef(),
	)

	origFields := bqCol.ds.schema.Fields()
	newFields := make([]dataset.Field, len(origFields)+1)
	copy(newFields, origFields)
	newFields[len(origFields)] = dataset.Field{Name: resultName, Dtype: bqCol.dtype}
	schema := dataset.NewSchema(newFields...)

	ds := bqCol.ds.withSQL(sql, schema, bqCol.ds.numRows)

	return &bqColumn{ds: ds, name: resultName, dtype: bqCol.dtype}, nil
}

// lazyRankWindowFn creates a lazy ranking window bqColumn.
func (e *Engine) lazyRankWindowFn(fn string, col dataset.AnyColumn) (dataset.AnyColumn, error) {
	bqCol, ok := col.(*bqColumn)
	if !ok {
		switch fn {
		case "RANK":
			return e.localEngine().Rank(col)
		case "DENSE_RANK":
			return e.localEngine().DenseRank(col)
		case "PERCENT_RANK":
			return e.localEngine().PercentRank(col)
		default:
			return nil, fmt.Errorf("bigquery: unknown rank function %q", fn)
		}
	}

	resultName := fmt.Sprintf("%s_%s", fn, bqCol.name)

	dtype := dataset.DTypeInt64
	if fn == "PERCENT_RANK" {
		dtype = dataset.DTypeFloat64
	}

	sql := fmt.Sprintf(
		"SELECT *, %s() OVER(ORDER BY `%s`) AS `%s` FROM %s",
		fn, bqCol.name, resultName, bqCol.ds.sourceRef(),
	)

	origFields := bqCol.ds.schema.Fields()
	newFields := make([]dataset.Field, len(origFields)+1)
	copy(newFields, origFields)
	newFields[len(origFields)] = dataset.Field{Name: resultName, Dtype: dtype}
	schema := dataset.NewSchema(newFields...)

	ds := bqCol.ds.withSQL(sql, schema, bqCol.ds.numRows)

	return &bqColumn{ds: ds, name: resultName, dtype: dtype}, nil
}

// --- Caster ---

// Cast converts a column to the target DType via SQL CAST.
func (e *Engine) Cast(col dataset.AnyColumn, target dataset.DType) (dataset.AnyColumn, error) {
	bqCol, ok := col.(*bqColumn)
	if !ok {
		return e.localEngine().Cast(col, target)
	}

	bqType := DtypeToBQSQL(target)
	sql := fmt.Sprintf(
		"SELECT CAST(`%s` AS %s) AS `%s` FROM %s",
		bqCol.name, bqType, bqCol.name, bqCol.ds.sourceRef(),
	)

	ds := bqCol.ds.withSQL(sql,
		dataset.NewSchema(dataset.Field{Name: bqCol.name, Dtype: target}),
		bqCol.ds.numRows,
	)

	return &bqColumn{ds: ds, name: bqCol.name, dtype: target}, nil
}

// DtypeToBQSQL maps dataset.DType to BigQuery SQL type names.
func DtypeToBQSQL(dt dataset.DType) string {
	switch dt {
	case dataset.DTypeFloat64:
		return "FLOAT64"
	case dataset.DTypeInt64:
		return "INT64"
	case dataset.DTypeString:
		return "STRING"
	case dataset.DTypeBool:
		return "BOOL"
	case dataset.DTypeTimestamp:
		return "TIMESTAMP"
	default:
		return "STRING"
	}
}
