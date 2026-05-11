package memory

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/TuSKan/ggplot/dataset"
)

// Join implements the Joiner interface with a hash-join algorithm.
// It supports Inner, Left, Right, Full, Semi, and Anti joins.
func (e *Engine) Join(left, right dataset.Table, spec dataset.JoinSpec) (dataset.Table, error) {
	if len(spec.LeftCols) == 0 || len(spec.RightCols) == 0 {
		return nil, fmt.Errorf("Join: %w", ErrJoinKeyMismatch)
	}

	if len(spec.LeftCols) != len(spec.RightCols) {
		return nil, fmt.Errorf("memory: Join key column count mismatch: left=%d, right=%d", //nolint:err113 // error contains dynamic context values that vary per call site.
			len(spec.LeftCols), len(spec.RightCols))
	}

	// Validate key columns exist.
	for _, name := range spec.LeftCols {
		if !left.Schema().HasField(name) {
			return nil, fmt.Errorf("left dataset has no column %q: %w", name, ErrJoinKeyMismatch)
		}
	}

	for _, name := range spec.RightCols {
		if !right.Schema().HasField(name) {
			return nil, fmt.Errorf("right dataset has no column %q: %w", name, ErrJoinKeyMismatch)
		}
	}

	// Build hash index on right key columns.
	rightIndex, err := buildHashIndex(right, spec.RightCols)
	if err != nil {
		return nil, err
	}

	// Probe left against hash index and collect row pairs.
	leftIndices, rightIndices, err := probeJoin(left, right, spec, rightIndex)
	if err != nil {
		return nil, err
	}

	// Build output schema and columns.
	return buildJoinResult(e, left, right, spec, leftIndices, rightIndices)
}

// buildHashIndex creates a map from hash key → row indices for the right dataset.
func buildHashIndex(ds dataset.Table, cols []string) (map[string][]int, error) {
	n := int(ds.NumRows())
	index := make(map[string][]int, n)

	keyCols := make([]dataset.AnyColumn, len(cols))
	for i, name := range cols {
		col, err := ds.Column(name)
		if err != nil {
			return nil, fmt.Errorf("memory: %w", err)
		}

		keyCols[i] = col
	}

	for row := range n {
		key := hashKey(keyCols, row)
		index[key] = append(index[key], row)
	}

	return index, nil
}

// hashKey produces a string key for a single row across multiple columns.
func hashKey(cols []dataset.AnyColumn, row int) string {
	if len(cols) == 1 {
		return colValueString(cols[0], row)
	}

	var b strings.Builder

	for i, col := range cols {
		if i > 0 {
			b.WriteByte('\x00') // separator that won't appear in data
		}

		b.WriteString(colValueString(col, row))
	}

	return b.String()
}

// colValueString extracts the string representation of a column value at row.
func colValueString(col dataset.AnyColumn, row int) string {
	switch c := col.(type) {
	case *float64Column:
		return strconv.FormatFloat(c.data[row], 'g', -1, 64)
	case *int64Column:
		return strconv.FormatInt(c.data[row], 10)
	case *stringColumn:
		return c.data[row]
	case *boolColumn:
		if c.data[row] {
			return "T"
		}

		return "F"
	default:
		return strconv.Itoa(row)
	}
}

// probeJoin probes the left dataset against the right hash index and produces
// row index pairs. A value of -1 means "no match" (null-fill).
func probeJoin(left, right dataset.Table, spec dataset.JoinSpec, //nolint:gocognit // probeJoin is a complex pipeline — splitting reduces clarity.
	rightIndex map[string][]int) (leftIdx, rightIdx []int, err error) {
	leftKeyCols := make([]dataset.AnyColumn, len(spec.LeftCols))
	for i, name := range spec.LeftCols {
		col, err := left.Column(name)
		if err != nil {
			return nil, nil, fmt.Errorf("memory: %w", err)
		}

		leftKeyCols[i] = col
	}

	nLeft := int(left.NumRows())
	nRight := int(right.NumRows())

	switch spec.Type {
	case dataset.JoinInner:
		for i := range nLeft {
			key := hashKey(leftKeyCols, i)
			if matches, ok := rightIndex[key]; ok {
				for _, j := range matches {
					leftIdx = append(leftIdx, i)
					rightIdx = append(rightIdx, j)
				}
			}
		}

	case dataset.JoinLeft:
		for i := range nLeft {
			key := hashKey(leftKeyCols, i)
			if matches, ok := rightIndex[key]; ok {
				for _, j := range matches {
					leftIdx = append(leftIdx, i)
					rightIdx = append(rightIdx, j)
				}
			} else {
				leftIdx = append(leftIdx, i)
				rightIdx = append(rightIdx, -1) // null-fill right
			}
		}

	case dataset.JoinRight:
		// Track which right rows were matched.
		rightMatched := make([]bool, nRight)

		for i := range nLeft {
			key := hashKey(leftKeyCols, i)
			if matches, ok := rightIndex[key]; ok {
				for _, j := range matches {
					leftIdx = append(leftIdx, i)
					rightIdx = append(rightIdx, j)
					rightMatched[j] = true
				}
			}
		}
		// Append unmatched right rows.
		for j := range nRight {
			if !rightMatched[j] {
				leftIdx = append(leftIdx, -1) // null-fill left
				rightIdx = append(rightIdx, j)
			}
		}

	case dataset.JoinFull:
		rightMatched := make([]bool, nRight)

		for i := range nLeft {
			key := hashKey(leftKeyCols, i)
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

		for j := range nRight {
			if !rightMatched[j] {
				leftIdx = append(leftIdx, -1)
				rightIdx = append(rightIdx, j)
			}
		}

	case dataset.JoinSemi:
		for i := range nLeft {
			key := hashKey(leftKeyCols, i)
			if _, ok := rightIndex[key]; ok {
				leftIdx = append(leftIdx, i)
			}
		}

	case dataset.JoinAnti:
		for i := range nLeft {
			key := hashKey(leftKeyCols, i)
			if _, ok := rightIndex[key]; !ok {
				leftIdx = append(leftIdx, i)
			}
		}

	default:
		return nil, nil, fmt.Errorf("unsupported join type %d: %w", spec.Type, ErrUnsupportedType)
	}

	return leftIdx, rightIdx, nil
}

// buildJoinResult constructs the output dataset from row index pairs.
func buildJoinResult(e *Engine, left, right dataset.Table, spec dataset.JoinSpec,
	leftIdx, rightIdx []int) (dataset.Table, error) {
	isSemiAnti := spec.Type == dataset.JoinSemi || spec.Type == dataset.JoinAnti
	n := len(leftIdx)

	// Build the set of right key columns to exclude from right output.
	rightKeySet := make(map[string]bool, len(spec.RightCols))
	for _, name := range spec.RightCols {
		rightKeySet[name] = true
	}

	// Build output schema: left columns + non-key right columns.
	var fields []dataset.Field

	leftSchema := left.Schema()
	for i := range leftSchema.NumFields() {
		fields = append(fields, leftSchema.Field(i))
	}

	if !isSemiAnti {
		rightSchema := right.Schema()
		for i := range rightSchema.NumFields() {
			f := rightSchema.Field(i)
			if rightKeySet[f.Name] {
				continue // skip duplicate key columns
			}
			// Handle name collision: suffix with "_right".
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
	for i := range leftSchema.NumFields() {
		f := leftSchema.Field(i)
		col, _ := left.Column(f.Name)
		gathered := gatherColumn(col, leftIdx, n, f.Name)
		outCols = append(outCols, gathered)
	}

	// Right columns (skip key columns, skip for semi/anti).
	if !isSemiAnti {
		rightSchema := right.Schema()
		for i := range rightSchema.NumFields() {
			f := rightSchema.Field(i)
			if rightKeySet[f.Name] {
				continue
			}

			col, _ := right.Column(f.Name)

			finalName := f.Name
			if leftSchema.HasField(f.Name) {
				finalName = f.Name + "_right"
			}

			gathered := gatherColumn(col, rightIdx, n, finalName)
			outCols = append(outCols, gathered)
		}
	}

	return e.FromColumns(outSchema, outCols...)
}

// gatherColumn produces a new column by gathering rows at the given indices.
// An index of -1 inserts a null value (NaN for float64, 0 for int64, "" for string).
func gatherColumn(col dataset.AnyColumn, indices []int, n int, name string) dataset.AnyColumn {
	switch c := col.(type) {
	case *float64Column:
		out := make([]float64, n)

		for i, idx := range indices {
			if idx < 0 {
				out[i] = math.NaN()
			} else {
				out[i] = c.data[idx]
			}
		}

		return &float64Column{name: name, data: out}
	case *int64Column:
		out := make([]int64, n)

		for i, idx := range indices {
			if idx >= 0 {
				out[i] = c.data[idx]
			}
			// idx < 0 → 0 (zero value)
		}

		return &int64Column{name: name, data: out, dtype: c.dtype}
	case *stringColumn:
		out := make([]string, n)

		for i, idx := range indices {
			if idx >= 0 {
				out[i] = c.data[idx]
			}
			// idx < 0 → "" (zero value)
		}

		return &stringColumn{name: name, data: out}
	case *boolColumn:
		out := make([]bool, n)

		for i, idx := range indices {
			if idx >= 0 {
				out[i] = c.data[idx]
			}
		}

		return &boolColumn{name: name, data: out}
	default:
		return col
	}
}
