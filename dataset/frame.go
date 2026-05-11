package dataset

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"hash/fnv"
	"math"
)

// Dataset is the fluent API for data manipulation. All verbs return a new
// Dataset that records the operation lazily — no computation happens until
// [Dataset.Collect] is called. The chain forms a linked list of [op] nodes
// rooted at a materialised Table.
//
// Usage:
//
//	result, err := dataset.From(ds).
//	    Select("x", "y").
//	    Filter(dataset.Gt("x", 0)).
//	    Arrange("x").
//	    Collect(ctx)
type Dataset struct {
	// Lazy chain.
	eng    Engine   // engine for this chain (propagated from root)
	parent *Dataset // previous node in the chain; nil = root
	op     op       // operation this node represents

	// Materialised state — only populated for root nodes or after Collect.
	tbl Table
	err error
}

// From wraps a Table in a Dataset for fluent verb chaining.
func From(ds Table) Dataset {
	return Dataset{eng: GetEngine(ds), tbl: ds, op: op{kind: opNone}}
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
		return Dataset{}, fmt.Errorf("dataset: %w", err)
	}

	return Dataset{eng: eng, tbl: tbl, op: op{kind: opNone}}, nil
}

// ReplaceColumn returns a lazy Dataset that replaces a named column with
// new float64 values when collected.
func ReplaceColumn(ds Dataset, name string, values []float64) Dataset {
	return Dataset{
		eng:    ds.eng,
		parent: &ds,
		op:     op{kind: opReplaceCol, replaceCol: name, replaceVals: values},
	}
}

// Err returns the first error encountered in the chain, or nil.
func (f Dataset) Err() error { return f.err }

// Table returns the underlying Table, or nil if the Dataset is uncollected.
// Callers must check for nil or call Collect(ctx) before accessing the Table.
func (f Dataset) Table() Table {
	return f.tbl
}

// Column retrieves a named column. Requires a collected Dataset.
func (f Dataset) Column(name string) (AnyColumn, error) {
	if f.tbl == nil {
		return nil, errors.New("dataset: Column() on uncollected Dataset — call Collect(ctx) first")
	}

	col, colErr := f.tbl.Column(name)
	if colErr != nil {
		return nil, fmt.Errorf("dataset: %w", colErr)
	}

	return col, nil
}

// NumRows returns the number of rows, or 0 if uncollected.
func (f Dataset) NumRows() int64 {
	if f.tbl == nil {
		return 0
	}

	return f.tbl.NumRows()
}

// NumCols returns the number of columns, or 0 if uncollected.
func (f Dataset) NumCols() int64 {
	if f.tbl == nil {
		return 0
	}

	return f.tbl.NumCols()
}

// Schema returns the schema, or nil if uncollected.
func (f Dataset) Schema() *Schema {
	if f.tbl == nil {
		return nil
	}

	return f.tbl.Schema()
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

	eng := f.eng
	if eng == nil {
		if f.tbl != nil {
			eng = GetEngine(f.tbl)
		}
	}

	if eng == nil {
		return nil, f.withError(errors.New("dataset: Dataset requires an engine"))
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
	return Dataset{eng: f.eng, parent: &f, op: op{kind: opSelect, cols: cols}}
}

// Rename renames a column.
func (f Dataset) Rename(oldName, newName string) Dataset {
	return Dataset{eng: f.eng, parent: &f, op: op{kind: opRename, renameOld: oldName, renameNew: newName}}
}

func (f Dataset) execSelect(cols []string) Dataset {
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
			return f.withError(&ColumnNotFoundError{Name: name})
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

	return Dataset{eng: eng, tbl: ds}
}

func (f Dataset) execRename(oldName, newName string) Dataset {
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
	for i := range schema.NumFields() {
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

	ns := NewSchema(fields...)

	ds, err := factory.FromColumns(ns, columns...)
	if err != nil {
		return f.withError(err)
	}

	return Dataset{eng: eng, tbl: ds}
}

// --- Filtering ---

// Filter keeps rows where the Masker evaluates to true.
func (f Dataset) Filter(mask Masker) Dataset {
	return Dataset{eng: f.eng, parent: &f, op: op{kind: opFilter, mask: mask}}
}

func (f Dataset) execFilter(mask Masker) Dataset {
	eng, fr := f.requireEngine()
	if fr.err != nil {
		return fr
	}

	if filterer, ok := eng.(Filterer); ok {
		ds, err := filterer.Filter(f.tbl, mask)
		if err != nil {
			return f.withError(err)
		}

		return Dataset{eng: eng, tbl: ds}
	}

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

	return Dataset{eng: eng, tbl: ds}
}

// SelectRows returns a new materialised Dataset containing only the rows at
// the given indices. This is more efficient than Filter when you already have
// indices (avoids O(n) bool-mask allocation).
//
// The Dataset must be materialised (collected). Use on collected datasets only.
func (f Dataset) SelectRows(indices []int) (Dataset, error) {
	if f.tbl == nil {
		return f, errors.New("dataset: SelectRows requires a materialised dataset (call Collect first)")
	}

	eng := f.eng
	if eng == nil {
		eng = GetEngine(f.tbl)
	}

	sel, ok := eng.(Selector)
	if !ok {
		return f, fmt.Errorf("engine %q does not support Selector", eng.Name())
	}

	factory, ok := eng.(ColumnFactory)
	if !ok {
		return f, fmt.Errorf("engine %q does not support ColumnFactory", eng.Name())
	}

	ds, err := applySelect(sel, factory, f.tbl, indices)
	if err != nil {
		return f, err
	}

	return Dataset{eng: eng, tbl: ds}, nil
}

// --- Sorting ---

// Arrange sorts the dataset by the named column (ascending).
func (f Dataset) Arrange(cols ...string) Dataset {
	return Dataset{eng: f.eng, parent: &f, op: op{kind: opArrange, cols: cols}}
}

// --- Row Slicing ---

// Head returns the first n rows.
func (f Dataset) Head(n int) Dataset {
	return Dataset{eng: f.eng, parent: &f, op: op{kind: opHead, n: n}}
}

// Tail returns the last n rows.
func (f Dataset) Tail(n int) Dataset {
	return Dataset{eng: f.eng, parent: &f, op: op{kind: opTail, n: n}}
}

// Slice returns rows in the range [start, end).
func (f Dataset) Slice(start, end int) Dataset {
	return Dataset{eng: f.eng, parent: &f, op: op{kind: opSlice, start: start, end: end}}
}

// --- Distinct ---

// Distinct removes duplicate rows based on the specified columns.
func (f Dataset) Distinct(cols ...string) Dataset {
	return Dataset{eng: f.eng, parent: &f, op: op{kind: opDistinct, cols: cols}}
}

// --- exec methods ---

func (f Dataset) execArrange(cols []string) Dataset {
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

	return Dataset{eng: eng, tbl: ds}
}

func (f Dataset) execHead(n int) Dataset {
	if n >= int(f.tbl.NumRows()) {
		return f
	}

	return f.execSlice(0, n)
}

func (f Dataset) execTail(n int) Dataset {
	if n >= int(f.tbl.NumRows()) {
		return f
	}

	return f.execSlice(int(f.tbl.NumRows())-n, int(f.tbl.NumRows()))
}

func (f Dataset) execSlice(start, end int) Dataset {
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

	return Dataset{eng: eng, tbl: ds}
}

func (f Dataset) execDistinct(cols []string) Dataset {
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

	seen := make(map[uint64]struct{}, int(f.tbl.NumRows())/2)

	var indices []int

	hasher := newRowHasher(f.tbl, cols)
	for row := range f.tbl.NumRows() {
		r := int(row)

		key := hasher.hash(r)
		if _, exists := seen[key]; !exists {
			seen[key] = struct{}{}

			indices = append(indices, r)
		}
	}

	ds, err := applySelect(sel, factory, f.tbl, indices)
	if err != nil {
		return f.withError(err)
	}

	return Dataset{eng: eng, tbl: ds}
}

// --- Joins ---

// LeftJoin performs a left join against other on the given key spec.
func (f Dataset) LeftJoin(other Table, spec JoinSpec) Dataset {
	spec.Type = JoinLeft
	return Dataset{eng: f.eng, parent: &f, op: op{kind: opJoin, joinOther: other, joinSpec: spec}}
}

// InnerJoin performs an inner join against other on the given key spec.
func (f Dataset) InnerJoin(other Table, spec JoinSpec) Dataset {
	spec.Type = JoinInner
	return Dataset{eng: f.eng, parent: &f, op: op{kind: opJoin, joinOther: other, joinSpec: spec}}
}

// RightJoin performs a right join against other on the given key spec.
func (f Dataset) RightJoin(other Table, spec JoinSpec) Dataset {
	spec.Type = JoinRight
	return Dataset{eng: f.eng, parent: &f, op: op{kind: opJoin, joinOther: other, joinSpec: spec}}
}

// FullJoin performs a full outer join against other on the given key spec.
func (f Dataset) FullJoin(other Table, spec JoinSpec) Dataset {
	spec.Type = JoinFull
	return Dataset{eng: f.eng, parent: &f, op: op{kind: opJoin, joinOther: other, joinSpec: spec}}
}

// SemiJoin keeps rows from the left that have a match in other.
func (f Dataset) SemiJoin(other Table, spec JoinSpec) Dataset {
	spec.Type = JoinSemi
	return Dataset{eng: f.eng, parent: &f, op: op{kind: opJoin, joinOther: other, joinSpec: spec}}
}

// AntiJoin keeps rows from the left that have no match in other.
func (f Dataset) AntiJoin(other Table, spec JoinSpec) Dataset {
	spec.Type = JoinAnti
	return Dataset{eng: f.eng, parent: &f, op: op{kind: opJoin, joinOther: other, joinSpec: spec}}
}

func (f Dataset) execJoin(other Table, spec JoinSpec) Dataset {
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

	return Dataset{eng: eng, tbl: ds}
}

// --- Reshape ---

// PivotLonger reshapes wide data to long format.
func (f Dataset) PivotLonger(spec PivotLongerSpec) Dataset {
	return Dataset{eng: f.eng, parent: &f, op: op{kind: opPivotLonger, pivotL: spec}}
}

// PivotWider reshapes long data to wide format.
func (f Dataset) PivotWider(spec PivotWiderSpec) Dataset {
	return Dataset{eng: f.eng, parent: &f, op: op{kind: opPivotWider, pivotW: spec}}
}

// Separate splits a string column into multiple columns by a separator.
func (f Dataset) Separate(col string, into []string, sep string) Dataset {
	return Dataset{eng: f.eng, parent: &f, op: op{kind: opSeparate, sepCol: col, into: into, sep: sep}}
}

func (f Dataset) execPivotLonger(spec PivotLongerSpec) Dataset {
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

	return Dataset{eng: eng, tbl: ds}
}

func (f Dataset) execPivotWider(spec PivotWiderSpec) Dataset {
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

	return Dataset{eng: eng, tbl: ds}
}

func (f Dataset) execSeparate(col string, into []string, sep string) Dataset {
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

	return Dataset{eng: eng, tbl: ds}
}

// --- Fill / DropNA ---

// Fill forward- or backward-fills missing values in the named column.
func (f Dataset) Fill(col string, dir FillDirection) Dataset {
	return Dataset{eng: f.eng, parent: &f, op: op{kind: opFill, fillCol: col, fillDir: dir}}
}

// DropNA removes rows with missing values in the specified columns.
func (f Dataset) DropNA(cols ...string) Dataset {
	return Dataset{eng: f.eng, parent: &f, op: op{kind: opDropNA, cols: cols}}
}

func (f Dataset) execFill(col string, dir FillDirection) Dataset {
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

func (f Dataset) execDropNA(cols []string) Dataset {
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

	return Dataset{eng: eng, tbl: ds}
}

// --- Composing ---

// Stack vertically concatenates this dataset with others (row-bind).
func (f Dataset) Stack(others ...Table) Dataset {
	return Dataset{eng: f.eng, parent: &f, op: op{kind: opStack, others: others}}
}

// Combine horizontally concatenates this dataset with others (column-bind).
func (f Dataset) Combine(others ...Table) Dataset {
	return Dataset{eng: f.eng, parent: &f, op: op{kind: opCombine, others: others}}
}

func (f Dataset) execStack(others []Table) Dataset {
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

	return Dataset{eng: eng, tbl: ds}
}

func (f Dataset) execCombine(others []Table) Dataset {
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

	return Dataset{eng: eng, tbl: ds}
}

// --- Internal ---

func (f Dataset) replaceColumn(factory ColumnFactory, name string, newCol AnyColumn) Dataset {
	schema := f.tbl.Schema()

	columns := make([]AnyColumn, schema.NumFields())
	for i := range schema.NumFields() {
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

	return Dataset{eng: f.eng, tbl: ds}
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

// AggSum is the sum aggregation.
const (
	AggSum AggFunc = iota
	AggMean
	AggMin
	AggMax
	AggCount
	AggMedian
	AggVariance
)

// Sum builds a sum aggregation spec.
func Sum(out, in string) AggSpec { return AggSpec{OutputName: out, InputName: in, Fn: AggSum} }

// Mean builds a mean aggregation spec.
func Mean(out, in string) AggSpec { return AggSpec{OutputName: out, InputName: in, Fn: AggMean} }

// Min builds a min aggregation spec.
func Min(out, in string) AggSpec { return AggSpec{OutputName: out, InputName: in, Fn: AggMin} }

// Max builds a max aggregation spec.
func Max(out, in string) AggSpec { return AggSpec{OutputName: out, InputName: in, Fn: AggMax} }

// Count builds a count aggregation spec.
func Count(out, in string) AggSpec { return AggSpec{OutputName: out, InputName: in, Fn: AggCount} }

// Median builds a median aggregation spec.
func Median(out, in string) AggSpec { return AggSpec{OutputName: out, InputName: in, Fn: AggMedian} }

// Variance builds a variance aggregation spec.
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

// Summarize applies aggregations per group, producing a lazy Dataset.
func (gf GroupedFrame) Summarize(specs ...AggSpec) Dataset {
	f := gf.frame

	return Dataset{
		eng:    f.eng,
		parent: &f,
		op:     op{kind: opGroupBy, cols: gf.groupCols, aggSpecs: specs},
	}
}

// dispatchAgg calls the appropriate Aggregator method.
func dispatchAgg(agg Aggregator, fn AggFunc, col AnyColumn) (AnyColumn, error) {
	switch fn {
	case AggSum:
		result, err := agg.Sum(col)
		if err != nil {
			return nil, fmt.Errorf("dataset: %w", err)
		}

		return result, nil
	case AggMean:
		result, err := agg.Mean(col)
		if err != nil {
			return nil, fmt.Errorf("dataset: %w", err)
		}

		return result, nil
	case AggMin:
		lo, _, err := agg.MinMax(col)
		if err != nil {
			return nil, fmt.Errorf("dataset: %w", err)
		}

		return lo, nil
	case AggMax:
		_, hi, err := agg.MinMax(col)
		if err != nil {
			return nil, fmt.Errorf("dataset: %w", err)
		}

		return hi, nil
	case AggCount:
		result, err := agg.Count(col)
		if err != nil {
			return nil, fmt.Errorf("dataset: %w", err)
		}

		return result, nil
	case AggMedian:
		result, err := agg.Median(col)
		if err != nil {
			return nil, fmt.Errorf("dataset: %w", err)
		}

		return result, nil
	case AggVariance:
		result, err := agg.Variance(col)
		if err != nil {
			return nil, fmt.Errorf("dataset: %w", err)
		}

		return result, nil
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
		return nil, errors.New("dataset: no agg results to merge")
	}

	n := len(results)
	switch results[0].DType() { //nolint:exhaustive // handled by default case.
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
	return Dataset{eng: f.eng, parent: &f, op: op{kind: opMutate, mutName: name, mutFn: fn}}
}

// applyTake applies a Take operation to all columns in a dataset using the engine's Selector.
func applySelect(sel Selector, factory ColumnFactory, ds Table, indices []int) (Table, error) {
	schema := ds.Schema()

	columns := make([]AnyColumn, schema.NumFields())
	for i := range schema.NumFields() {
		col, err := ds.Column(schema.Field(i).Name)
		if err != nil {
			return nil, fmt.Errorf("dataset: %w", err)
		}

		taken, err := sel.Select(col, indices)
		if err != nil {
			return nil, fmt.Errorf("dataset: %w", err)
		}

		columns[i] = taken
	}

	result, err := factory.FromColumns(schema, columns...)
	if err != nil {
		return nil, fmt.Errorf("dataset: %w", err)
	}

	return result, nil
}

// applySlice applies a SliceColumn to all columns in a dataset using the engine's Selector.
func applySlice(sel Selector, factory ColumnFactory, ds Table, start, end int) (Table, error) {
	schema := ds.Schema()

	columns := make([]AnyColumn, schema.NumFields())
	for i := range schema.NumFields() {
		col, err := ds.Column(schema.Field(i).Name)
		if err != nil {
			return nil, fmt.Errorf("dataset: %w", err)
		}

		sliced, err := sel.Slice(col, start, end)
		if err != nil {
			return nil, fmt.Errorf("dataset: %w", err)
		}

		columns[i] = sliced
	}

	result, err := factory.FromColumns(schema, columns...)
	if err != nil {
		return nil, fmt.Errorf("dataset: %w", err)
	}

	return result, nil
}

// rowHasher caches the underlying Value arrays to avoid O(N^2) allocations
// during hashing. Construct once via newRowHasher, then call hash(row) per row.
//
// Used by Distinct and GroupBy for deduplication/grouping.
//
// TODO: FNV-64 has a birthday-bound collision probability that becomes
// non-negligible above ~4B distinct groups. For datasets with >10M distinct
// keys consider adding an equality fallback (secondary check on collision).
type rowHasher struct {
	h      hash.Hash64
	buf    [8]byte
	ncols  int
	dtypes []DType
	float  [][]float64
	intv   [][]int64
	str    [][]string
	boolv  [][]bool
	nulls  [][]bool
}

func newRowHasher(ds Table, cols []string) *rowHasher {
	n := len(cols)
	rh := &rowHasher{
		h:      fnv.New64a(),
		ncols:  n,
		dtypes: make([]DType, n),
		float:  make([][]float64, n),
		intv:   make([][]int64, n),
		str:    make([][]string, n),
		boolv:  make([][]bool, n),
		nulls:  make([][]bool, n),
	}

	for i, name := range cols {
		col, _ := ds.Column(name)
		if col == nil {
			continue // dtypes[i] stays 0 → falls through to default sentinel
		}

		rh.dtypes[i] = col.DType()
		switch c := col.(type) {
		case Column[float64]:
			rh.float[i] = c.Values()
			rh.nulls[i] = c.IsNull()
		case Column[int64]:
			rh.intv[i] = c.Values()
			rh.nulls[i] = c.IsNull()
		case Column[string]:
			// NOTE: Arrow's arrowStringColumn.Values() materialises a full
			// []string copy. For very large string-keyed GroupBys, a future
			// optimisation could hash Arrow string values in-place via
			// arr.Value(i) to avoid this allocation.
			rh.str[i] = c.Values()
			rh.nulls[i] = c.IsNull()
		case Column[bool]:
			rh.boolv[i] = c.Values()
			rh.nulls[i] = c.IsNull()
		}
	}

	return rh
}

func (rh *rowHasher) hash(row int) uint64 {
	rh.h.Reset()

	for i := range rh.ncols {
		if rh.nulls[i] != nil && rh.nulls[i][row] {
			_, _ = rh.h.Write([]byte{0xFF})
			continue
		}

		switch rh.dtypes[i] { //nolint:exhaustive // handled by default case.
		case DTypeFloat64:
			binary.LittleEndian.PutUint64(rh.buf[:], math.Float64bits(rh.float[i][row]))
			_, _ = rh.h.Write(rh.buf[:])
		case DTypeInt64, DTypeTimestamp:
			binary.LittleEndian.PutUint64(rh.buf[:], uint64(rh.intv[i][row])) //nolint:gosec // G115: int64 row values are non-negative.
			_, _ = rh.h.Write(rh.buf[:])
		case DTypeString:
			_, _ = rh.h.Write([]byte(rh.str[i][row]))
			_, _ = rh.h.Write([]byte{0})
		case DTypeBool:
			if rh.boolv[i][row] {
				_, _ = rh.h.Write([]byte{1})
			} else {
				_, _ = rh.h.Write([]byte{0})
			}
		default:
			_, _ = rh.h.Write([]byte{0xFF})
		}
	}

	return rh.h.Sum64()
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

// --- exec: GroupBy + Summarize ---

func (f Dataset) execGroupBy(groupCols []string, specs []AggSpec) Dataset {
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

	type group struct{ indices []int }

	seen := make(map[uint64]int, int(f.tbl.NumRows())/2)

	var groups []group

	hasher := newRowHasher(f.tbl, groupCols)
	for row := range f.tbl.NumRows() {
		r := int(row)

		key := hasher.hash(r)
		if idx, exists := seen[key]; exists {
			groups[idx].indices = append(groups[idx].indices, r)
		} else {
			seen[key] = len(groups)
			groups = append(groups, group{indices: []int{r}})
		}
	}

	nGroups := len(groups)

	var outFields []Field

	for _, name := range groupCols {
		idx := f.tbl.Schema().FieldIndex(name)
		if idx < 0 {
			return f.withError(&ColumnNotFoundError{Name: name})
		}

		outFields = append(outFields, f.tbl.Schema().Field(idx))
	}

	for _, spec := range specs {
		dtype := resolveAggDType(spec.Fn, f.tbl, spec.InputName)
		outFields = append(outFields, Field{Name: spec.OutputName, Dtype: dtype})
	}

	outSchema := NewSchema(outFields...)
	outCols := make([]AnyColumn, len(outFields))

	firstIndices := make([]int, nGroups)
	for i, g := range groups {
		firstIndices[i] = g.indices[0]
	}

	for ci, name := range groupCols {
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

	for si, spec := range specs {
		col, err := f.tbl.Column(spec.InputName)
		if err != nil {
			return f.withError(err)
		}

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

		merged, err := mergeAggResults(factory, spec.OutputName, aggResults)
		if err != nil {
			return f.withError(err)
		}

		outCols[len(groupCols)+si] = merged
	}

	ds, err := factory.FromColumns(outSchema, outCols...)
	if err != nil {
		return f.withError(err)
	}

	return Dataset{eng: eng, tbl: ds}
}

// --- exec: Mutate ---

func (f Dataset) execMutate(name string, fn MutateFunc) Dataset {
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

	schema := f.tbl.Schema()
	if schema.HasField(name) {
		return f.replaceColumn(factory, name, newCol)
	}

	fields := schema.Fields()
	fields = append(fields, Field{Name: name, Dtype: newCol.DType()})
	newSchema := NewSchema(fields...)

	columns := make([]AnyColumn, schema.NumFields()+1)
	for i := range schema.NumFields() {
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

	return Dataset{eng: eng, tbl: ds}
}

// --- exec: ReplaceColumn ---

func (f Dataset) execReplaceCol(colName string, values []float64) Dataset {
	eng, fr := f.requireEngine()
	if fr.err != nil {
		return fr
	}

	factory, ok := eng.(ColumnFactory)
	if !ok {
		return f.withError(fmt.Errorf("engine %q does not support ColumnFactory", eng.Name()))
	}

	newCol := factory.NewFloat64Column(colName, values)

	return f.replaceColumn(factory, colName, newCol)
}
