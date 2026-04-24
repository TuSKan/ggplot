package dataset

import (
	"fmt"
	"strings"
)

// Frame is the fluent API for data manipulation. All verbs return a new Frame
// (immutable chain). Every operation delegates to the dataset's engine via
// sub-interfaces — the Frame never touches raw data directly.
//
// Usage:
//
//	result := dataset.From(ds).
//	    Select("x", "y").
//	    Filter(dataset.Gt("x", 0)).
//	    Arrange("x").
//	    Collect()
type Dataset struct {
	tbl Table
	err error
}

// From wraps a Table in a Dataset for fluent verb chaining.
func From(ds Table) Dataset {
	return Dataset{tbl: ds}
}

// NewDataset creates a Dataset from an engine and columns.
// The schema is inferred from the columns' names and types.
func NewDataset(eng Engine, cols ...AnyColumn) (Dataset, error) {
	factory, ok := eng.(ColumnFactory)
	if !ok {
		return Dataset{}, fmt.Errorf("engine %q does not support ColumnFactory", eng.Name())
	}
	fields := make([]Field, len(cols))
	for i, c := range cols {
		fields[i] = Field{Name: c.Name(), Dtype: c.DType()}
	}
	schema := NewSchema(fields...)
	tbl, err := factory.FromColumns(schema, cols...)
	if err != nil {
		return Dataset{}, err
	}
	return Dataset{tbl: tbl}, nil
}

// ReplaceColumn replaces a named column in a Dataset with new float64 values.
// All other columns are preserved. Used for discrete-to-numeric remapping.
func ReplaceColumn(ds Dataset, name string, values []float64) (Dataset, error) {
	eng := GetEngine(ds.Table())
	if eng == nil {
		return Dataset{}, fmt.Errorf("ReplaceColumn: no engine")
	}
	factory, ok := eng.(ColumnFactory)
	if !ok {
		return Dataset{}, fmt.Errorf("ReplaceColumn: engine %q does not support ColumnFactory", eng.Name())
	}

	newCol := factory.NewFloat64Column(name, values)
	n := int(ds.NumCols())
	cols := make([]AnyColumn, 0, n)
	for i := 0; i < n; i++ {
		f := ds.Schema().Field(i)
		if f.Name == name {
			cols = append(cols, newCol)
		} else {
			c, err := ds.Column(f.Name)
			if err != nil {
				return Dataset{}, err
			}
			cols = append(cols, c)
		}
	}
	return NewDataset(eng, cols...)
}

// Err returns the first error encountered in the chain, or nil.
func (f Dataset) Err() error { return f.err }

// Table returns the underlying Table, or nil if an error occurred.
func (f Dataset) Table() Table { return f.tbl }

// Convenience forwarding methods — allow Dataset to be used where Table is expected.
func (f Dataset) Column(name string) (AnyColumn, error) { return f.tbl.Column(name) }
func (f Dataset) NumRows() int64                        { return f.tbl.NumRows() }
func (f Dataset) NumCols() int64                        { return f.tbl.NumCols() }
func (f Dataset) Schema() *Schema                       { return f.tbl.Schema() }

// Collect materializes the frame's pipeline and returns the Dataset and error.
func (f Dataset) Collect() (Table, error) {
	return f.tbl, f.err
}

// withError returns a Frame with an error set; short-circuits all further verbs.
func (f Dataset) withError(err error) Dataset {
	if f.err != nil {
		return f // keep the first error
	}
	return Dataset{tbl: f.tbl, err: err}
}

// requireEngine extracts the engine and returns it.
func (f Dataset) requireEngine() (Engine, Dataset) {
	if f.err != nil {
		return nil, f
	}
	eng := GetEngine(f.tbl)
	if eng == nil {
		return nil, f.withError(fmt.Errorf("dataset: Frame requires a dataset with an engine"))
	}
	return eng, f
}

// requireSelector extracts the Selector sub-interface from the engine.
func (f Dataset) requireSelector(eng Engine) (Selector, ColumnFactory, error) {
	sel, ok := eng.(Selector)
	if !ok {
		return nil, nil, fmt.Errorf("engine %q does not support Selector", eng.Name())
	}
	factory, ok := eng.(ColumnFactory)
	if !ok {
		return nil, nil, fmt.Errorf("engine %q does not support ColumnFactory", eng.Name())
	}
	return sel, factory, nil
}

// --- Selection ---

// Select keeps only the named columns, in the order specified.
func (f Dataset) Select(cols ...string) Dataset {
	eng, fr := f.requireEngine()
	if fr.err != nil {
		return fr
	}
	factory, ok := eng.(ColumnFactory)
	if !ok {
		return f.withError(fmt.Errorf("engine %q does not support ColumnFactory", eng.Name()))
	}

	fields := make([]Field, 0, len(cols))
	columns := make([]AnyColumn, 0, len(cols))
	for _, name := range cols {
		idx := f.tbl.Schema().FieldIndex(name)
		if idx < 0 {
			return f.withError(&ErrColumnNotFound{Name: name})
		}
		fields = append(fields, f.tbl.Schema().Field(idx))
		col, err := f.tbl.Column(name)
		if err != nil {
			return f.withError(err)
		}
		columns = append(columns, col)
	}

	schema := NewSchema(fields...)
	ds, err := factory.FromColumns(schema, columns...)
	if err != nil {
		return f.withError(err)
	}
	return Dataset{tbl: ds}
}

// Rename renames a column.
func (f Dataset) Rename(oldName, newName string) Dataset {
	eng, fr := f.requireEngine()
	if fr.err != nil {
		return fr
	}
	factory, ok := eng.(ColumnFactory)
	if !ok {
		return f.withError(fmt.Errorf("engine %q does not support ColumnFactory", eng.Name()))
	}

	schema := f.tbl.Schema()
	fields := make([]Field, schema.NumFields())
	columns := make([]AnyColumn, schema.NumFields())
	for i := 0; i < schema.NumFields(); i++ {
		field := schema.Field(i)
		col, err := f.tbl.Column(field.Name)
		if err != nil {
			return f.withError(err)
		}
		if field.Name == oldName {
			field.Name = newName
			col = renameColumn(col, newName)
		}
		fields[i] = field
		columns[i] = col
	}

	newSchema := NewSchema(fields...)
	ds, err := factory.FromColumns(newSchema, columns...)
	if err != nil {
		return f.withError(err)
	}
	return Dataset{tbl: ds}
}

// --- Filtering ---

// Filter keeps rows where the Masker evaluates to true.
func (f Dataset) Filter(mask Masker) Dataset {
	eng, fr := f.requireEngine()
	if fr.err != nil {
		return fr
	}

	// Try engine-native Filterer first
	if filterer, ok := eng.(Filterer); ok {
		ds, err := filterer.Filter(f.tbl, mask)
		if err != nil {
			return f.withError(err)
		}
		return Dataset{tbl: ds}
	}

	// Via Selector: mask → indices → Take
	sel, factory, err := f.requireSelector(eng)
	if err != nil {
		return f.withError(err)
	}

	bools, err := mask.Mask(f.tbl)
	if err != nil {
		return f.withError(err)
	}

	indices := sel.FilterIndices(bools)
	ds, err := applySelect(sel, factory, f.tbl, indices)
	if err != nil {
		return f.withError(err)
	}
	return Dataset{tbl: ds}
}

// --- Sorting ---

// Arrange sorts the dataset by the named column (ascending).
// Engine's Selector.SortIndices computes the permutation; Selector.Take applies it.
func (f Dataset) Arrange(cols ...string) Dataset {
	eng, fr := f.requireEngine()
	if fr.err != nil {
		return fr
	}
	sel, factory, err := f.requireSelector(eng)
	if err != nil {
		return f.withError(err)
	}

	if len(cols) == 0 {
		return f
	}
	col, err := f.tbl.Column(cols[0])
	if err != nil {
		return f.withError(err)
	}
	indices, err := sel.SortIndices(col)
	if err != nil {
		return f.withError(err)
	}

	ds, err := applySelect(sel, factory, f.tbl, indices)
	if err != nil {
		return f.withError(err)
	}
	return Dataset{tbl: ds}
}

// --- Row Slicing ---

// Head returns the first n rows.
func (f Dataset) Head(n int) Dataset {
	if f.err != nil {
		return f
	}
	if n >= int(f.tbl.NumRows()) {
		return f
	}
	return f.Slice(0, n)
}

// Tail returns the last n rows.
func (f Dataset) Tail(n int) Dataset {
	if f.err != nil {
		return f
	}
	if n >= int(f.tbl.NumRows()) {
		return f
	}
	return f.Slice(int(f.tbl.NumRows())-n, int(f.tbl.NumRows()))
}

// Slice returns rows in the range [start, end).
// Engine's Selector.SliceColumn handles this — for Arrow, zero-copy via array.NewSlice.
func (f Dataset) Slice(start, end int) Dataset {
	eng, fr := f.requireEngine()
	if fr.err != nil {
		return fr
	}
	sel, factory, err := f.requireSelector(eng)
	if err != nil {
		return f.withError(err)
	}

	if start < 0 {
		start = 0
	}
	if end > int(f.tbl.NumRows()) {
		end = int(f.tbl.NumRows())
	}
	if start >= end {
		return f.withError(fmt.Errorf("dataset: Slice start (%d) >= end (%d)", start, end))
	}

	ds, err := applySlice(sel, factory, f.tbl, start, end)
	if err != nil {
		return f.withError(err)
	}
	return Dataset{tbl: ds}
}

// --- Distinct ---

// Distinct removes duplicate rows based on the specified columns.
// If no columns are specified, all columns are used.
func (f Dataset) Distinct(cols ...string) Dataset {
	eng, fr := f.requireEngine()
	if fr.err != nil {
		return fr
	}
	sel, factory, err := f.requireSelector(eng)
	if err != nil {
		return f.withError(err)
	}

	if len(cols) == 0 {
		cols = Names(f.tbl)
	}

	// Build unique row indices
	seen := make(map[string]struct{})
	var indices []int
	for row := 0; row < int(f.tbl.NumRows()); row++ {
		key := rowKey(f.tbl, cols, row)
		if _, exists := seen[key]; !exists {
			seen[key] = struct{}{}
			indices = append(indices, row)
		}
	}

	ds, err := applySelect(sel, factory, f.tbl, indices)
	if err != nil {
		return f.withError(err)
	}
	return Dataset{tbl: ds}
}

// --- Joins ---

func (f Dataset) LeftJoin(other Table, spec JoinSpec) Dataset {
	spec.Type = JoinLeft
	return f.join(other, spec)
}

func (f Dataset) InnerJoin(other Table, spec JoinSpec) Dataset {
	spec.Type = JoinInner
	return f.join(other, spec)
}

func (f Dataset) RightJoin(other Table, spec JoinSpec) Dataset {
	spec.Type = JoinRight
	return f.join(other, spec)
}

func (f Dataset) FullJoin(other Table, spec JoinSpec) Dataset {
	spec.Type = JoinFull
	return f.join(other, spec)
}

func (f Dataset) SemiJoin(other Table, spec JoinSpec) Dataset {
	spec.Type = JoinSemi
	return f.join(other, spec)
}

func (f Dataset) AntiJoin(other Table, spec JoinSpec) Dataset {
	spec.Type = JoinAnti
	return f.join(other, spec)
}

func (f Dataset) join(other Table, spec JoinSpec) Dataset {
	eng, fr := f.requireEngine()
	if fr.err != nil {
		return fr
	}
	joiner, ok := eng.(Joiner)
	if !ok {
		return f.withError(fmt.Errorf("engine %q does not support Joiner", eng.Name()))
	}
	ds, err := joiner.Join(f.tbl, other, spec)
	if err != nil {
		return f.withError(err)
	}
	return Dataset{tbl: ds}
}

// --- Reshape ---

func (f Dataset) PivotLonger(spec PivotLongerSpec) Dataset {
	eng, fr := f.requireEngine()
	if fr.err != nil {
		return fr
	}
	reshaper, ok := eng.(Reshaper)
	if !ok {
		return f.withError(fmt.Errorf("engine %q does not support Reshaper", eng.Name()))
	}
	ds, err := reshaper.PivotLonger(f.tbl, spec)
	if err != nil {
		return f.withError(err)
	}
	return Dataset{tbl: ds}
}

func (f Dataset) PivotWider(spec PivotWiderSpec) Dataset {
	eng, fr := f.requireEngine()
	if fr.err != nil {
		return fr
	}
	reshaper, ok := eng.(Reshaper)
	if !ok {
		return f.withError(fmt.Errorf("engine %q does not support Reshaper", eng.Name()))
	}
	ds, err := reshaper.PivotWider(f.tbl, spec)
	if err != nil {
		return f.withError(err)
	}
	return Dataset{tbl: ds}
}

func (f Dataset) Separate(col string, into []string, sep string) Dataset {
	eng, fr := f.requireEngine()
	if fr.err != nil {
		return fr
	}
	reshaper, ok := eng.(Reshaper)
	if !ok {
		return f.withError(fmt.Errorf("engine %q does not support Reshaper", eng.Name()))
	}
	ds, err := reshaper.Separate(f.tbl, col, into, sep)
	if err != nil {
		return f.withError(err)
	}
	return Dataset{tbl: ds}
}

// --- Fill / DropNA ---

func (f Dataset) Fill(col string, dir FillDirection) Dataset {
	eng, fr := f.requireEngine()
	if fr.err != nil {
		return fr
	}
	filler, ok := eng.(Filler)
	if !ok {
		return f.withError(fmt.Errorf("engine %q does not support Filler", eng.Name()))
	}
	factory, ok2 := eng.(ColumnFactory)
	if !ok2 {
		return f.withError(fmt.Errorf("engine %q does not support ColumnFactory", eng.Name()))
	}

	c, err := f.tbl.Column(col)
	if err != nil {
		return f.withError(err)
	}
	filled, err := filler.Fill(c, dir)
	if err != nil {
		return f.withError(err)
	}
	return f.replaceColumn(factory, col, filled)
}

func (f Dataset) DropNA(cols ...string) Dataset {
	eng, fr := f.requireEngine()
	if fr.err != nil {
		return fr
	}
	filler, ok := eng.(Filler)
	if !ok {
		return f.withError(fmt.Errorf("engine %q does not support Filler", eng.Name()))
	}
	ds, err := filler.DropNA(f.tbl, cols...)
	if err != nil {
		return f.withError(err)
	}
	return Dataset{tbl: ds}
}

// --- Composing ---

func (f Dataset) Stack(others ...Table) Dataset {
	eng, fr := f.requireEngine()
	if fr.err != nil {
		return fr
	}
	composer, ok := eng.(Composer)
	if !ok {
		return f.withError(fmt.Errorf("engine %q does not support Composer", eng.Name()))
	}
	all := append([]Table{f.tbl}, others...)
	ds, err := composer.Stack(all...)
	if err != nil {
		return f.withError(err)
	}
	return Dataset{tbl: ds}
}

func (f Dataset) Combine(others ...Table) Dataset {
	eng, fr := f.requireEngine()
	if fr.err != nil {
		return fr
	}
	composer, ok := eng.(Composer)
	if !ok {
		return f.withError(fmt.Errorf("engine %q does not support Composer", eng.Name()))
	}
	all := append([]Table{f.tbl}, others...)
	ds, err := composer.Combine(all...)
	if err != nil {
		return f.withError(err)
	}
	return Dataset{tbl: ds}
}

// --- Internal ---

func (f Dataset) replaceColumn(factory ColumnFactory, name string, newCol AnyColumn) Dataset {
	schema := f.tbl.Schema()
	columns := make([]AnyColumn, schema.NumFields())
	for i := 0; i < schema.NumFields(); i++ {
		field := schema.Field(i)
		if field.Name == name {
			columns[i] = newCol
		} else {
			col, err := f.tbl.Column(field.Name)
			if err != nil {
				return f.withError(err)
			}
			columns[i] = col
		}
	}
	ds, err := factory.FromColumns(schema, columns...)
	if err != nil {
		return f.withError(err)
	}
	return Dataset{tbl: ds}
}

// --- GroupBy + Summarize ---

// AggSpec describes a single aggregation to apply in Summarize.
type AggSpec struct {
	OutputName string  // name of the result column
	InputName  string  // name of the source column
	Fn         AggFunc // which aggregation to apply
}

// AggFunc identifies an aggregation function.
type AggFunc int

const (
	AggSum AggFunc = iota
	AggMean
	AggMin
	AggMax
	AggCount
	AggMedian
	AggVariance
)

// Agg helpers for building AggSpecs.
func Sum(out, in string) AggSpec    { return AggSpec{OutputName: out, InputName: in, Fn: AggSum} }
func Mean(out, in string) AggSpec   { return AggSpec{OutputName: out, InputName: in, Fn: AggMean} }
func Min(out, in string) AggSpec    { return AggSpec{OutputName: out, InputName: in, Fn: AggMin} }
func Max(out, in string) AggSpec    { return AggSpec{OutputName: out, InputName: in, Fn: AggMax} }
func Count(out, in string) AggSpec  { return AggSpec{OutputName: out, InputName: in, Fn: AggCount} }
func Median(out, in string) AggSpec { return AggSpec{OutputName: out, InputName: in, Fn: AggMedian} }
func Variance(out, in string) AggSpec {
	return AggSpec{OutputName: out, InputName: in, Fn: AggVariance}
}

// GroupedFrame holds a Frame with group-by columns set.
type GroupedFrame struct {
	frame     Dataset
	groupCols []string
}

// GroupBy specifies columns to group by. Returns a GroupedFrame for Summarize.
func (f Dataset) GroupBy(cols ...string) GroupedFrame {
	return GroupedFrame{frame: f, groupCols: cols}
}

// Summarize applies aggregations per group using the engine's Aggregator.
// All computation is delegated to the engine — the Frame only orchestrates grouping.
func (gf GroupedFrame) Summarize(specs ...AggSpec) Dataset {
	f := gf.frame
	eng, fr := f.requireEngine()
	if fr.err != nil {
		return fr
	}
	agg, ok := eng.(Aggregator)
	if !ok {
		return f.withError(fmt.Errorf("engine %q does not support Aggregator", eng.Name()))
	}
	sel, factory, err := f.requireSelector(eng)
	if err != nil {
		return f.withError(err)
	}

	// Build group index: key → row indices
	type group struct {
		key     string
		indices []int
	}
	seen := make(map[string]int) // key → index in groups
	var groups []group

	for row := 0; row < int(f.tbl.NumRows()); row++ {
		key := rowKey(f.tbl, gf.groupCols, row)
		if idx, exists := seen[key]; exists {
			groups[idx].indices = append(groups[idx].indices, row)
		} else {
			seen[key] = len(groups)
			groups = append(groups, group{key: key, indices: []int{row}})
		}
	}

	nGroups := len(groups)

	// Build output fielTable: group columns + agg output columns
	var outFields []Field
	for _, name := range gf.groupCols {
		idx := f.tbl.Schema().FieldIndex(name)
		if idx < 0 {
			return f.withError(&ErrColumnNotFound{Name: name})
		}
		outFields = append(outFields, f.tbl.Schema().Field(idx))
	}
	for _, spec := range specs {
		dtype := resolveAggDType(spec.Fn, f.tbl, spec.InputName)
		outFields = append(outFields, Field{Name: spec.OutputName, Dtype: dtype})
	}
	outSchema := NewSchema(outFields...)

	// Compute per-group results
	outCols := make([]AnyColumn, len(outFields))

	// Group-key columns: take the first row of each group
	firstIndices := make([]int, nGroups)
	for i, g := range groups {
		firstIndices[i] = g.indices[0]
	}
	for ci, name := range gf.groupCols {
		col, err := f.tbl.Column(name)
		if err != nil {
			return f.withError(err)
		}
		taken, err := sel.Select(col, firstIndices)
		if err != nil {
			return f.withError(err)
		}
		outCols[ci] = taken
	}

	// Agg columns: slice each group, aggregate
	for si, spec := range specs {
		col, err := f.tbl.Column(spec.InputName)
		if err != nil {
			return f.withError(err)
		}

		// Collect per-group agg results
		aggResults := make([]AnyColumn, nGroups)
		for gi, g := range groups {
			groupCol, err := sel.Select(col, g.indices)
			if err != nil {
				return f.withError(err)
			}
			result, err := dispatchAgg(agg, spec.Fn, groupCol)
			if err != nil {
				return f.withError(err)
			}
			aggResults[gi] = result
		}

		// Merge single-element agg results into one column
		merged, err := mergeAggResults(factory, spec.OutputName, aggResults)
		if err != nil {
			return f.withError(err)
		}
		outCols[len(gf.groupCols)+si] = merged
	}

	ds, err := factory.FromColumns(outSchema, outCols...)
	if err != nil {
		return f.withError(err)
	}
	return Dataset{tbl: ds}
}

// dispatchAgg calls the appropriate Aggregator method.
func dispatchAgg(agg Aggregator, fn AggFunc, col AnyColumn) (AnyColumn, error) {
	switch fn {
	case AggSum:
		return agg.Sum(col)
	case AggMean:
		return agg.Mean(col)
	case AggMin:
		min, _, err := agg.MinMax(col)
		return min, err
	case AggMax:
		_, max, err := agg.MinMax(col)
		return max, err
	case AggCount:
		return agg.Count(col)
	case AggMedian:
		return agg.Median(col)
	case AggVariance:
		return agg.Variance(col)
	default:
		return nil, fmt.Errorf("dataset: unknown AggFunc %d", fn)
	}
}

// resolveAggDType determines the output DType for an aggregation.
func resolveAggDType(fn AggFunc, ds Table, colName string) DType {
	col, err := ds.Column(colName)
	if err != nil {
		return DTypeFloat64
	}
	switch fn {
	case AggSum:
		return col.DType() // preserves type
	case AggMean, AggMedian, AggVariance:
		return DTypeFloat64 // always float64
	case AggMin, AggMax:
		return col.DType() // preserves type
	case AggCount:
		return DTypeInt64 // always int64
	default:
		return DTypeFloat64
	}
}

// mergeAggResults combines N single-element AnyColumns into one N-element column.
func mergeAggResults(factory ColumnFactory, name string, results []AnyColumn) (AnyColumn, error) {
	if len(results) == 0 {
		return nil, fmt.Errorf("dataset: no agg results to merge")
	}
	n := len(results)
	switch results[0].DType() {
	case DTypeFloat64:
		vals := make([]float64, n)
		for i, r := range results {
			vals[i] = r.(Column[float64]).Values()[0]
		}
		return factory.NewFloat64Column(name, vals), nil
	case DTypeInt64:
		vals := make([]int64, n)
		for i, r := range results {
			vals[i] = r.(Column[int64]).Values()[0]
		}
		return factory.NewInt64Column(name, vals), nil
	case DTypeString:
		vals := make([]string, n)
		for i, r := range results {
			vals[i] = r.(Column[string]).Values()[0]
		}
		return factory.NewStringColumn(name, vals), nil
	case DTypeTimestamp:
		vals := make([]int64, n)
		for i, r := range results {
			vals[i] = r.(Column[int64]).Values()[0]
		}
		return factory.NewTimestampColumn(name, vals), nil
	default:
		return nil, fmt.Errorf("dataset: unsupported agg result DType %s", results[0].DType())
	}
}

// --- Mutate ---

// MutateFunc describes a column transformation for Mutate.
type MutateFunc interface {
	// Apply produces a new column from the dataset.
	Apply(ds Table) (AnyColumn, error)
}

// Mutate appends or replaces a column using a MutateFunc.
func (f Dataset) Mutate(name string, fn MutateFunc) Dataset {
	eng, fr := f.requireEngine()
	if fr.err != nil {
		return fr
	}
	factory, ok := eng.(ColumnFactory)
	if !ok {
		return f.withError(fmt.Errorf("engine %q does not support ColumnFactory", eng.Name()))
	}

	newCol, err := fn.Apply(f.tbl)
	if err != nil {
		return f.withError(err)
	}

	// Replace existing or append new
	schema := f.tbl.Schema()
	if schema.HasField(name) {
		return f.replaceColumn(factory, name, newCol)
	}

	// Append: new schema + new column
	fields := schema.Fields()
	fields = append(fields, Field{Name: name, Dtype: newCol.DType()})
	newSchema := NewSchema(fields...)

	columns := make([]AnyColumn, schema.NumFields()+1)
	for i := 0; i < schema.NumFields(); i++ {
		col, err := f.tbl.Column(schema.Field(i).Name)
		if err != nil {
			return f.withError(err)
		}
		columns[i] = col
	}
	columns[schema.NumFields()] = newCol

	ds, err := factory.FromColumns(newSchema, columns...)
	if err != nil {
		return f.withError(err)
	}
	return Dataset{tbl: ds}
}

// applyTake applies a Take operation to all columns in a dataset using the engine's Selector.
func applySelect(sel Selector, factory ColumnFactory, ds Table, indices []int) (Table, error) {
	schema := ds.Schema()
	columns := make([]AnyColumn, schema.NumFields())
	for i := 0; i < schema.NumFields(); i++ {
		col, err := ds.Column(schema.Field(i).Name)
		if err != nil {
			return nil, err
		}
		taken, err := sel.Select(col, indices)
		if err != nil {
			return nil, err
		}
		columns[i] = taken
	}
	return factory.FromColumns(schema, columns...)
}

// applySlice applies a SliceColumn to all columns in a dataset using the engine's Selector.
func applySlice(sel Selector, factory ColumnFactory, ds Table, start, end int) (Table, error) {
	schema := ds.Schema()
	columns := make([]AnyColumn, schema.NumFields())
	for i := 0; i < schema.NumFields(); i++ {
		col, err := ds.Column(schema.Field(i).Name)
		if err != nil {
			return nil, err
		}
		sliced, err := sel.Slice(col, start, end)
		if err != nil {
			return nil, err
		}
		columns[i] = sliced
	}
	return factory.FromColumns(schema, columns...)
}

// rowKey generates a string key from the specified columns for a given row.
// Used by Distinct for deduplication.
func rowKey(ds Table, cols []string, row int) string {
	parts := make([]string, len(cols))
	for i, name := range cols {
		col, _ := ds.Column(name)
		if col == nil {
			parts[i] = "<nil>"
			continue
		}
		switch col.DType() {
		case DTypeFloat64:
			parts[i] = fmt.Sprintf("%v", col.(Column[float64]).Values()[row])
		case DTypeInt64, DTypeTimestamp:
			parts[i] = fmt.Sprintf("%v", col.(Column[int64]).Values()[row])
		case DTypeString:
			parts[i] = col.(Column[string]).Values()[row]
		case DTypeBool:
			parts[i] = fmt.Sprintf("%v", col.(Column[bool]).Values()[row])
		default:
			parts[i] = "?"
		}
	}
	return strings.Join(parts, "\x00")
}

// renamedColumn wraps an AnyColumn with a different name.
type renamedColumn struct {
	inner   AnyColumn
	newName string
}

func (c *renamedColumn) Name() string { return c.newName }
func (c *renamedColumn) Len() int64   { return c.inner.Len() }
func (c *renamedColumn) DType() DType { return c.inner.DType() }

// renameColumn wraps a column with a new name.
func renameColumn(col AnyColumn, name string) AnyColumn {
	return &renamedColumn{inner: col, newName: name}
}
