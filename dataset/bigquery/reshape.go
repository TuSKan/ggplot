package bigquery

import (
	"errors"
	"fmt"
	"strings"

	"github.com/TuSKan/ggplot/dataset"
)

// --- Reshaper ---
//
// Pivot operations generate SQL UNPIVOT/CASE WHEN expressions.
// All return lazy bqDatasets backed by pendingSQL.

// PivotLonger gathers multiple columns into name+value pairs.
// SQL: UNPIVOT (value FOR name IN (col1, col2, ...))
func (e *Engine) PivotLonger(ds dataset.Table, spec dataset.PivotLongerSpec) (dataset.Table, error) {
	bq, ok := ds.(*bqDataset)
	if !ok {
		return nil, errors.New("bigquery: PivotLonger requires a BigQuery dataset")
	}

	if len(spec.Cols) == 0 {
		return ds, nil
	}

	// Identify non-pivot (id) columns
	pivotSet := make(map[string]bool, len(spec.Cols))
	for _, c := range spec.Cols {
		pivotSet[c] = true
	}

	var idCols []string

	for _, f := range bq.schema.Fields() {
		if !pivotSet[f.Name] {
			idCols = append(idCols, f.Name)
		}
	}

	// Build UNPIVOT SQL
	// SELECT id_cols, namesTo, valuesTo
	// FROM source
	// UNPIVOT (valuesTo FOR namesTo IN (col1, col2, ...))
	pivotColList := make([]string, len(spec.Cols))
	for i, c := range spec.Cols {
		pivotColList[i] = "`" + c + "`"
	}

	sql := fmt.Sprintf(
		"SELECT * FROM %s UNPIVOT (`%s` FOR `%s` IN (%s))",
		bq.sourceRef(),
		spec.ValuesTo,
		spec.NamesTo,
		strings.Join(pivotColList, ", "),
	)

	// Build output schema: id columns + namesTo (string) + valuesTo (float64)
	outFields := make([]dataset.Field, 0, len(idCols)+2)
	for _, name := range idCols {
		idx := bq.schema.FieldIndex(name)
		if idx >= 0 {
			outFields = append(outFields, bq.schema.Field(idx))
		}
	}

	outFields = append(outFields, dataset.Field{Name: spec.NamesTo, Dtype: dataset.DTypeString})

	// Infer value dtype from first pivot column
	valueDtype := dataset.DTypeFloat64

	if len(spec.Cols) > 0 {
		idx := bq.schema.FieldIndex(spec.Cols[0])
		if idx >= 0 {
			valueDtype = bq.schema.Field(idx).Dtype
		}
	}

	outFields = append(outFields, dataset.Field{Name: spec.ValuesTo, Dtype: valueDtype})

	schema := dataset.NewSchema(outFields...)
	estimatedRows := bq.numRows * int64(len(spec.Cols))

	return bq.withSQL(sql, schema, estimatedRows), nil
}

// PivotWider spreads a name column's values into new columns.
// SQL: SELECT id_cols, MAX(IF(name_col='val1', value_col, NULL)) AS val1, ...
// FROM source GROUP BY id_cols
func (e *Engine) PivotWider(ds dataset.Table, spec dataset.PivotWiderSpec) (dataset.Table, error) {
	bq, ok := ds.(*bqDataset)
	if !ok {
		return nil, errors.New("bigquery: PivotWider requires a BigQuery dataset")
	}

	// PivotWider needs to know the distinct values of NamesFrom.
	// This requires a query — but we keep the result lazy.
	// For now, materialize to get distinct names, then build the pivot SQL.
	mat, err := bq.materialize()
	if err != nil {
		return nil, err
	}

	// Get distinct values of the NamesFrom column
	distinctSQL := fmt.Sprintf(
		"SELECT DISTINCT `%s` FROM `%s` ORDER BY 1",
		spec.NamesFrom, mat.table.FullyQualified(),
	)

	distinctRef, distinctMeta, err := e.execToTempTable(distinctSQL)
	if err != nil {
		return nil, fmt.Errorf("bigquery: PivotWider distinct query failed: %w", err)
	}

	// Download the distinct names (small result)
	distinctDS := &bqDataset{
		engine:  e,
		schema:  bqSchemaToDataset(distinctMeta.Schema),
		table:   distinctRef,
		numRows: int64(distinctMeta.NumRows),
	}

	localDistinct, err := distinctDS.download()
	if err != nil {
		return nil, err
	}

	namesCol, err := localDistinct.Column(spec.NamesFrom)
	if err != nil {
		return nil, fmt.Errorf("bigquery: %w", err)
	}

	// Extract string values
	var pivotNames []string
	if typedCol, ok := namesCol.(dataset.Column[string]); ok {
		pivotNames = typedCol.Values()
	} else {
		// Fallback: convert to strings
		for i := range namesCol.Len() {
			pivotNames = append(pivotNames, fmt.Sprintf("v%d", i))
		}
	}

	// Build the CASE WHEN pivot SQL
	var idCols []string

	for _, f := range bq.schema.Fields() {
		if f.Name != spec.NamesFrom && f.Name != spec.ValuesFrom {
			idCols = append(idCols, f.Name)
		}
	}

	// SELECT id_cols, MAX(IF(name='val1', value, NULL)) AS val1, ...
	// FROM source GROUP BY id_cols
	selectParts := make([]string, 0, len(idCols)+len(pivotNames))
	for _, c := range idCols {
		selectParts = append(selectParts, "`"+c+"`")
	}

	for _, name := range pivotNames {
		selectParts = append(selectParts, fmt.Sprintf(
			"MAX(IF(`%s` = '%s', `%s`, NULL)) AS `%s`",
			spec.NamesFrom, name, spec.ValuesFrom, name,
		))
	}

	groupByParts := make([]string, len(idCols))
	for i, c := range idCols {
		groupByParts[i] = "`" + c + "`"
	}

	sql := fmt.Sprintf(
		"SELECT %s FROM %s GROUP BY %s",
		strings.Join(selectParts, ", "),
		bq.sourceRef(),
		strings.Join(groupByParts, ", "),
	)

	// Build output schema
	valueDtype := dataset.DTypeFloat64

	idx := bq.schema.FieldIndex(spec.ValuesFrom)
	if idx >= 0 {
		valueDtype = bq.schema.Field(idx).Dtype
	}

	outFields := make([]dataset.Field, 0, len(idCols)+len(pivotNames))
	for _, c := range idCols {
		fidx := bq.schema.FieldIndex(c)
		if fidx >= 0 {
			outFields = append(outFields, bq.schema.Field(fidx))
		}
	}

	for _, name := range pivotNames {
		outFields = append(outFields, dataset.Field{Name: name, Dtype: valueDtype, Nullable: true})
	}

	schema := dataset.NewSchema(outFields...)

	return bq.withSQL(sql, schema, bq.numRows/int64(max(len(pivotNames), 1))), nil
}

// Separate splits a string column into multiple columns by a separator.
// SQL: SPLIT(col, sep)[OFFSET(i)] AS into_i
func (e *Engine) Separate(ds dataset.Table, col string, into []string, sep string) (dataset.Table, error) {
	bq, ok := ds.(*bqDataset)
	if !ok {
		return nil, errors.New("bigquery: Separate requires a BigQuery dataset")
	}

	// Build SELECT with SPLIT
	var selectParts []string

	for _, f := range bq.schema.Fields() {
		if f.Name == col {
			for i, name := range into {
				selectParts = append(selectParts, fmt.Sprintf(
					"SPLIT(`%s`, '%s')[SAFE_OFFSET(%d)] AS `%s`",
					col, sep, i, name,
				))
			}
		} else {
			selectParts = append(selectParts, "`"+f.Name+"`")
		}
	}

	sql := fmt.Sprintf("SELECT %s FROM %s", strings.Join(selectParts, ", "), bq.sourceRef())

	// Build schema: replace col with into columns
	var outFields []dataset.Field

	for _, f := range bq.schema.Fields() {
		if f.Name == col {
			for _, name := range into {
				outFields = append(outFields, dataset.Field{Name: name, Dtype: dataset.DTypeString, Nullable: true})
			}
		} else {
			outFields = append(outFields, f)
		}
	}

	return bq.withSQL(sql, dataset.NewSchema(outFields...), bq.numRows), nil
}

// Concatenate joins multiple string columns into one with a separator.
// SQL: CONCAT(col1, sep, col2, sep, col3) AS output
func (e *Engine) Concatenate(ds dataset.Table, col string, from []string, sep string) (dataset.Table, error) {
	bq, ok := ds.(*bqDataset)
	if !ok {
		return nil, errors.New("bigquery: Concatenate requires a BigQuery dataset")
	}

	fromSet := make(map[string]bool, len(from))
	for _, c := range from {
		fromSet[c] = true
	}

	// Build CONCAT expression
	concatParts := make([]string, len(from))
	for i, c := range from {
		concatParts[i] = "`" + c + "`"
		if i < len(from)-1 {
			concatParts[i] += fmt.Sprintf(", '%s'", sep)
		}
	}

	concatExpr := "CONCAT(" + strings.Join(concatParts, ", ") + ")"

	// Build SELECT: keep non-from columns + new concat column
	var selectParts []string

	for _, f := range bq.schema.Fields() {
		if !fromSet[f.Name] {
			selectParts = append(selectParts, "`"+f.Name+"`")
		}
	}

	selectParts = append(selectParts, fmt.Sprintf("%s AS `%s`", concatExpr, col))

	sql := fmt.Sprintf("SELECT %s FROM %s", strings.Join(selectParts, ", "), bq.sourceRef())

	// Build schema
	var outFields []dataset.Field

	for _, f := range bq.schema.Fields() {
		if !fromSet[f.Name] {
			outFields = append(outFields, f)
		}
	}

	outFields = append(outFields, dataset.Field{Name: col, Dtype: dataset.DTypeString})

	return bq.withSQL(sql, dataset.NewSchema(outFields...), bq.numRows), nil
}

// Complete generates all combinations of the given columns.
// SQL: CROSS JOIN of DISTINCT values for each column.
func (e *Engine) Complete(ds dataset.Table, cols ...string) (dataset.Table, error) {
	bq, ok := ds.(*bqDataset)
	if !ok {
		return nil, errors.New("bigquery: Complete requires a BigQuery dataset")
	}

	if len(cols) == 0 {
		return ds, nil
	}

	// Build CROSS JOIN of DISTINCT values
	crossParts := make([]string, len(cols))
	for i, c := range cols {
		crossParts[i] = fmt.Sprintf(
			"(SELECT DISTINCT `%s` FROM %s) AS t%d",
			c, bq.sourceRef(), i,
		)
	}

	sql := "SELECT * FROM " + strings.Join(crossParts, " CROSS JOIN ")

	// Build schema from selected columns
	outFields := make([]dataset.Field, len(cols))
	for i, c := range cols {
		idx := bq.schema.FieldIndex(c)
		if idx >= 0 {
			outFields[i] = bq.schema.Field(idx)
		} else {
			outFields[i] = dataset.Field{Name: c, Dtype: dataset.DTypeString}
		}
	}

	return bq.withSQL(sql, dataset.NewSchema(outFields...), bq.numRows), nil
}
