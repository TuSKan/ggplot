package arrow

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/TuSKan/ggplot/dataset"

	"github.com/apache/arrow-go/v18/arrow/array"
)

// Join implements the Joiner interface with a hash-join algorithm.
// It supports Inner, Left, Right, Full, Semi, and Anti joins.
func (e *Engine) Join(left, right dataset.Dataset, spec dataset.JoinSpec) (dataset.Dataset, error) {
	if len(spec.LeftCols) == 0 || len(spec.RightCols) == 0 {
		return nil, fmt.Errorf("arrow: Join requires at least one key column")
	}
	if len(spec.LeftCols) != len(spec.RightCols) {
		return nil, fmt.Errorf("arrow: Join key column count mismatch: left=%d, right=%d",
			len(spec.LeftCols), len(spec.RightCols))
	}

	// Validate key columns exist.
	for _, name := range spec.LeftCols {
		if !left.Schema().HasField(name) {
			return nil, fmt.Errorf("arrow: left dataset has no column %q", name)
		}
	}
	for _, name := range spec.RightCols {
		if !right.Schema().HasField(name) {
			return nil, fmt.Errorf("arrow: right dataset has no column %q", name)
		}
	}

	// Build hash index on right key columns.
	rightIndex, err := arrowBuildHashIndex(right, spec.RightCols)
	if err != nil {
		return nil, err
	}

	// Probe left against hash index and collect row pairs.
	leftIndices, rightIndices, err := arrowProbeJoin(left, right, spec, rightIndex)
	if err != nil {
		return nil, err
	}

	// Build output schema and columns.
	return arrowBuildJoinResult(e, left, right, spec, leftIndices, rightIndices)
}

// arrowBuildHashIndex creates map[string][]int for the right dataset's key columns.
func arrowBuildHashIndex(ds dataset.Dataset, cols []string) (map[string][]int, error) {
	n := int(ds.NumRows())
	index := make(map[string][]int, n)
	keyCols := make([]dataset.AnyColumn, len(cols))
	for i, name := range cols {
		col, err := ds.Column(name)
		if err != nil {
			return nil, err
		}
		keyCols[i] = col
	}
	for row := 0; row < n; row++ {
		key := arrowHashKey(keyCols, row)
		index[key] = append(index[key], row)
	}
	return index, nil
}

// arrowHashKey produces a string key for a single row.
func arrowHashKey(cols []dataset.AnyColumn, row int) string {
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

// arrowColValueString extracts string representation of an arrow column value.
func arrowColValueString(col dataset.AnyColumn, row int) string {
	switch c := col.(type) {
	case *arrowFloat64Column:
		return strconv.FormatFloat(c.arr.Value(row), 'g', -1, 64)
	case *arrowInt64Column:
		return strconv.FormatInt(c.arr.Value(row), 10)
	case *arrowStringColumn:
		return c.arr.Value(row)
	case *arrowBoolColumn:
		if c.arr.Value(row) {
			return "T"
		}
		return "F"
	default:
		return fmt.Sprintf("%v", row)
	}
}

// arrowProbeJoin probes left against right hash index and produces row pairs.
func arrowProbeJoin(left, right dataset.Dataset, spec dataset.JoinSpec,
	rightIndex map[string][]int) (leftIdx, rightIdx []int, err error) {

	leftKeyCols := make([]dataset.AnyColumn, len(spec.LeftCols))
	for i, name := range spec.LeftCols {
		col, err := left.Column(name)
		if err != nil {
			return nil, nil, err
		}
		leftKeyCols[i] = col
	}

	nLeft := int(left.NumRows())
	nRight := int(right.NumRows())

	switch spec.Type {
	case dataset.JoinInner:
		for i := 0; i < nLeft; i++ {
			key := arrowHashKey(leftKeyCols, i)
			if matches, ok := rightIndex[key]; ok {
				for _, j := range matches {
					leftIdx = append(leftIdx, i)
					rightIdx = append(rightIdx, j)
				}
			}
		}

	case dataset.JoinLeft:
		for i := 0; i < nLeft; i++ {
			key := arrowHashKey(leftKeyCols, i)
			if matches, ok := rightIndex[key]; ok {
				for _, j := range matches {
					leftIdx = append(leftIdx, i)
					rightIdx = append(rightIdx, j)
				}
			} else {
				leftIdx = append(leftIdx, i)
				rightIdx = append(rightIdx, -1)
			}
		}

	case dataset.JoinRight:
		rightMatched := make([]bool, nRight)
		for i := 0; i < nLeft; i++ {
			key := arrowHashKey(leftKeyCols, i)
			if matches, ok := rightIndex[key]; ok {
				for _, j := range matches {
					leftIdx = append(leftIdx, i)
					rightIdx = append(rightIdx, j)
					rightMatched[j] = true
				}
			}
		}
		for j := 0; j < nRight; j++ {
			if !rightMatched[j] {
				leftIdx = append(leftIdx, -1)
				rightIdx = append(rightIdx, j)
			}
		}

	case dataset.JoinFull:
		rightMatched := make([]bool, nRight)
		for i := 0; i < nLeft; i++ {
			key := arrowHashKey(leftKeyCols, i)
			if matches, ok := rightIndex[key]; ok {
				for _, j := range matches {
					leftIdx = append(leftIdx, i)
					rightIdx = append(rightIdx, j)
					rightMatched[j] = true
				}
			} else {
				leftIdx = append(leftIdx, i)
				rightIdx = append(rightIdx, -1)
			}
		}
		for j := 0; j < nRight; j++ {
			if !rightMatched[j] {
				leftIdx = append(leftIdx, -1)
				rightIdx = append(rightIdx, j)
			}
		}

	case dataset.JoinSemi:
		for i := 0; i < nLeft; i++ {
			key := arrowHashKey(leftKeyCols, i)
			if _, ok := rightIndex[key]; ok {
				leftIdx = append(leftIdx, i)
			}
		}

	case dataset.JoinAnti:
		for i := 0; i < nLeft; i++ {
			key := arrowHashKey(leftKeyCols, i)
			if _, ok := rightIndex[key]; !ok {
				leftIdx = append(leftIdx, i)
			}
		}

	default:
		return nil, nil, fmt.Errorf("arrow: unsupported join type %d", spec.Type)
	}

	return leftIdx, rightIdx, nil
}

// arrowBuildJoinResult constructs the output dataset from row index pairs.
func arrowBuildJoinResult(e *Engine, left, right dataset.Dataset, spec dataset.JoinSpec,
	leftIdx, rightIdx []int) (dataset.Dataset, error) {

	isSemiAnti := spec.Type == dataset.JoinSemi || spec.Type == dataset.JoinAnti
	n := len(leftIdx)

	// Right key columns to exclude from output.
	rightKeySet := make(map[string]bool, len(spec.RightCols))
	for _, name := range spec.RightCols {
		rightKeySet[name] = true
	}

	// Build output schema.
	var fields []dataset.Field
	leftSchema := left.Schema()
	for i := 0; i < leftSchema.NumFields(); i++ {
		fields = append(fields, leftSchema.Field(i))
	}
	if !isSemiAnti {
		rightSchema := right.Schema()
		for i := 0; i < rightSchema.NumFields(); i++ {
			f := rightSchema.Field(i)
			if rightKeySet[f.Name] {
				continue
			}
			finalName := f.Name
			if leftSchema.HasField(f.Name) {
				finalName = f.Name + "_right"
			}
			fields = append(fields, dataset.Field{Name: finalName, Dtype: f.Dtype, Nullable: true})
		}
	}
	outSchema := dataset.NewSchema(fields...)

	// Gather columns.
	var outCols []dataset.AnyColumn

	// Left columns.
	for i := 0; i < leftSchema.NumFields(); i++ {
		f := leftSchema.Field(i)
		col, _ := left.Column(f.Name)
		gathered := e.arrowGatherColumn(col, leftIdx, n, f.Name)
		outCols = append(outCols, gathered)
	}

	// Right columns (skip key columns, skip for semi/anti).
	if !isSemiAnti {
		rightSchema := right.Schema()
		for i := 0; i < rightSchema.NumFields(); i++ {
			f := rightSchema.Field(i)
			if rightKeySet[f.Name] {
				continue
			}
			col, _ := right.Column(f.Name)
			finalName := f.Name
			if leftSchema.HasField(f.Name) {
				finalName = f.Name + "_right"
			}
			gathered := e.arrowGatherColumn(col, rightIdx, n, finalName)
			outCols = append(outCols, gathered)
		}
	}

	return e.FromColumns(outSchema, outCols...)
}

// arrowGatherColumn produces a new Arrow column by gathering rows at the given indices.
// An index of -1 inserts a null via Arrow's null bitmap.
func (e *Engine) arrowGatherColumn(col dataset.AnyColumn, indices []int, n int, name string) dataset.AnyColumn {
	switch c := col.(type) {
	case *arrowFloat64Column:
		b := array.NewFloat64Builder(e.alloc)
		defer b.Release()
		b.Reserve(n)
		for _, idx := range indices {
			if idx < 0 {
				b.AppendNull()
			} else {
				b.Append(c.arr.Value(idx))
			}
		}
		return &arrowFloat64Column{name: name, arr: b.NewFloat64Array()}

	case *arrowInt64Column:
		b := array.NewInt64Builder(e.alloc)
		defer b.Release()
		b.Reserve(n)
		for _, idx := range indices {
			if idx < 0 {
				b.AppendNull()
			} else {
				b.Append(c.arr.Value(idx))
			}
		}
		return &arrowInt64Column{name: name, arr: b.NewInt64Array(), dtype: c.dtype}

	case *arrowStringColumn:
		b := array.NewStringBuilder(e.alloc)
		defer b.Release()
		b.Reserve(n)
		for _, idx := range indices {
			if idx < 0 {
				b.AppendNull()
			} else {
				b.Append(c.arr.Value(idx))
			}
		}
		return &arrowStringColumn{name: name, arr: b.NewStringArray()}

	case *arrowBoolColumn:
		b := array.NewBooleanBuilder(e.alloc)
		defer b.Release()
		b.Reserve(n)
		for _, idx := range indices {
			if idx < 0 {
				b.AppendNull()
			} else {
				b.Append(c.arr.Value(idx))
			}
		}
		return &arrowBoolColumn{name: name, arr: b.NewBooleanArray()}

	default:
		return col
	}
}
