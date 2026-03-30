package dataset

import "fmt"

// MutateRegistry maps names to column factories for lazy derived datasets.
type MutateRegistry map[string]func() (Column, error)

type mutateDataset struct {
	parent Dataset
	mut    MutateRegistry
}

// Mutate returns a lazily mutated dataset.
func Mutate(parent Dataset, mut MutateRegistry) Dataset {
	return &mutateDataset{parent: parent, mut: mut}
}

func (m *mutateDataset) Columns() []string {
	keys := make(map[string]bool)
	for _, c := range m.parent.Columns() {
		keys[c] = true
	}
	for k := range m.mut {
		keys[k] = true
	}
	cols := make([]string, 0, len(keys))
	for k := range keys {
		cols = append(cols, k)
	}
	return cols
}

func (m *mutateDataset) Column(name string) (Column, error) {
	if factory, ok := m.mut[name]; ok {
		return factory()
	}
	return m.parent.Column(name)
}

func (m *mutateDataset) Len() int { return m.parent.Len() }

// Filter constructs a lazily evaluated derived dataset from a boolean mask.
func Filter(parent Dataset, mask []bool) Dataset {
	l := 0
	for _, keep := range mask {
		if keep {
			l++
		}
	}
	return &filterDataset{parent: parent, mask: mask, len: l}
}

type filterDataset struct {
	parent Dataset
	mask   []bool
	len    int
}

func (f *filterDataset) Columns() []string { return f.parent.Columns() }
func (f *filterDataset) Len() int          { return f.len }

func (f *filterDataset) Column(name string) (Column, error) {
	col, err := f.parent.Column(name)
	if err != nil {
		return nil, err
	}

	// Request the underlying concrete implementer (like Arrow) to filter itself zero-copy
	// using the mask, typically by building a DictionaryArray.
	if p, ok := col.(NativeFilterProvider); ok {
		if c, err := p.FilterColumn(f.mask, f.len); err == nil && c != nil {
			return c, nil
		}
	}
	return &filterColumn{parent: col, mask: f.mask, len: f.len}, nil
}

// NativeFilterProvider asks the underlying column to run a native abstraction filter.
type NativeFilterProvider interface {
	FilterColumn(mask []bool, count int) (Column, error)
}

type filterColumn struct {
	parent Column
	mask   []bool
	len    int
}

func (fc *filterColumn) Len() int { return fc.len }

// GroupByDataset is a logical partition over a dataset.
type GroupByDataset struct {
	parent Dataset
	groups map[string]Dataset
}

// GroupBy stub for dividing a dataset.
func GroupBy(parent Dataset, column string) (*GroupByDataset, error) {
	col, err := parent.Column(column)
	if err != nil {
		return nil, err
	}

	iterCol, ok := col.(IterableColumn)
	if !ok {
		return nil, fmt.Errorf("dataset: column %q does not support string iteration for grouping", column)
	}

	iter, err := iterCol.Strings()
	if err != nil {
		return nil, err
	}

	groupMasks := make(map[string][]bool)
	n := parent.Len()
	i := 0
	for {
		val, isNull, ok := iter.Next()
		if !ok {
			break
		}
		if isNull {
			val = "NA"
		}

		mask, exists := groupMasks[val]
		if !exists {
			mask = make([]bool, n)
			groupMasks[val] = mask
		}
		mask[i] = true
		i++
	}

	g := &GroupByDataset{
		parent: parent,
		groups: make(map[string]Dataset, len(groupMasks)),
	}

	for val, mask := range groupMasks {
		g.groups[val] = Filter(parent, mask)
	}

	return g, nil
}

// TransformedFloat64Column wraps a parent column
// mapping mathematical transforms continuously!
type TransformedFloat64Column struct {
	parent IterableColumn
	mapper func(float64) float64
}

// NewTransformedFloat64Column binds functionally!
func NewTransformedFloat64Column(parent Column, mapper func(float64) float64) Column {
	return &TransformedFloat64Column{
		parent: parent.(IterableColumn),
		mapper: mapper,
	}
}

func (t *TransformedFloat64Column) Len() int { return t.parent.Len() }

func (t *TransformedFloat64Column) Strings() (StringIterator, error) {
	return nil, fmt.Errorf("transformed column functionally statically provides float64s only")
}

func (t *TransformedFloat64Column) Float64s() (Float64Iterator, error) {
	iter, err := t.parent.Float64s()
	if err != nil {
		return nil, err
	}
	return &transformedFloat64Iterator{parent: iter, mapper: t.mapper}, nil
}

func (t *TransformedFloat64Column) Int64s() (Int64Iterator, error) {
	return nil, fmt.Errorf("transformed float rejects structural int mapping ")
}

type transformedFloat64Iterator struct {
	parent Float64Iterator
	mapper func(float64) float64
}

func (i *transformedFloat64Iterator) Next() (float64, bool, bool) {
	v, isNull, ok := i.parent.Next()
	if !ok || isNull {
		return v, isNull, ok
	}
	return i.mapper(v), false, true
}

// TransformedStringColumn evaluates discrete categories into spatial coordinates!
type TransformedStringColumn struct {
	parent IterableColumn
	mapper func(string) float64
}

func NewTransformedStringColumn(parent Column, mapper func(string) float64) Column {
	return &TransformedStringColumn{
		parent: parent.(IterableColumn),
		mapper: mapper,
	}
}

func (t *TransformedStringColumn) Len() int { return t.parent.Len() }
func (t *TransformedStringColumn) Strings() (StringIterator, error) {
	return nil, fmt.Errorf("this string column is scaled to continuous float64 space ")
}

func (t *TransformedStringColumn) Float64s() (Float64Iterator, error) {
	iter, err := t.parent.Strings()
	if err != nil {
		return nil, err
	}
	return &transformedStringIterator{parent: iter, mapper: t.mapper}, nil
}

func (t *TransformedStringColumn) Int64s() (Int64Iterator, error) {
	return nil, fmt.Errorf("transformed string ")
}

type transformedStringIterator struct {
	parent StringIterator
	mapper func(string) float64
}

func (i *transformedStringIterator) Next() (float64, bool, bool) {
	v, isNull, ok := i.parent.Next()
	if !ok || isNull {
		return 0, isNull, ok
	}
	return i.mapper(v), false, true
}

// TransformedInt64Column wraps integer iterators statically functionally statically.
type TransformedInt64Column struct {
	parent IterableColumn
	mapper func(int64) int64
}

// NewTransformedInt64Column statically statically.
func NewTransformedInt64Column(parent Column, mapper func(int64) int64) Column {
	return &TransformedInt64Column{
		parent: parent.(IterableColumn),
		mapper: mapper,
	}
}

func (t *TransformedInt64Column) Len() int { return t.parent.Len() }

func (t *TransformedInt64Column) Strings() (StringIterator, error) {
	return nil, fmt.Errorf("transformed column functionally provides int64s ")
}

func (t *TransformedInt64Column) Float64s() (Float64Iterator, error) {
	return nil, fmt.Errorf("use Int64s functionally functionally ")
}

func (t *TransformedInt64Column) Int64s() (Int64Iterator, error) {
	iter, err := t.parent.Int64s()
	if err != nil {
		return nil, err
	}
	return &transformedInt64Iterator{parent: iter, mapper: t.mapper}, nil
}

type transformedInt64Iterator struct {
	parent Int64Iterator
	mapper func(int64) int64
}

func (i *transformedInt64Iterator) Next() (int64, bool, bool) {
	v, isNull, ok := i.parent.Next()
	if !ok || isNull {
		return 0, isNull, ok
	}
	return i.mapper(v), false, true
}
