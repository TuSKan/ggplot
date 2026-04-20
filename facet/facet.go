// Package facet splits a dataset into subsets for "small multiple" panel layouts.
// Faceting is a core component of the Grammar of Graphics, allowing the same
// plot specification to be repeated across levels of a categorical variable.
package facet

import (
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
	col, err := ds.Column(f.col)
	if err != nil {
		return nil, err
	}

	iter, ok := col.(dataset.IterableColumn)
	if !ok {
		return nil, &dataset.ErrColumnNotFound{Name: f.col}
	}

	sit, err := iter.Strings()
	if err != nil {
		return nil, err
	}

	// Collect group memberships.
	n := ds.Len()
	groupMasks := make(map[string][]bool)
	var order []string

	for i := 0; i < n; i++ {
		v, isNull, ok := sit.Next()
		if !ok {
			break
		}
		if isNull {
			v = "NA"
		}
		if _, exists := groupMasks[v]; !exists {
			groupMasks[v] = make([]bool, n)
			order = append(order, v)
		}
		groupMasks[v][i] = true
	}

	panels := make([]Panel, 0, len(order))
	for _, label := range order {
		panels = append(panels, Panel{
			Label:   label,
			Dataset: dataset.FilterMask(ds, groupMasks[label]),
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

	n := ds.Len()
	panels := make([]Panel, 0, len(rowVals)*len(colVals))

	for _, rv := range rowVals {
		for _, cv := range colVals {
			mask := make([]bool, n)
			// Read both columns to build mask.
			rc, _ := ds.Column(g.rowCol)
			cc, _ := ds.Column(g.colCol)
			ri, _ := rc.(dataset.IterableColumn).Strings()
			ci, _ := cc.(dataset.IterableColumn).Strings()

			for i := 0; i < n; i++ {
				rval, _, _ := ri.Next()
				cval, _, _ := ci.Next()
				mask[i] = rval == rv && cval == cv
			}

			panels = append(panels, Panel{
				Label:   rv + " | " + cv,
				Dataset: dataset.FilterMask(ds, mask),
			})
		}
	}
	return panels, nil
}

func (g *gridFacet) GridDims(nPanels int) (int, int) {
	// Grid dimensions: infer from panel count assuming square-ish layout.
	// The actual layout is nRowVals × nColVals, but since we only get nPanels here,
	// we compute the best factorization.
	if nPanels <= 0 {
		return 1, 1
	}
	cols := ceilSqrt(nPanels)
	rows := (nPanels + cols - 1) / cols
	return rows, cols
}
func (g *gridFacet) String() string { return "grid(" + g.rowCol + " ~ " + g.colCol + ")" }

// --- Helpers ---

func distinctStrings(ds dataset.Dataset, col string) ([]string, error) {
	c, err := ds.Column(col)
	if err != nil {
		return nil, err
	}
	iter, ok := c.(dataset.IterableColumn)
	if !ok {
		return nil, &dataset.ErrColumnNotFound{Name: col}
	}
	sit, err := iter.Strings()
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	var order []string
	for {
		v, isNull, ok := sit.Next()
		if !ok {
			break
		}
		if isNull {
			v = "NA"
		}
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
