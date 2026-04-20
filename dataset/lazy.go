package dataset

import (
	"fmt"
	"math"
	"sort"
)

// --- Select ---

type selectDataset struct {
	parent Dataset
	cols   []string
}

func (s *selectDataset) Columns() []string { return s.cols }
func (s *selectDataset) Len() int          { return s.parent.Len() }
func (s *selectDataset) Column(name string) (Column, error) {
	for _, c := range s.cols {
		if c == name {
			return s.parent.Column(name)
		}
	}
	return nil, &ErrColumnNotFound{Name: name}
}

// --- Filter (lazy with predicate) ---

type lazyFilterDataset struct {
	parent Dataset
	pred   Predicate
	// cached evaluation
	mask   []bool
	count  int
	evaled bool
}

func (f *lazyFilterDataset) eval() error {
	if f.evaled {
		return nil
	}
	mask, err := f.pred.Eval(f.parent)
	if err != nil {
		return err
	}
	f.mask = mask
	f.count = 0
	for _, b := range mask {
		if b {
			f.count++
		}
	}
	f.evaled = true
	return nil
}

func (f *lazyFilterDataset) Columns() []string { return f.parent.Columns() }
func (f *lazyFilterDataset) Len() int {
	if err := f.eval(); err != nil {
		return 0
	}
	return f.count
}

func (f *lazyFilterDataset) Column(name string) (Column, error) {
	if err := f.eval(); err != nil {
		return nil, err
	}
	col, err := f.parent.Column(name)
	if err != nil {
		return nil, err
	}
	// Delegate to native filter if available.
	if p, ok := col.(NativeFilterProvider); ok {
		if c, err := p.FilterColumn(f.mask, f.count); err == nil && c != nil {
			return c, nil
		}
	}
	// Generic fallback: materialize via iterator.
	return filterColumnGeneric(col, f.mask, f.count)
}

// filterColumnGeneric materializes a filtered column using iterators.
func filterColumnGeneric(col Column, mask []bool, count int) (Column, error) {
	iter, ok := col.(IterableColumn)
	if !ok {
		return nil, fmt.Errorf("dataset: column does not support iteration for filtering")
	}

	switch col.DType() {
	case DTypeFloat64:
		flt, err := iter.Float64s()
		if err != nil {
			return nil, err
		}
		data := make([]float64, 0, count)
		var nulls []bool
		for i, keep := range mask {
			if i >= col.Len() {
				break
			}
			v, isNull, ok := flt.Next()
			if !ok {
				break
			}
			if keep {
				data = append(data, v)
				if isNull {
					if nulls == nil {
						nulls = make([]bool, len(data)-1, count)
					}
					nulls = append(nulls, true)
				} else if nulls != nil {
					nulls = append(nulls, false)
				}
			}
		}
		return &Float64Column{Data: data, Nulls: nulls}, nil

	case DTypeInt64:
		it, err := iter.Int64s()
		if err != nil {
			return nil, err
		}
		data := make([]int64, 0, count)
		for _, keep := range mask {
			v, _, ok := it.Next()
			if !ok {
				break
			}
			if keep {
				data = append(data, v)
			}
		}
		return &Int64Column{Data: data}, nil

	case DTypeString:
		sit, err := iter.Strings()
		if err != nil {
			return nil, err
		}
		data := make([]string, 0, count)
		for _, keep := range mask {
			v, _, ok := sit.Next()
			if !ok {
				break
			}
			if keep {
				data = append(data, v)
			}
		}
		return &StringColumn{Data: data}, nil

	default:
		return nil, fmt.Errorf("dataset: unsupported column type %s for filtering", col.DType())
	}
}

// --- Filter (pre-computed mask, for backward compat and GroupBy) ---

// FilterMask creates a filtered dataset from a pre-computed boolean mask.
func FilterMask(parent Dataset, mask []bool) Dataset {
	count := 0
	for _, b := range mask {
		if b {
			count++
		}
	}
	return &maskedDataset{parent: parent, mask: mask, count: count}
}

type maskedDataset struct {
	parent Dataset
	mask   []bool
	count  int
}

func (m *maskedDataset) Columns() []string { return m.parent.Columns() }
func (m *maskedDataset) Len() int          { return m.count }
func (m *maskedDataset) Column(name string) (Column, error) {
	col, err := m.parent.Column(name)
	if err != nil {
		return nil, err
	}
	if p, ok := col.(NativeFilterProvider); ok {
		if c, err := p.FilterColumn(m.mask, m.count); err == nil && c != nil {
			return c, nil
		}
	}
	return filterColumnGeneric(col, m.mask, m.count)
}

// --- Mutate ---

type mutateDataset struct {
	parent  Dataset
	colName string
	fn      ColumnFunc
}

func (m *mutateDataset) Columns() []string {
	parent := m.parent.Columns()
	for _, c := range parent {
		if c == m.colName {
			return parent // replacing existing column
		}
	}
	return append(append([]string(nil), parent...), m.colName)
}

func (m *mutateDataset) Len() int { return m.parent.Len() }

func (m *mutateDataset) Column(name string) (Column, error) {
	if name == m.colName {
		return m.fn(m.parent)
	}
	return m.parent.Column(name)
}

// --- Rename ---

type renameDataset struct {
	parent   Dataset
	old, new string
}

func (r *renameDataset) Columns() []string {
	cols := r.parent.Columns()
	out := make([]string, len(cols))
	for i, c := range cols {
		if c == r.old {
			out[i] = r.new
		} else {
			out[i] = c
		}
	}
	return out
}

func (r *renameDataset) Len() int { return r.parent.Len() }
func (r *renameDataset) Column(name string) (Column, error) {
	if name == r.new {
		return r.parent.Column(r.old)
	}
	if name == r.old {
		return nil, &ErrColumnNotFound{Name: name}
	}
	return r.parent.Column(name)
}

// --- Slice ---

func sliceDatasetFrom(d Dataset, i, j int) Dataset {
	if i < 0 {
		i = 0
	}
	if j > d.Len() {
		j = d.Len()
	}
	if i > j {
		i = j
	}
	if sp, ok := d.(NativeSliceProvider); ok {
		if native := sp.SliceDataset(i, j); native != nil {
			return native
		}
	}
	return &slicedDataset{parent: d, offset: i, length: j - i}
}

type slicedDataset struct {
	parent Dataset
	offset int
	length int
}

func (s *slicedDataset) Columns() []string { return s.parent.Columns() }
func (s *slicedDataset) Len() int          { return s.length }
func (s *slicedDataset) Column(name string) (Column, error) {
	col, err := s.parent.Column(name)
	if err != nil {
		return nil, err
	}
	if p, ok := col.(NativeColumnSliceProvider); ok {
		if native := p.SliceColumn(s.offset, s.offset+s.length); native != nil {
			return native, nil
		}
	}
	// Generic fallback: materialize the slice.
	return sliceColumnGeneric(col, s.offset, s.length)
}

func sliceColumnGeneric(col Column, offset, length int) (Column, error) {
	iter, ok := col.(IterableColumn)
	if !ok {
		return nil, fmt.Errorf("dataset: column does not support iteration for slicing")
	}
	switch col.DType() {
	case DTypeFloat64:
		flt, err := iter.Float64s()
		if err != nil {
			return nil, err
		}
		// Skip offset items.
		for i := 0; i < offset; i++ {
			if _, _, ok := flt.Next(); !ok {
				break
			}
		}
		data := make([]float64, 0, length)
		for i := 0; i < length; i++ {
			v, _, ok := flt.Next()
			if !ok {
				break
			}
			data = append(data, v)
		}
		return &Float64Column{Data: data}, nil

	case DTypeString:
		sit, err := iter.Strings()
		if err != nil {
			return nil, err
		}
		for i := 0; i < offset; i++ {
			if _, _, ok := sit.Next(); !ok {
				break
			}
		}
		data := make([]string, 0, length)
		for i := 0; i < length; i++ {
			v, _, ok := sit.Next()
			if !ok {
				break
			}
			data = append(data, v)
		}
		return &StringColumn{Data: data}, nil

	default:
		return nil, fmt.Errorf("dataset: unsupported column type %s for slicing", col.DType())
	}
}

// --- Distinct (fully implemented) ---

type distinctDataset struct {
	parent Dataset
	cols   []string
	// cached
	built   bool
	indices []int
}

func (d *distinctDataset) buildIndices() {
	if d.built {
		return
	}
	n := d.parent.Len()

	// Determine which columns to use for distinct keys.
	keyCols := d.cols
	if len(keyCols) == 0 {
		keyCols = d.parent.Columns()
	}

	// Build row keys by reading all key columns.
	type rowIter struct {
		sit StringIter
		flt Float64Iter
		dt  DType
	}
	iters := make([]rowIter, len(keyCols))
	for i, name := range keyCols {
		col, err := d.parent.Column(name)
		if err != nil {
			continue
		}
		iter, ok := col.(IterableColumn)
		if !ok {
			continue
		}
		ri := rowIter{dt: col.DType()}
		switch col.DType() {
		case DTypeString:
			ri.sit, _ = iter.Strings()
		default:
			ri.flt, _ = iter.Float64s()
		}
		iters[i] = ri
	}

	seen := make(map[string]struct{})
	d.indices = make([]int, 0, n)
	for row := 0; row < n; row++ {
		key := ""
		for _, ri := range iters {
			if ri.sit != nil {
				v, _, _ := ri.sit.Next()
				key += v + "\x00"
			} else if ri.flt != nil {
				v, _, _ := ri.flt.Next()
				key += fmt.Sprintf("%g\x00", v)
			}
		}
		if _, exists := seen[key]; !exists {
			seen[key] = struct{}{}
			d.indices = append(d.indices, row)
		}
	}
	d.built = true
}

func (d *distinctDataset) Columns() []string { return d.parent.Columns() }
func (d *distinctDataset) Len() int {
	d.buildIndices()
	return len(d.indices)
}
func (d *distinctDataset) Column(name string) (Column, error) {
	d.buildIndices()
	col, err := d.parent.Column(name)
	if err != nil {
		return nil, err
	}
	// Build a mask from indices.
	n := d.parent.Len()
	mask := make([]bool, n)
	for _, idx := range d.indices {
		if idx < n {
			mask[idx] = true
		}
	}
	if p, ok := col.(NativeFilterProvider); ok {
		if c, err := p.FilterColumn(mask, len(d.indices)); err == nil && c != nil {
			return c, nil
		}
	}
	return filterColumnGeneric(col, mask, len(d.indices))
}

// --- Arrange (fully implemented) ---

type arrangeDataset struct {
	parent Dataset
	col    string
	desc   bool
	// cached
	built   bool
	indices []int
}

func (a *arrangeDataset) buildIndices() {
	if a.built {
		return
	}
	n := a.parent.Len()
	a.indices = make([]int, n)
	for i := range a.indices {
		a.indices[i] = i
	}

	col, err := a.parent.Column(a.col)
	if err != nil {
		a.built = true
		return
	}

	vals, err := CollectFloat64s(col)
	if err != nil {
		a.built = true
		return
	}

	desc := a.desc
	sort.Slice(a.indices, func(i, j int) bool {
		vi, vj := vals[a.indices[i]], vals[a.indices[j]]
		if math.IsNaN(vi) {
			return false // NaN sorts last
		}
		if math.IsNaN(vj) {
			return true
		}
		if desc {
			return vi > vj
		}
		return vi < vj
	})
	a.built = true
}

func (a *arrangeDataset) Columns() []string { return a.parent.Columns() }
func (a *arrangeDataset) Len() int          { return a.parent.Len() }
func (a *arrangeDataset) Column(name string) (Column, error) {
	a.buildIndices()
	col, err := a.parent.Column(name)
	if err != nil {
		return nil, err
	}
	return reorderColumn(col, a.indices)
}

func reorderColumn(col Column, indices []int) (Column, error) {
	iter, ok := col.(IterableColumn)
	if !ok {
		return nil, fmt.Errorf("dataset: column does not support iteration for reordering")
	}
	switch col.DType() {
	case DTypeFloat64:
		flt, err := iter.Float64s()
		if err != nil {
			return nil, err
		}
		allVals := make([]float64, col.Len())
		for i := range allVals {
			v, _, ok := flt.Next()
			if !ok {
				break
			}
			allVals[i] = v
		}
		result := make([]float64, len(indices))
		for i, idx := range indices {
			result[i] = allVals[idx]
		}
		return &Float64Column{Data: result}, nil

	case DTypeInt64:
		it, err := iter.Int64s()
		if err != nil {
			return nil, err
		}
		allVals := make([]int64, col.Len())
		for i := range allVals {
			v, _, ok := it.Next()
			if !ok {
				break
			}
			allVals[i] = v
		}
		result := make([]int64, len(indices))
		for i, idx := range indices {
			result[i] = allVals[idx]
		}
		return &Int64Column{Data: result}, nil

	case DTypeString:
		sit, err := iter.Strings()
		if err != nil {
			return nil, err
		}
		allVals := make([]string, col.Len())
		for i := range allVals {
			v, _, ok := sit.Next()
			if !ok {
				break
			}
			allVals[i] = v
		}
		result := make([]string, len(indices))
		for i, idx := range indices {
			result[i] = allVals[idx]
		}
		return &StringColumn{Data: result}, nil

	default:
		return nil, fmt.Errorf("dataset: unsupported column type %s for reordering", col.DType())
	}
}

// --- Summarize (GroupBy + Aggregation) ---

type summarizeDataset struct {
	parent    Dataset
	groupCols []string
	aggs      []Aggregation

	built   bool
	columns map[string]Column
	nRows   int
}

func (s *summarizeDataset) Columns() []string {
	out := make([]string, 0, len(s.groupCols)+len(s.aggs))
	out = append(out, s.groupCols...)
	for _, a := range s.aggs {
		out = append(out, a.Name)
	}
	return out
}

func (s *summarizeDataset) Len() int {
	if err := s.build(); err != nil {
		return 0
	}
	return s.nRows
}

func (s *summarizeDataset) Column(name string) (Column, error) {
	if err := s.build(); err != nil {
		return nil, err
	}
	col, ok := s.columns[name]
	if !ok {
		return nil, &ErrColumnNotFound{Name: name}
	}
	return col, nil
}

func (s *summarizeDataset) build() error {
	if s.built {
		return nil
	}

	type groupKey string
	groups := make(map[groupKey][]int)
	var order []groupKey

	n := s.parent.Len()

	// Build group keys from the grouping columns.
	keyParts := make([]StringIter, len(s.groupCols))
	for i, gc := range s.groupCols {
		col, err := s.parent.Column(gc)
		if err != nil {
			return fmt.Errorf("dataset: summarize group column %q: %w", gc, err)
		}
		iter, ok := col.(IterableColumn)
		if !ok {
			return fmt.Errorf("dataset: group column %q does not support iteration", gc)
		}
		// Use Strings() for grouping; float64/int64 columns provide
		// cross-type String iteration automatically via the new column types.
		sit, err := iter.Strings()
		if err != nil {
			return fmt.Errorf("dataset: group column %q: %w", gc, err)
		}
		keyParts[i] = sit
	}

	for row := 0; row < n; row++ {
		keyStr := ""
		for _, kp := range keyParts {
			v, isNull, ok := kp.Next()
			if !ok {
				break
			}
			if isNull {
				v = "NA"
			}
			keyStr += v + "\x00"
		}
		k := groupKey(keyStr)
		if _, exists := groups[k]; !exists {
			order = append(order, k)
		}
		groups[k] = append(groups[k], row)
	}

	nGroups := len(order)
	s.nRows = nGroups
	s.columns = make(map[string]Column, len(s.groupCols)+len(s.aggs))

	// Build group-key columns (take first row of each group).
	for _, gc := range s.groupCols {
		col, err := s.parent.Column(gc)
		if err != nil {
			return err
		}
		allVals, err := CollectStrings(col)
		if err != nil {
			continue
		}
		groupVals := make([]string, nGroups)
		for i, k := range order {
			rows := groups[k]
			if len(rows) > 0 && rows[0] < len(allVals) {
				groupVals[i] = allVals[rows[0]]
			}
		}
		s.columns[gc] = &StringColumn{Data: groupVals}
	}

	// Build aggregation columns.
	for _, agg := range s.aggs {
		col, err := s.parent.Column(agg.Source)
		if err != nil {
			return fmt.Errorf("dataset: summarize source column %q: %w", agg.Source, err)
		}
		allVals, err := CollectFloat64s(col)
		if err != nil {
			return err
		}

		results := make([]float64, nGroups)
		for i, k := range order {
			rows := groups[k]
			vals := make([]float64, 0, len(rows))
			for _, r := range rows {
				if r < len(allVals) && !math.IsNaN(allVals[r]) {
					vals = append(vals, allVals[r])
				}
			}
			results[i] = agg.Fn(vals)
		}
		s.columns[agg.Name] = &Float64Column{Data: results}
	}

	s.built = true
	return nil
}

// --- Transformed columns ---

type transformedFloat64Column struct {
	parent IterableColumn
	mapper func(float64) float64
}

func (t *transformedFloat64Column) Len() int     { return t.parent.Len() }
func (t *transformedFloat64Column) DType() DType { return DTypeFloat64 }
func (t *transformedFloat64Column) Float64s() (Float64Iter, error) {
	iter, err := t.parent.Float64s()
	if err != nil {
		return nil, err
	}
	return &mappedFloat64Iter{parent: iter, mapper: t.mapper}, nil
}
func (t *transformedFloat64Column) Int64s() (Int64Iter, error) {
	return nil, fmt.Errorf("dataset: transformed float64 column does not support Int64 iteration")
}
func (t *transformedFloat64Column) Strings() (StringIter, error) {
	return nil, fmt.Errorf("dataset: transformed float64 column does not support String iteration")
}

type mappedFloat64Iter struct {
	parent Float64Iter
	mapper func(float64) float64
}

func (i *mappedFloat64Iter) Next() (float64, bool, bool) {
	v, isNull, ok := i.parent.Next()
	if !ok || isNull {
		return v, isNull, ok
	}
	return i.mapper(v), false, true
}

type transformedStringColumn struct {
	parent IterableColumn
	mapper func(string) float64
}

func (t *transformedStringColumn) Len() int     { return t.parent.Len() }
func (t *transformedStringColumn) DType() DType { return DTypeFloat64 }
func (t *transformedStringColumn) Float64s() (Float64Iter, error) {
	iter, err := t.parent.Strings()
	if err != nil {
		return nil, err
	}
	return &mappedStringToFloat64Iter{parent: iter, mapper: t.mapper}, nil
}
func (t *transformedStringColumn) Int64s() (Int64Iter, error) {
	return nil, fmt.Errorf("dataset: transformed string column does not support Int64 iteration")
}
func (t *transformedStringColumn) Strings() (StringIter, error) {
	return nil, fmt.Errorf("dataset: transformed string→float64 column does not support String iteration")
}

type mappedStringToFloat64Iter struct {
	parent StringIter
	mapper func(string) float64
}

func (i *mappedStringToFloat64Iter) Next() (float64, bool, bool) {
	v, isNull, ok := i.parent.Next()
	if !ok || isNull {
		return 0, isNull, ok
	}
	return i.mapper(v), false, true
}

type constFloat64Column struct {
	val    float64
	length int
}

func (c *constFloat64Column) Len() int     { return c.length }
func (c *constFloat64Column) DType() DType { return DTypeFloat64 }
func (c *constFloat64Column) Float64s() (Float64Iter, error) {
	return &constFloat64Iter{val: c.val, remaining: c.length}, nil
}
func (c *constFloat64Column) Int64s() (Int64Iter, error) {
	return nil, fmt.Errorf("dataset: constant float64 column does not support Int64 iteration")
}
func (c *constFloat64Column) Strings() (StringIter, error) {
	return nil, fmt.Errorf("dataset: constant float64 column does not support String iteration")
}

type constFloat64Iter struct {
	val       float64
	remaining int
}

func (i *constFloat64Iter) Next() (float64, bool, bool) {
	if i.remaining <= 0 {
		return 0, false, false
	}
	i.remaining--
	return i.val, false, true
}

// --- Frame.Collect materialization ---

func (f Frame) collect() (Frame, error) {
	cols := f.Dataset.Columns()
	memCols := make(map[string]Column, len(cols))
	n := f.Dataset.Len()

	for _, name := range cols {
		col, err := f.Dataset.Column(name)
		if err != nil {
			return Frame{}, err
		}
		iter, ok := col.(IterableColumn)
		if !ok {
			memCols[name] = col
			continue
		}

		switch col.DType() {
		case DTypeFloat64:
			flt, err := iter.Float64s()
			if err != nil {
				return Frame{}, err
			}
			data := make([]float64, 0, n)
			var nulls []bool
			for {
				v, isNull, ok := flt.Next()
				if !ok {
					break
				}
				data = append(data, v)
				if isNull {
					if nulls == nil {
						nulls = make([]bool, len(data)-1, n)
					}
					nulls = append(nulls, true)
				} else if nulls != nil {
					nulls = append(nulls, false)
				}
			}
			memCols[name] = &Float64Column{Data: data, Nulls: nulls}

		case DTypeInt64:
			it, err := iter.Int64s()
			if err != nil {
				return Frame{}, err
			}
			data := make([]int64, 0, n)
			for {
				v, _, ok := it.Next()
				if !ok {
					break
				}
				data = append(data, v)
			}
			memCols[name] = &Int64Column{Data: data}

		case DTypeString:
			sit, err := iter.Strings()
			if err != nil {
				return Frame{}, err
			}
			data := make([]string, 0, n)
			for {
				v, _, ok := sit.Next()
				if !ok {
					break
				}
				data = append(data, v)
			}
			memCols[name] = &StringColumn{Data: data}

		default:
			memCols[name] = col
		}
	}

	length := 0
	for _, col := range memCols {
		length = col.Len()
		break
	}

	return Frame{Dataset: &memoryDataset{cols: cols, columns: memCols, length: length}}, nil
}

// --- Compile-time checks ---

var (
	_ Dataset = (*selectDataset)(nil)
	_ Dataset = (*lazyFilterDataset)(nil)
	_ Dataset = (*maskedDataset)(nil)
	_ Dataset = (*mutateDataset)(nil)
	_ Dataset = (*renameDataset)(nil)
	_ Dataset = (*slicedDataset)(nil)
	_ Dataset = (*distinctDataset)(nil)
	_ Dataset = (*arrangeDataset)(nil)
	_ Dataset = (*summarizeDataset)(nil)

	_ IterableColumn = (*transformedFloat64Column)(nil)
	_ IterableColumn = (*transformedStringColumn)(nil)
	_ IterableColumn = (*constFloat64Column)(nil)
)
