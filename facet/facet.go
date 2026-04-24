// Package facet splits a dataset into subsets for "small multiple" panel layouts.
// Faceting is a core component of the Grammar of Graphics, allowing the same
// plot specification to be repeated across levels of a categorical variable.
package facet

import (
	"fmt"

	"github.com/TuSKan/ggplot/dataset"
)

// Facet defines how a dataset is split into panels for small-multiple layouts.
type Facet interface {
	// Split partitions the dataset into panels. Each panel has a label
	// and a filtered subset of the data.
	Split(ds dataset.Dataset) ([]Panel, error)

	// GridDims returns the (rows, cols) grid dimensions for layout.
	// For Wrap, this is computed from the number of panels and nCols.
	GridDims(nPanels int) (rows, cols int)

	// String returns a human-readable description.
	String() string
}

// Panel represents a single facet panel containing a data subset.
type Panel struct {
	Label   string
	Dataset dataset.Dataset
}

// --- No faceting ---

// None returns a no-op facet that produces a single panel.
func None() Facet { return none{} }

type none struct{}

func (none) Split(ds dataset.Dataset) ([]Panel, error) {
	return []Panel{{Label: "", Dataset: ds}}, nil
}
func (none) GridDims(int) (int, int) { return 1, 1 }
func (none) String() string          { return "none" }

// --- Wrap faceting ---

// WrapOpt configures a Wrap facet.
type WrapOpt func(*wrapFacet)

// NCols sets the number of columns for the wrapped layout.
func NCols(n int) WrapOpt { return func(f *wrapFacet) { f.nCols = n } }

// NRows sets the number of rows for the wrapped layout.
func NRows(n int) WrapOpt { return func(f *wrapFacet) { f.nRows = n } }

// Wrap creates a facet that wraps panels across a grid layout, splitting
// the data by the given column's distinct values.
func Wrap(col string, opts ...WrapOpt) Facet {
	f := &wrapFacet{col: col, nCols: 0, nRows: 0}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

type wrapFacet struct {
	col   string
	nCols int
	nRows int
}

func (f *wrapFacet) Split(ds dataset.Dataset) ([]Panel, error) {
	vals, err := columnStrings(ds, f.col)
	if err != nil {
		return nil, err
	}

	n := len(vals)
	groupMasks := make(map[string][]bool)
	var order []string

	for i := 0; i < n; i++ {
		v := vals[i]
		if _, exists := groupMasks[v]; !exists {
			groupMasks[v] = make([]bool, n)
			order = append(order, v)
		}
		groupMasks[v][i] = true
	}

	panels := make([]Panel, 0, len(order))
	for _, label := range order {
		filtered := ds.Filter(dataset.BoolMask(groupMasks[label]))
		if filtered.Err() != nil {
			return nil, filtered.Err()
		}
		panels = append(panels, Panel{
			Label:   label,
			Dataset: filtered,
		})
	}
	return panels, nil
}

func (f *wrapFacet) GridDims(nPanels int) (int, int) {
	cols := f.nCols
	rows := f.nRows

	if cols > 0 && rows > 0 {
		return rows, cols
	}
	if cols > 0 {
		rows = (nPanels + cols - 1) / cols
		return rows, cols
	}
	if rows > 0 {
		cols = (nPanels + rows - 1) / rows
		return rows, cols
	}

	// Auto: prefer roughly square layouts.
	cols = ceilSqrt(nPanels)
	rows = (nPanels + cols - 1) / cols
	return rows, cols
}

func (f *wrapFacet) String() string { return "wrap(" + f.col + ")" }

// --- Grid faceting ---

// Grid creates a facet that arranges panels in a row × col matrix,
// splitting by one variable for rows and another for columns.
func Grid(rowCol, colCol string) Facet {
	return &gridFacet{rowCol: rowCol, colCol: colCol}
}

type gridFacet struct {
	rowCol string
	colCol string
}

func (g *gridFacet) Split(ds dataset.Dataset) ([]Panel, error) {
	// Extract distinct values for row and column facet variables.
	rowVals, err := distinctStrings(ds, g.rowCol)
	if err != nil {
		return nil, err
	}
	colVals, err := distinctStrings(ds, g.colCol)
	if err != nil {
		return nil, err
	}

	rStrings, _ := columnStrings(ds, g.rowCol)
	cStrings, _ := columnStrings(ds, g.colCol)
	n := len(rStrings)

	panels := make([]Panel, 0, len(rowVals)*len(colVals))

	for _, rv := range rowVals {
		for _, cv := range colVals {
			mask := make([]bool, n)
			for i := 0; i < n; i++ {
				mask[i] = rStrings[i] == rv && cStrings[i] == cv
			}

			filtered := ds.Filter(dataset.BoolMask(mask))
			if filtered.Err() != nil {
				return nil, filtered.Err()
			}
			panels = append(panels, Panel{
				Label:   rv + " | " + cv,
				Dataset: filtered,
			})
		}
	}
	return panels, nil
}

func (g *gridFacet) GridDims(nPanels int) (int, int) {
	if nPanels <= 0 {
		return 1, 1
	}
	cols := ceilSqrt(nPanels)
	rows := (nPanels + cols - 1) / cols
	return rows, cols
}
func (g *gridFacet) String() string { return "grid(" + g.rowCol + " ~ " + g.colCol + ")" }

// --- Helpers ---

// columnStrings extracts a string slice from a column, supporting string and
// float64/int64 columns via fmt.Sprintf conversion.
func columnStrings(ds dataset.Dataset, col string) ([]string, error) {
	c, err := ds.Column(col)
	if err != nil {
		return nil, err
	}
	n := int(c.Len())
	out := make([]string, n)

	switch tc := c.(type) {
	case dataset.Column[string]:
		copy(out, tc.Values())
	case dataset.Column[float64]:
		for i, v := range tc.Values() {
			out[i] = fmt.Sprintf("%g", v)
		}
	case dataset.Column[int64]:
		for i, v := range tc.Values() {
			out[i] = fmt.Sprintf("%d", v)
		}
	case dataset.Column[bool]:
		for i, v := range tc.Values() {
			if v {
				out[i] = "TRUE"
			} else {
				out[i] = "FALSE"
			}
		}
	default:
		return nil, fmt.Errorf("facet: unsupported column type %T for %q", c, col)
	}
	return out, nil
}

func distinctStrings(ds dataset.Dataset, col string) ([]string, error) {
	vals, err := columnStrings(ds, col)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{})
	var order []string
	for _, v := range vals {
		if _, exists := seen[v]; !exists {
			seen[v] = struct{}{}
			order = append(order, v)
		}
	}
	return order, nil
}

func ceilSqrt(n int) int {
	if n <= 0 {
		return 1
	}
	s := 1
	for s*s < n {
		s++
	}
	return s
}
