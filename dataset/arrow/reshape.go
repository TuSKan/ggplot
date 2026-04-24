package arrow

import (
	"fmt"
	"math"
	"strings"

	"github.com/TuSKan/ggplot/dataset"

	"github.com/apache/arrow-go/v18/arrow/array"
)

// PivotLonger reshapes a wide dataset to long format.
func (e *Engine) PivotLonger(ds dataset.Table, spec dataset.PivotLongerSpec) (dataset.Table, error) {
	if len(spec.Cols) == 0 {
		return nil, fmt.Errorf("arrow: PivotLonger requires at least one column to pivot")
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

	pivotSet := make(map[string]bool, nPivot)
	for _, name := range spec.Cols {
		if !schema.HasField(name) {
			return nil, fmt.Errorf("arrow: PivotLonger: column %q not found", name)
		}
		pivotSet[name] = true
	}

	// Validate same type.
	var pivotDType dataset.DType
	for i, name := range spec.Cols {
		f := schema.Field(schema.FieldIndex(name))
		if i == 0 {
			pivotDType = f.Dtype
		} else if f.Dtype != pivotDType {
			return nil, fmt.Errorf("arrow: PivotLonger: column %q has type %s, expected %s",
				name, f.Dtype, pivotDType)
		}
	}

	// Build output.
	var outFields []dataset.Field
	var outCols []dataset.AnyColumn

	// ID columns: repeat each value nPivot times.
	for i := 0; i < schema.NumFields(); i++ {
		f := schema.Field(i)
		if pivotSet[f.Name] {
			continue
		}
		outFields = append(outFields, f)
		col, _ := ds.Column(f.Name)
		outCols = append(outCols, e.arrowRepeatColumn(col, nPivot, outLen, f.Name))
	}

	// Names column.
	namesData := make([]string, outLen)
	for row := 0; row < nRows; row++ {
		for p, name := range spec.Cols {
			namesData[row*nPivot+p] = name
		}
	}
	outFields = append(outFields, dataset.StringCol(spec.NamesTo))
	outCols = append(outCols, e.NewStringColumn(spec.NamesTo, namesData))

	// Values column.
	outFields = append(outFields, dataset.Field{Name: spec.ValuesTo, Dtype: pivotDType})
	valCol := e.arrowGatherPivotValues(ds, spec.Cols, pivotDType, nRows, nPivot, outLen, spec.ValuesTo)
	outCols = append(outCols, valCol)

	outSchema := dataset.NewSchema(outFields...)
	return e.FromColumns(outSchema, outCols...)
}

func (e *Engine) arrowRepeatColumn(col dataset.AnyColumn, times, outLen int, name string) dataset.AnyColumn {
	switch c := col.(type) {
	case *arrowFloat64Column:
		b := array.NewFloat64Builder(e.alloc)
		defer b.Release()
		b.Reserve(outLen)
		for row := 0; row < c.arr.Len(); row++ {
			v := c.arr.Value(row)
			for t := 0; t < times; t++ {
				b.Append(v)
			}
		}
		return &arrowFloat64Column{name: name, arr: b.NewFloat64Array()}
	case *arrowInt64Column:
		b := array.NewInt64Builder(e.alloc)
		defer b.Release()
		b.Reserve(outLen)
		for row := 0; row < c.arr.Len(); row++ {
			v := c.arr.Value(row)
			for t := 0; t < times; t++ {
				b.Append(v)
			}
		}
		return &arrowInt64Column{name: name, arr: b.NewInt64Array(), dtype: c.dtype}
	case *arrowStringColumn:
		b := array.NewStringBuilder(e.alloc)
		defer b.Release()
		b.Reserve(outLen)
		for row := 0; row < c.arr.Len(); row++ {
			v := c.arr.Value(row)
			for t := 0; t < times; t++ {
				b.Append(v)
			}
		}
		return &arrowStringColumn{name: name, arr: b.NewStringArray()}
	case *arrowBoolColumn:
		b := array.NewBooleanBuilder(e.alloc)
		defer b.Release()
		b.Reserve(outLen)
		for row := 0; row < c.arr.Len(); row++ {
			v := c.arr.Value(row)
			for t := 0; t < times; t++ {
				b.Append(v)
			}
		}
		return &arrowBoolColumn{name: name, arr: b.NewBooleanArray()}
	default:
		return col
	}
}

func (e *Engine) arrowGatherPivotValues(ds dataset.Table, cols []string, dtype dataset.DType,
	nRows, nPivot, outLen int, name string) dataset.AnyColumn {
	switch dtype {
	case dataset.DTypeFloat64:
		b := array.NewFloat64Builder(e.alloc)
		defer b.Release()
		b.Reserve(outLen)
		pivotCols := make([]*arrowFloat64Column, len(cols))
		for i, cn := range cols {
			c, _ := ds.Column(cn)
			pivotCols[i] = c.(*arrowFloat64Column)
		}
		for row := 0; row < nRows; row++ {
			for _, pc := range pivotCols {
				b.Append(pc.arr.Value(row))
			}
		}
		return &arrowFloat64Column{name: name, arr: b.NewFloat64Array()}
	case dataset.DTypeInt64:
		b := array.NewInt64Builder(e.alloc)
		defer b.Release()
		b.Reserve(outLen)
		pivotCols := make([]*arrowInt64Column, len(cols))
		for i, cn := range cols {
			c, _ := ds.Column(cn)
			pivotCols[i] = c.(*arrowInt64Column)
		}
		for row := 0; row < nRows; row++ {
			for _, pc := range pivotCols {
				b.Append(pc.arr.Value(row))
			}
		}
		return &arrowInt64Column{name: name, arr: b.NewInt64Array()}
	case dataset.DTypeString:
		b := array.NewStringBuilder(e.alloc)
		defer b.Release()
		b.Reserve(outLen)
		pivotCols := make([]*arrowStringColumn, len(cols))
		for i, cn := range cols {
			c, _ := ds.Column(cn)
			pivotCols[i] = c.(*arrowStringColumn)
		}
		for row := 0; row < nRows; row++ {
			for _, pc := range pivotCols {
				b.Append(pc.arr.Value(row))
			}
		}
		return &arrowStringColumn{name: name, arr: b.NewStringArray()}
	default:
		return e.NewStringColumn(name, make([]string, outLen))
	}
}

// PivotWider reshapes a long dataset to wide format.
func (e *Engine) PivotWider(ds dataset.Table, spec dataset.PivotWiderSpec) (dataset.Table, error) {
	if spec.NamesFrom == "" || spec.ValuesFrom == "" {
		return nil, fmt.Errorf("arrow: PivotWider requires NamesFrom and ValuesFrom")
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
	nameStr, ok := nameCol.(*arrowStringColumn)
	if !ok {
		return nil, fmt.Errorf("arrow: PivotWider: NamesFrom %q must be string column", spec.NamesFrom)
	}

	// Find unique pivot names.
	var pivotNames []string
	seen := map[string]bool{}
	for i := 0; i < nameStr.arr.Len(); i++ {
		v := nameStr.arr.Value(i)
		if !seen[v] {
			pivotNames = append(pivotNames, v)
			seen[v] = true
		}
	}

	// ID columns.
	var idColNames []string
	for i := 0; i < schema.NumFields(); i++ {
		f := schema.Field(i)
		if f.Name == spec.NamesFrom || f.Name == spec.ValuesFrom {
			continue
		}
		idColNames = append(idColNames, f.Name)
	}

	// Build id-key → output row.
	idCols := make([]dataset.AnyColumn, len(idColNames))
	for i, name := range idColNames {
		idCols[i], _ = ds.Column(name)
	}

	idKeyToRow := map[string]int{}
	rowIDKeys := make([]string, nRows)
	for row := 0; row < nRows; row++ {
		key := arrowIDKey(idCols, row)
		rowIDKeys[row] = key
		if _, ok := idKeyToRow[key]; !ok {
			idKeyToRow[key] = len(idKeyToRow)
		}
	}
	outLen := len(idKeyToRow)

	// Output schema.
	var outFields []dataset.Field
	for _, name := range idColNames {
		outFields = append(outFields, schema.Field(schema.FieldIndex(name)))
	}
	valDType := valCol.DType()
	for _, pn := range pivotNames {
		outFields = append(outFields, dataset.Field{Name: pn, Dtype: valDType})
	}
	outSchema := dataset.NewSchema(outFields...)

	// Gather id columns.
	firstRowMap := map[int]int{}
	for row := 0; row < nRows; row++ {
		idx := idKeyToRow[rowIDKeys[row]]
		if _, exists := firstRowMap[idx]; !exists {
			firstRowMap[idx] = row
		}
	}
	idIndices := make([]int, outLen)
	for idx, row := range firstRowMap {
		idIndices[idx] = row
	}

	var outCols []dataset.AnyColumn
	for _, name := range idColNames {
		col, _ := ds.Column(name)
		outCols = append(outCols, e.arrowGatherColumn(col, idIndices, outLen, name))
	}

	// Build pivot-value columns.
	pivotNameToIdx := map[string]int{}
	for i, pn := range pivotNames {
		pivotNameToIdx[pn] = i
	}

	switch c := valCol.(type) {
	case *arrowFloat64Column:
		pivotData := make([][]float64, len(pivotNames))
		pivotValid := make([][]bool, len(pivotNames))
		for i := range pivotData {
			pivotData[i] = make([]float64, outLen)
			pivotValid[i] = make([]bool, outLen)
		}
		for row := 0; row < nRows; row++ {
			outRow := idKeyToRow[rowIDKeys[row]]
			pIdx := pivotNameToIdx[nameStr.arr.Value(row)]
			pivotData[pIdx][outRow] = c.arr.Value(row)
			pivotValid[pIdx][outRow] = true
		}
		for i, pn := range pivotNames {
			b := array.NewFloat64Builder(e.alloc)
			b.Reserve(outLen)
			for j := 0; j < outLen; j++ {
				if pivotValid[i][j] {
					b.Append(pivotData[i][j])
				} else {
					b.AppendNull()
				}
			}
			outCols = append(outCols, &arrowFloat64Column{name: pn, arr: b.NewFloat64Array()})
			b.Release()
		}
	case *arrowInt64Column:
		pivotData := make([][]int64, len(pivotNames))
		pivotValid := make([][]bool, len(pivotNames))
		for i := range pivotData {
			pivotData[i] = make([]int64, outLen)
			pivotValid[i] = make([]bool, outLen)
		}
		for row := 0; row < nRows; row++ {
			outRow := idKeyToRow[rowIDKeys[row]]
			pIdx := pivotNameToIdx[nameStr.arr.Value(row)]
			pivotData[pIdx][outRow] = c.arr.Value(row)
			pivotValid[pIdx][outRow] = true
		}
		for i, pn := range pivotNames {
			b := array.NewInt64Builder(e.alloc)
			b.Reserve(outLen)
			for j := 0; j < outLen; j++ {
				if pivotValid[i][j] {
					b.Append(pivotData[i][j])
				} else {
					b.AppendNull()
				}
			}
			outCols = append(outCols, &arrowInt64Column{name: pn, arr: b.NewInt64Array(), dtype: c.dtype})
			b.Release()
		}
	case *arrowStringColumn:
		pivotData := make([][]string, len(pivotNames))
		pivotValid := make([][]bool, len(pivotNames))
		for i := range pivotData {
			pivotData[i] = make([]string, outLen)
			pivotValid[i] = make([]bool, outLen)
		}
		for row := 0; row < nRows; row++ {
			outRow := idKeyToRow[rowIDKeys[row]]
			pIdx := pivotNameToIdx[nameStr.arr.Value(row)]
			pivotData[pIdx][outRow] = c.arr.Value(row)
			pivotValid[pIdx][outRow] = true
		}
		for i, pn := range pivotNames {
			b := array.NewStringBuilder(e.alloc)
			b.Reserve(outLen)
			for j := 0; j < outLen; j++ {
				if pivotValid[i][j] {
					b.Append(pivotData[i][j])
				} else {
					b.AppendNull()
				}
			}
			outCols = append(outCols, &arrowStringColumn{name: pn, arr: b.NewStringArray()})
			b.Release()
		}
	default:
		return nil, fmt.Errorf("arrow: PivotWider: unsupported value type %s", valDType)
	}

	return e.FromColumns(outSchema, outCols...)
}

func arrowIDKey(cols []dataset.AnyColumn, row int) string {
	if len(cols) == 0 {
		return ""
	}
	if len(cols) == 1 {
		return arrowColValueString(cols[0], row)
	}
	var b strings.Builder
	for i, col := range cols {
		if i > 0 {
			b.WriteByte('\x00')
		}
		b.WriteString(arrowColValueString(col, row))
	}
	return b.String()
}

// Separate splits a string column by a delimiter into multiple columns.
func (e *Engine) Separate(ds dataset.Table, col string, into []string, sep string) (dataset.Table, error) {
	srcCol, err := ds.Column(col)
	if err != nil {
		return nil, err
	}
	sc, ok := srcCol.(*arrowStringColumn)
	if !ok {
		return nil, fmt.Errorf("arrow: Separate: column %q must be string", col)
	}

	n := int(ds.NumRows())
	nParts := len(into)
	partData := make([][]string, nParts)
	for i := range partData {
		partData[i] = make([]string, n)
	}

	for row := 0; row < n; row++ {
		parts := strings.SplitN(sc.arr.Value(row), sep, nParts)
		for p := 0; p < nParts; p++ {
			if p < len(parts) {
				partData[p][row] = parts[p]
			}
		}
	}

	schema := ds.Schema()
	var outFields []dataset.Field
	var outCols []dataset.AnyColumn

	for i := 0; i < schema.NumFields(); i++ {
		f := schema.Field(i)
		if f.Name == col {
			for p, name := range into {
				outFields = append(outFields, dataset.StringCol(name))
				outCols = append(outCols, e.NewStringColumn(name, partData[p]))
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

	srcCols := make([]*arrowStringColumn, len(from))
	for i, name := range from {
		c, err := ds.Column(name)
		if err != nil {
			return nil, err
		}
		sc, ok := c.(*arrowStringColumn)
		if !ok {
			return nil, fmt.Errorf("arrow: Concatenate: column %q must be string", name)
		}
		srcCols[i] = sc
	}

	data := make([]string, n)
	parts := make([]string, len(from))
	for row := 0; row < n; row++ {
		for i, sc := range srcCols {
			parts[i] = sc.arr.Value(row)
		}
		data[row] = strings.Join(parts, sep)
	}

	fromSet := make(map[string]bool, len(from))
	for _, name := range from {
		fromSet[name] = true
	}

	var outFields []dataset.Field
	var outCols []dataset.AnyColumn
	added := false

	for i := 0; i < schema.NumFields(); i++ {
		f := schema.Field(i)
		if fromSet[f.Name] {
			if !added {
				outFields = append(outFields, dataset.StringCol(col))
				outCols = append(outCols, e.NewStringColumn(col, data))
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
		outCols = append(outCols, e.NewStringColumn(col, data))
	}

	outSchema := dataset.NewSchema(outFields...)
	return e.FromColumns(outSchema, outCols...)
}

// Complete generates all combinations of the specified columns' unique values.
func (e *Engine) Complete(ds dataset.Table, cols ...string) (dataset.Table, error) {
	if len(cols) == 0 {
		return ds, nil
	}

	schema := ds.Schema()
	n := int(ds.NumRows())

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
		for row := 0; row < n; row++ {
			v := arrowColValueString(col, row)
			if !seen[v] {
				vals = append(vals, v)
				seen[v] = true
			}
		}
		uniques[i] = colUniques{name: name, values: vals}
	}

	totalCombos := 1
	for _, u := range uniques {
		totalCombos *= len(u.values)
	}

	combos := make([][]string, totalCombos)
	repeat := 1
	for i := len(uniques) - 1; i >= 0; i-- {
		uLen := len(uniques[i].values)
		for c := 0; c < totalCombos; c++ {
			if combos[c] == nil {
				combos[c] = make([]string, len(cols))
			}
			combos[c][i] = uniques[i].values[(c/repeat)%uLen]
		}
		repeat *= uLen
	}

	completeCols := make([]dataset.AnyColumn, len(cols))
	for i, name := range cols {
		completeCols[i], _ = ds.Column(name)
	}
	existingRows := map[string]int{}
	for row := 0; row < n; row++ {
		key := arrowIDKey(completeCols, row)
		existingRows[key] = row
	}

	outIndices := make([]int, totalCombos)
	outKeys := make([][]string, totalCombos)
	for c := 0; c < totalCombos; c++ {
		key := strings.Join(combos[c], "\x00")
		if row, ok := existingRows[key]; ok {
			outIndices[c] = row
		} else {
			outIndices[c] = -1
		}
		outKeys[c] = combos[c]
	}

	var outFields []dataset.Field
	var outCols []dataset.AnyColumn
	completeSet := map[string]int{}
	for i, name := range cols {
		completeSet[name] = i
	}

	for i := 0; i < schema.NumFields(); i++ {
		f := schema.Field(i)
		outFields = append(outFields, f)

		if cIdx, isComplete := completeSet[f.Name]; isComplete {
			data := make([]string, totalCombos)
			for c := 0; c < totalCombos; c++ {
				data[c] = outKeys[c][cIdx]
			}
			col, _ := ds.Column(f.Name)
			outCols = append(outCols, e.arrowCastStringSlice(col, data, f.Name, totalCombos))
		} else {
			col, _ := ds.Column(f.Name)
			outCols = append(outCols, e.arrowGatherColumn(col, outIndices, totalCombos, f.Name))
		}
	}

	outSchema := dataset.NewSchema(outFields...)
	return e.FromColumns(outSchema, outCols...)
}

func (e *Engine) arrowCastStringSlice(original dataset.AnyColumn, data []string, name string, n int) dataset.AnyColumn {
	switch c := original.(type) {
	case *arrowFloat64Column:
		b := array.NewFloat64Builder(e.alloc)
		defer b.Release()
		b.Reserve(n)
		for _, s := range data {
			var v float64
			_, err := fmt.Sscanf(s, "%g", &v)
			if err != nil {
				b.Append(math.NaN())
			} else {
				b.Append(v)
			}
		}
		return &arrowFloat64Column{name: name, arr: b.NewFloat64Array()}
	case *arrowInt64Column:
		b := array.NewInt64Builder(e.alloc)
		defer b.Release()
		b.Reserve(n)
		for _, s := range data {
			var v int64
			_, err := fmt.Sscanf(s, "%d", &v)
			if err != nil {
				b.AppendNull()
			} else {
				b.Append(v)
			}
		}
		return &arrowInt64Column{name: name, arr: b.NewInt64Array(), dtype: c.dtype}
	default:
		return e.NewStringColumn(name, data)
	}
}
