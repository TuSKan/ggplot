package dataset

import (
	"fmt"
)

// Dataset represents a declarative, lazy-evaluating columnar source.
type Dataset interface {
	// Columns returns the available column names.
	Columns() []string

	// Column fetches the abstract column data block.
	Column(name string) (Column, error)

	// Len returns the logical number of rows.
	Len() int
}

// Column represents an immutable 1D data sequence.
// An abstraction that does not bind directly to backend types, but leaves them
// open for generic or type-asserted queries.
type Column interface {
	Len() int
}

// Slice creates a zero-copy partial view of the given dataset.
// If the underlying dataset implements NativeSliceProvider, it delegates.
func Slice(d Dataset, i, j int) Dataset {
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

	return &sliceDataset{
		parent: d,
		offset: i,
		length: j - i,
	}
}

// NativeSliceProvider allows datasets (like Arrow implementations) to slice themselves.
type NativeSliceProvider interface {
	SliceDataset(i, j int) Dataset
}

type sliceDataset struct {
	parent Dataset
	offset int
	length int
}

func (s *sliceDataset) Columns() []string { return s.parent.Columns() }
func (s *sliceDataset) Len() int          { return s.length }

func (s *sliceDataset) Column(name string) (Column, error) {
	col, err := s.parent.Column(name)
	if err != nil {
		return nil, err
	}
	if p, ok := col.(NativeColumnSliceProvider); ok {
		if nativeCol := p.SliceColumn(s.offset, s.offset+s.length); nativeCol != nil {
			return nativeCol, nil
		}
	}
	return &sliceColumn{parent: col, offset: s.offset, length: s.length}, nil
}

// NativeColumnSliceProvider allows a column to provide a zero-copy representation of its slice.
type NativeColumnSliceProvider interface {
	SliceColumn(i, j int) Column
}

type sliceColumn struct {
	parent Column
	offset int
	length int
}

func (c *sliceColumn) Len() int { return c.length }

// ErrColumnNotFound is a typed error for when a dataset lacks a requested column.
type ErrColumnNotFound struct {
	Name string
}

func (e *ErrColumnNotFound) Error() string {
	return fmt.Sprintf("dataset: column %q not found", e.Name)
}
