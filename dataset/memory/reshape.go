package memory

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/TuSKan/ggplot/dataset"
)

// PivotLonger reshapes a wide dataset to long format.
// Columns listed in spec.Cols are "gathered" into two new columns:
// spec.NamesTo (holds original column names) and spec.ValuesTo (holds values).
// All other columns are repeated for each gathered column.
func (e *Engine) PivotLonger(ds dataset.Table, spec dataset.PivotLongerSpec) (dataset.Table, error) {
	if len(spec.Cols) == 0 {
		return nil, errors.New("memory: PivotLonger requires at least one column to pivot")
	}

	if spec.NamesTo == "" {
		spec.NamesTo = "name"
	}

	if spec.ValuesTo == "" {
		spec.ValuesTo = "value"
	}

	schema := ds.Schema()
	nRows := int(ds.NumRows())
	nPivot := len(spec.Cols)
	outLen := nRows * nPivot

	// Identify pivot and id (non-pivot) columns.
	pivotSet := make(map[string]bool, nPivot)

	for _, name := range spec.Cols {
		if !schema.HasField(name) {
			return nil, fmt.Errorf("memory: PivotLonger: column %q not found", name)
		}

		pivotSet[name] = true
	}

	// Validate all pivot columns have the same type.
	var pivotDType dataset.DType

	for i, name := range spec.Cols {
		f := schema.Field(schema.FieldIndex(name))
		if i == 0 {
			pivotDType = f.Dtype
		} else if f.Dtype != pivotDType {
			return nil, fmt.Errorf("memory: PivotLonger: column %q has type %s, expected %s",
				name, f.Dtype, pivotDType)
		}
	}

	// Build output: repeat id columns, interleave pivot values.
	var (
		outFields []dataset.Field
		outCols   []dataset.AnyColumn
	)

	// ID columns: each value repeats nPivot times.
	for i := 0; i < schema.NumFields(); i++ {
		f := schema.Field(i)
		if pivotSet[f.Name] {
			continue
		}

		outFields = append(outFields, f)
		col, _ := ds.Column(f.Name)
		outCols = append(outCols, repeatColumn(col, nPivot, outLen, f.Name))
	}

	// Names column: cycle through pivot column names.
	namesData := make([]string, outLen)

	for row := range nRows {
		for p, name := range spec.Cols {
			namesData[row*nPivot+p] = name
		}
	}

	outFields = append(outFields, dataset.StringCol(spec.NamesTo))
	outCols = append(outCols, &stringColumn{name: spec.NamesTo, data: namesData})

	// Values column: interleave pivot column values.
	outFields = append(outFields, dataset.Field{Name: spec.ValuesTo, Dtype: pivotDType})
	valCol := gatherPivotValues(ds, spec.Cols, pivotDType, nRows, nPivot, outLen, spec.ValuesTo)
	outCols = append(outCols, valCol)

	outSchema := dataset.NewSchema(outFields...)

	return e.FromColumns(outSchema, outCols...)
}

// repeatColumn repeats each element of col `times` times consecutively.
func repeatColumn(col dataset.AnyColumn, times, outLen int, name string) dataset.AnyColumn {
	switch c := col.(type) {
	case *float64Column:
		out := make([]float64, outLen)

		for i, v := range c.data {
			for t := range times {
				out[i*times+t] = v
			}
		}

		return &float64Column{name: name, data: out}
	case *int64Column:
		out := make([]int64, outLen)

		for i, v := range c.data {
			for t := range times {
				out[i*times+t] = v
			}
		}

		return &int64Column{name: name, data: out, dtype: c.dtype}
	case *stringColumn:
		out := make([]string, outLen)

		for i, v := range c.data {
			for t := range times {
				out[i*times+t] = v
			}
		}

		return &stringColumn{name: name, data: out}
	case *boolColumn:
		out := make([]bool, outLen)

		for i, v := range c.data {
			for t := range times {
				out[i*times+t] = v
			}
		}

		return &boolColumn{name: name, data: out}
	default:
		return col
	}
}

// gatherPivotValues collects values from multiple pivot columns into one.
func gatherPivotValues(ds dataset.Table, cols []string, dtype dataset.DType, nRows, nPivot, outLen int, name string) dataset.AnyColumn {
	switch dtype {
	case dataset.DTypeFloat64:
		out := make([]float64, outLen)

		for p, colName := range cols {
			col, _ := ds.Column(colName)

			c := col.(*float64Column)
			for row := range nRows {
				out[row*nPivot+p] = c.data[row]
			}
		}

		return &float64Column{name: name, data: out}
	case dataset.DTypeInt64:
		out := make([]int64, outLen)

		for p, colName := range cols {
			col, _ := ds.Column(colName)

			c := col.(*int64Column)
			for row := range nRows {
				out[row*nPivot+p] = c.data[row]
			}
		}

		return &int64Column{name: name, data: out}
	case dataset.DTypeString:
		out := make([]string, outLen)

		for p, colName := range cols {
			col, _ := ds.Column(colName)

			c := col.(*stringColumn)
			for row := range nRows {
				out[row*nPivot+p] = c.data[row]
			}
		}

		return &stringColumn{name: name, data: out}
	default:
		return &stringColumn{name: name, data: make([]string, outLen)}
	}
}

// PivotWider reshapes a long dataset to wide format.
// spec.NamesFrom identifies the column whose unique values become new column names.
// spec.ValuesFrom identifies the column whose values fill the new columns.
// All other columns are the "id" columns that define unique rows.
func (e *Engine) PivotWider(ds dataset.Table, spec dataset.PivotWiderSpec) (dataset.Table, error) {
	if spec.NamesFrom == "" || spec.ValuesFrom == "" {
		return nil, errors.New("memory: PivotWider requires NamesFrom and ValuesFrom")
	}

	schema := ds.Schema()
	nRows := int(ds.NumRows())

	nameCol, err := ds.Column(spec.NamesFrom)
	if err != nil {
		return nil, err
	}

	valCol, err := ds.Column(spec.ValuesFrom)
	if err != nil {
		return nil, err
	}

	nameStr, ok := nameCol.(*stringColumn)
	if !ok {
		return nil, fmt.Errorf("memory: PivotWider: NamesFrom %q must be string column", spec.NamesFrom)
	}

	// Find unique pivot names (preserving order).
	var pivotNames []string

	seen := map[string]bool{}
	for _, v := range nameStr.data {
		if !seen[v] {
			pivotNames = append(pivotNames, v)
			seen[v] = true
		}
	}

	// Identify id columns (everything except NamesFrom and ValuesFrom).
	var idColNames []string

	for i := 0; i < schema.NumFields(); i++ {
		f := schema.Field(i)
		if f.Name == spec.NamesFrom || f.Name == spec.ValuesFrom {
			continue
		}

		idColNames = append(idColNames, f.Name)
	}

	// Build id-key → output row index mapping.
	idCols := make([]dataset.AnyColumn, len(idColNames))
	for i, name := range idColNames {
		idCols[i], _ = ds.Column(name)
	}

	// Map each unique id-key combination to an output row.
	idKeyToRow := map[string]int{}

	rowIDKeys := make([]string, nRows)
	for row := range nRows {
		key := idKey(idCols, row)

		rowIDKeys[row] = key
		if _, ok := idKeyToRow[key]; !ok {
			idKeyToRow[key] = len(idKeyToRow)
		}
	}

	outLen := len(idKeyToRow)

	// Build output schema: id columns + pivot-name columns.
	var outFields []dataset.Field
	for _, name := range idColNames {
		outFields = append(outFields, schema.Field(schema.FieldIndex(name)))
	}

	valDType := valCol.DType()
	for _, pn := range pivotNames {
		outFields = append(outFields, dataset.Field{Name: pn, Dtype: valDType})
	}

	outSchema := dataset.NewSchema(outFields...)

	// Gather id columns (take first occurrence for each id-key).
	var outCols []dataset.AnyColumn

	firstRow := make([]int, outLen)

	for row := range nRows {
		idx := idKeyToRow[rowIDKeys[row]]
		if row == 0 || firstRow[idx] == 0 {
			firstRow[idx] = row
		}
	}
	// Fix: firstRow[0] might be 0 legitimately, use a different approach.
	firstRowMap := map[int]int{}

	for row := range nRows {
		idx := idKeyToRow[rowIDKeys[row]]
		if _, exists := firstRowMap[idx]; !exists {
			firstRowMap[idx] = row
		}
	}

	idIndices := make([]int, outLen)
	for idx, row := range firstRowMap {
		idIndices[idx] = row
	}

	for _, name := range idColNames {
		col, _ := ds.Column(name)
		outCols = append(outCols, gatherColumn(col, idIndices, outLen, name))
	}

	// Build pivot-value columns.
	pivotNameToIdx := map[string]int{}
	for i, pn := range pivotNames {
		pivotNameToIdx[pn] = i
	}

	switch c := valCol.(type) {
	case *float64Column:
		pivotData := make([][]float64, len(pivotNames))
		for i := range pivotData {
			pivotData[i] = make([]float64, outLen)
			for j := range pivotData[i] {
				pivotData[i][j] = math.NaN()
			}
		}

		for row := range nRows {
			outRow := idKeyToRow[rowIDKeys[row]]
			pIdx := pivotNameToIdx[nameStr.data[row]]
			pivotData[pIdx][outRow] = c.data[row]
		}

		for i, pn := range pivotNames {
			outCols = append(outCols, &float64Column{name: pn, data: pivotData[i]})
		}
	case *int64Column:
		pivotData := make([][]int64, len(pivotNames))
		for i := range pivotData {
			pivotData[i] = make([]int64, outLen)
		}

		for row := range nRows {
			outRow := idKeyToRow[rowIDKeys[row]]
			pIdx := pivotNameToIdx[nameStr.data[row]]
			pivotData[pIdx][outRow] = c.data[row]
		}

		for i, pn := range pivotNames {
			outCols = append(outCols, &int64Column{name: pn, data: pivotData[i], dtype: c.dtype})
		}
	case *stringColumn:
		pivotData := make([][]string, len(pivotNames))
		for i := range pivotData {
			pivotData[i] = make([]string, outLen)
		}

		for row := range nRows {
			outRow := idKeyToRow[rowIDKeys[row]]
			pIdx := pivotNameToIdx[nameStr.data[row]]
			pivotData[pIdx][outRow] = c.data[row]
		}

		for i, pn := range pivotNames {
			outCols = append(outCols, &stringColumn{name: pn, data: pivotData[i]})
		}
	default:
		return nil, fmt.Errorf("memory: PivotWider: unsupported value type %s", valDType)
	}

	return e.FromColumns(outSchema, outCols...)
}

// idKey creates a composite key from id columns for a given row.
func idKey(cols []dataset.AnyColumn, row int) string {
	if len(cols) == 0 {
		return ""
	}

	if len(cols) == 1 {
		return colValueString(cols[0], row)
	}

	var b strings.Builder

	for i, col := range cols {
		if i > 0 {
			b.WriteByte('\x00')
		}

		b.WriteString(colValueString(col, row))
	}

	return b.String()
}

// Separate splits a string column by a delimiter into multiple columns.
func (e *Engine) Separate(ds dataset.Table, col string, into []string, sep string) (dataset.Table, error) {
	srcCol, err := ds.Column(col)
	if err != nil {
		return nil, err
	}

	sc, ok := srcCol.(*stringColumn)
	if !ok {
		return nil, fmt.Errorf("memory: Separate: column %q must be string", col)
	}

	n := int(ds.NumRows())
	nParts := len(into)

	partData := make([][]string, nParts)
	for i := range partData {
		partData[i] = make([]string, n)
	}

	for row := range n {
		parts := strings.SplitN(sc.data[row], sep, nParts)
		for p := range nParts {
			if p < len(parts) {
				partData[p][row] = parts[p]
			}
		}
	}

	// Build output: replace source column with split columns.
	schema := ds.Schema()

	var (
		outFields []dataset.Field
		outCols   []dataset.AnyColumn
	)

	for i := 0; i < schema.NumFields(); i++ {
		f := schema.Field(i)
		if f.Name == col {
			// Replace with split columns.
			for p, name := range into {
				outFields = append(outFields, dataset.StringCol(name))
				outCols = append(outCols, &stringColumn{name: name, data: partData[p]})
			}
		} else {
			outFields = append(outFields, f)
			c, _ := ds.Column(f.Name)
			outCols = append(outCols, c)
		}
	}

	outSchema := dataset.NewSchema(outFields...)

	return e.FromColumns(outSchema, outCols...)
}

// Concatenate joins multiple string columns into one with a separator.
func (e *Engine) Concatenate(ds dataset.Table, col string, from []string, sep string) (dataset.Table, error) {
	n := int(ds.NumRows())
	schema := ds.Schema()

	srcCols := make([]*stringColumn, len(from))
	for i, name := range from {
		c, err := ds.Column(name)
		if err != nil {
			return nil, err
		}

		sc, ok := c.(*stringColumn)
		if !ok {
			return nil, fmt.Errorf("memory: Concatenate: column %q must be string", name)
		}

		srcCols[i] = sc
	}

	// Build concatenated data.
	data := make([]string, n)
	parts := make([]string, len(from))

	for row := range n {
		for i, sc := range srcCols {
			parts[i] = sc.data[row]
		}

		data[row] = strings.Join(parts, sep)
	}

	// Build output: keep non-from columns + add new column.
	fromSet := make(map[string]bool, len(from))
	for _, name := range from {
		fromSet[name] = true
	}

	var (
		outFields []dataset.Field
		outCols   []dataset.AnyColumn
	)

	added := false

	for i := 0; i < schema.NumFields(); i++ {
		f := schema.Field(i)
		if fromSet[f.Name] {
			if !added {
				// Insert concatenated column at position of first source column.
				outFields = append(outFields, dataset.StringCol(col))
				outCols = append(outCols, &stringColumn{name: col, data: data})
				added = true
			}

			continue
		}

		outFields = append(outFields, f)
		c, _ := ds.Column(f.Name)
		outCols = append(outCols, c)
	}

	if !added {
		outFields = append(outFields, dataset.StringCol(col))
		outCols = append(outCols, &stringColumn{name: col, data: data})
	}

	outSchema := dataset.NewSchema(outFields...)

	return e.FromColumns(outSchema, outCols...)
}

// Complete generates all combinations of the specified columns' unique values,
// filling missing rows with null values.
func (e *Engine) Complete(ds dataset.Table, cols ...string) (dataset.Table, error) {
	if len(cols) == 0 {
		return ds, nil
	}

	schema := ds.Schema()
	n := int(ds.NumRows())

	// Collect unique values for each specified column.
	type colUniques struct {
		name   string
		values []string
	}

	uniques := make([]colUniques, len(cols))
	for i, name := range cols {
		col, err := ds.Column(name)
		if err != nil {
			return nil, err
		}

		seen := map[string]bool{}

		var vals []string

		for row := range n {
			v := colValueString(col, row)
			if !seen[v] {
				vals = append(vals, v)
				seen[v] = true
			}
		}

		uniques[i] = colUniques{name: name, values: vals}
	}

	// Generate all combinations (cartesian product).
	totalCombos := 1
	for _, u := range uniques {
		totalCombos *= len(u.values)
	}

	combos := make([][]string, totalCombos)
	repeat := 1

	for i, v := range slices.Backward(uniques) {
		uLen := len(v.values)

		for c := range totalCombos {
			if combos[c] == nil {
				combos[c] = make([]string, len(cols))
			}

			combos[c][i] = v.values[(c/repeat)%uLen]
		}

		repeat *= uLen
	}

	// Build hash index of existing rows.
	completeCols := make([]dataset.AnyColumn, len(cols))
	for i, name := range cols {
		completeCols[i], _ = ds.Column(name)
	}

	existingRows := map[string]int{}

	for row := range n {
		key := idKey(completeCols, row)
		existingRows[key] = row
	}

	// Build output indices: existing row or -1 (null-fill).
	outIndices := make([]int, totalCombos)

	outKeys := make([][]string, totalCombos)
	for c := range totalCombos {
		key := strings.Join(combos[c], "\x00")
		if row, ok := existingRows[key]; ok {
			outIndices[c] = row
		} else {
			outIndices[c] = -1
		}

		outKeys[c] = combos[c]
	}

	// Build output columns.
	var (
		outFields []dataset.Field
		outCols   []dataset.AnyColumn
	)

	completeSet := map[string]int{}
	for i, name := range cols {
		completeSet[name] = i
	}

	for i := 0; i < schema.NumFields(); i++ {
		f := schema.Field(i)
		outFields = append(outFields, f)

		if cIdx, isComplete := completeSet[f.Name]; isComplete {
			// For complete columns, use the combo values directly.
			data := make([]string, totalCombos)
			for c := range totalCombos {
				data[c] = outKeys[c][cIdx]
			}
			// Convert back to original type.
			col, _ := ds.Column(f.Name)
			outCols = append(outCols, castStringSlice(col, data, f.Name, totalCombos))
		} else {
			// For non-complete columns, gather by index with null-fill.
			col, _ := ds.Column(f.Name)
			outCols = append(outCols, gatherColumn(col, outIndices, totalCombos, f.Name))
		}
	}

	outSchema := dataset.NewSchema(outFields...)

	return e.FromColumns(outSchema, outCols...)
}

// castStringSlice converts string representations back to the column's original type.
func castStringSlice(original dataset.AnyColumn, data []string, name string, n int) dataset.AnyColumn {
	switch c := original.(type) {
	case *float64Column:
		out := make([]float64, n)

		for i, s := range data {
			v, err := parseFloat(s)
			if err != nil {
				out[i] = math.NaN()
			} else {
				out[i] = v
			}
		}

		return &float64Column{name: name, data: out}
	case *int64Column:
		out := make([]int64, n)

		for i, s := range data {
			v, err := parseInt(s)
			if err != nil {
				out[i] = 0
			} else {
				out[i] = v
			}
		}

		return &int64Column{name: name, data: out, dtype: c.dtype}
	default:
		return &stringColumn{name: name, data: data}
	}
}

func parseFloat(s string) (float64, error) {
	var v float64

	_, err := fmt.Sscanf(s, "%g", &v)

	return v, err
}

func parseInt(s string) (int64, error) {
	var v int64

	_, err := fmt.Sscanf(s, "%d", &v)

	return v, err
}
