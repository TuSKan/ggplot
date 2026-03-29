package dataset

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
	return &GroupByDataset{parent: parent, groups: make(map[string]Dataset)}, nil
}
