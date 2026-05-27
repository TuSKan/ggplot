// Package facet splits a dataset into subsets for "small multiple" panel layouts.
// Faceting is a core component of the Grammar of Graphics, allowing the same
// plot specification to be repeated across levels of a categorical variable.
package facet

import (
	"context"
	"fmt"
	"strconv"

	"github.com/TuSKan/ggplot/dataset"
)

// ---------------------------------------------------------------------------
// Labeller
// ---------------------------------------------------------------------------

// Labeller formats facet panel strip labels. It receives the faceting
// variable name and the panel's raw value, returning the display string.
type Labeller func(variable, value string) string

// LabelValue returns a labeller that shows only the value.
// This is the default labeller.
//
//	"setosa", "versicolor", "virginica"
func LabelValue() Labeller {
	return func(_, value string) string { return value }
}

// LabelBoth returns a labeller that shows "variable: value".
//
//	"species: setosa", "species: versicolor"
func LabelBoth() Labeller {
	return func(variable, value string) string {
		return variable + ": " + value
	}
}

// LabelContext returns a labeller that shows "variable: value" for Grid
// facets (where context is ambiguous) and just "value" for Wrap facets.
// The choice is made internally by the facet type at Split time.
func LabelContext() Labeller {
	return nil // sentinel — Split() checks for nil and resolves per facet type.
}

// Label returns a custom labeller from a user-supplied function.
func Label(fn func(variable, value string) string) Labeller {
	return fn
}

// defaultLabeller returns LabelValue when l is nil.
func defaultLabeller(l Labeller) Labeller {
	if l == nil {
		return LabelValue()
	}

	return l
}

// ---------------------------------------------------------------------------
// Facet interface
// ---------------------------------------------------------------------------

// Facet defines how a dataset is split into panels for small-multiple layouts.
type Facet interface {
	// Split partitions the dataset into panels. Each panel has a label
	// and a filtered subset of the data.
	Split(ctx context.Context, ds dataset.Dataset) ([]Panel, error)

	// GridDims returns the (rows, cols) grid dimensions for layout.
	// For Wrap, this is computed from the number of panels and nCols.
	GridDims(nPanels int) (rows, cols int)

	// String returns a human-readable description.
	String() string
}

// Panel represents a single facet panel containing a data subset.
type Panel struct {
	Label    string // formatted display label
	RowVal   string // raw row facet value (Grid only, "" for Wrap)
	ColVal   string // raw column facet value (Grid only, "" for Wrap)
	Dataset  dataset.Dataset
	NumRows  int  // number of rows matching this panel's filter
	IsMargin bool // true for aggregate margin panels
}

// MarginLabel is the display value used for aggregate margin panels.
const MarginLabel = "All"

// --- No faceting ---

// None returns a no-op facet that produces a single panel.
func None() Facet { return none{} }

type none struct{}

func (none) Split(_ context.Context, ds dataset.Dataset) ([]Panel, error) {
	return []Panel{{Label: "", Dataset: ds, NumRows: int(ds.NumRows())}}, nil
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

// WithLabeller sets the label formatter for Wrap facets.
func WithLabeller(l Labeller) WrapOpt {
	return func(f *wrapFacet) { f.labeller = l }
}

// WithDrop controls whether empty panels are omitted (default: true).
// When false, panels with zero rows are emitted for all distinct values.
func WithDrop(drop bool) WrapOpt {
	return func(f *wrapFacet) { f.drop = drop }
}

// Wrap creates a facet that wraps panels across a grid layout, splitting
// the data by the given column's distinct values.
func Wrap(col string, opts ...WrapOpt) Facet {
	f := &wrapFacet{col: col, drop: true}
	for _, opt := range opts {
		opt(f)
	}

	return f
}

type wrapFacet struct {
	col      string
	nCols    int
	nRows    int
	labeller Labeller
	drop     bool
}

func (f *wrapFacet) Split(_ context.Context, ds dataset.Dataset) ([]Panel, error) {
	vals, err := facetStrings(ds, f.col)
	if err != nil {
		return nil, err
	}

	// Resolve labeller. LabelContext sentinel (nil) → LabelValue for Wrap.
	lbl := defaultLabeller(f.labeller)

	n := len(vals)
	groupMasks := make(map[string][]bool)

	var order []string

	for i := range n {
		v := vals[i]
		if _, exists := groupMasks[v]; !exists {
			groupMasks[v] = make([]bool, n)
			order = append(order, v)
		}

		groupMasks[v][i] = true
	}

	panels := make([]Panel, 0, len(order))
	for _, rawVal := range order {
		mask := groupMasks[rawVal]

		// Skip empty panels when drop=true.
		if f.drop && !maskHasTrue(mask) {
			continue
		}

		panels = append(panels, Panel{
			Label:   lbl(f.col, rawVal),
			Dataset: ds.Filter(dataset.BoolMask(mask)),
			NumRows: maskCount(mask),
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

// GridOpt configures a Grid facet.
type GridOpt func(*gridFacet)

// GridLabeller sets the label formatter for Grid facets.
func GridLabeller(l Labeller) GridOpt {
	return func(g *gridFacet) { g.labeller = l }
}

// GridDrop controls whether empty panels are omitted (default: true).
func GridDrop(drop bool) GridOpt {
	return func(g *gridFacet) { g.drop = drop }
}

// GridMargins enables aggregate margin panels (default: false).
// When true, extra panels are added that aggregate across row/column values.
func GridMargins(margins bool) GridOpt {
	return func(g *gridFacet) { g.margins = margins }
}

// Grid creates a facet that arranges panels in a row × col matrix,
// splitting by one variable for rows and another for columns.
func Grid(rowCol, colCol string, opts ...GridOpt) Facet {
	g := &gridFacet{rowCol: rowCol, colCol: colCol, drop: true}
	for _, opt := range opts {
		opt(g)
	}

	return g
}

type gridFacet struct {
	rowCol   string
	colCol   string
	nRowVals int // set by Split, used by GridDims
	nColVals int // set by Split, used by GridDims
	labeller Labeller
	drop     bool
	margins  bool
}

func (g *gridFacet) Split(_ context.Context, ds dataset.Dataset) ([]Panel, error) {
	rStrings, err := facetStrings(ds, g.rowCol)
	if err != nil {
		return nil, err
	}

	cStrings, err := facetStrings(ds, g.colCol)
	if err != nil {
		return nil, err
	}

	// Resolve labeller. LabelContext sentinel (nil) → LabelBoth for Grid.
	lbl := g.labeller
	if lbl == nil {
		lbl = LabelBoth()
	}

	// Compute distinct values preserving order.
	rowVals := distinctOrdered(rStrings)
	colVals := distinctOrdered(cStrings)

	n := len(rStrings)

	// Build data panels.
	panels := g.buildDataPanels(ds, rStrings, cStrings, rowVals, colVals, n, lbl)

	// Grid dimensions for data panels only.
	nDataRows := len(rowVals)
	nDataCols := len(colVals)

	// Add margin panels if requested.
	if g.margins {
		marginPanels := g.buildMarginPanels(ds, rStrings, cStrings, rowVals, colVals, n, lbl)
		panels = append(panels, marginPanels...)
		nDataRows++
		nDataCols++
	}

	g.nRowVals = nDataRows
	g.nColVals = nDataCols

	return panels, nil
}

// buildDataPanels creates panels for each (row, col) combination.
func (g *gridFacet) buildDataPanels(
	ds dataset.Dataset,
	rStrings, cStrings []string,
	rowVals, colVals []string,
	n int, lbl Labeller,
) []Panel {
	panels := make([]Panel, 0, len(rowVals)*len(colVals))

	for _, rv := range rowVals {
		for _, cv := range colVals {
			mask := make([]bool, n)
			for i := range n {
				mask[i] = rStrings[i] == rv && cStrings[i] == cv
			}

			if g.drop && !maskHasTrue(mask) {
				continue
			}

			panels = append(panels, Panel{
				Label:   lbl(g.rowCol, rv) + " | " + lbl(g.colCol, cv),
				RowVal:  rv,
				ColVal:  cv,
				Dataset: ds.Filter(dataset.BoolMask(mask)),
				NumRows: maskCount(mask),
			})
		}
	}

	return panels
}

// buildMarginPanels creates aggregate panels for margins.
// Row margins: aggregate across columns (one per row value).
// Column margins: aggregate across rows (one per col value).
// Corner margin: full dataset.
func (g *gridFacet) buildMarginPanels(
	ds dataset.Dataset,
	rStrings, cStrings []string,
	rowVals, colVals []string,
	n int, lbl Labeller,
) []Panel {
	panels := make([]Panel, 0, len(rowVals)+len(colVals)+1)

	// Row margins: one panel per row value, aggregated across all columns.
	for _, rv := range rowVals {
		mask := make([]bool, n)
		for i := range n {
			mask[i] = rStrings[i] == rv
		}

		panels = append(panels, Panel{
			Label:    lbl(g.rowCol, rv) + " | " + MarginLabel,
			RowVal:   rv,
			ColVal:   MarginLabel,
			Dataset:  ds.Filter(dataset.BoolMask(mask)),
			NumRows:  maskCount(mask),
			IsMargin: true,
		})
	}

	// Column margins: one panel per col value, aggregated across all rows.
	for _, cv := range colVals {
		mask := make([]bool, n)
		for i := range n {
			mask[i] = cStrings[i] == cv
		}

		panels = append(panels, Panel{
			Label:    MarginLabel + " | " + lbl(g.colCol, cv),
			RowVal:   MarginLabel,
			ColVal:   cv,
			Dataset:  ds.Filter(dataset.BoolMask(mask)),
			NumRows:  maskCount(mask),
			IsMargin: true,
		})
	}

	// Corner margin: full dataset (already materialized by pipeline).
	panels = append(panels, Panel{
		Label:    MarginLabel + " | " + MarginLabel,
		RowVal:   MarginLabel,
		ColVal:   MarginLabel,
		Dataset:  ds,
		NumRows:  int(ds.NumRows()),
		IsMargin: true,
	})

	return panels
}

func (g *gridFacet) GridDims(nPanels int) (int, int) {
	// Use actual row × col cardinalities from Split when available.
	if g.nRowVals > 0 && g.nColVals > 0 {
		return g.nRowVals, g.nColVals
	}
	// Fallback (shouldn't happen — Split is always called before GridDims).
	if nPanels <= 0 {
		return 1, 1
	}

	cols := ceilSqrt(nPanels)
	rows := (nPanels + cols - 1) / cols

	return rows, cols
}
func (g *gridFacet) String() string { return "grid(" + g.rowCol + " ~ " + g.colCol + ")" }

// --- Helpers ---

// facetStrings extracts a string slice from a column, supporting string,
// float64, int64, and bool columns via fmt.Sprintf conversion.
func facetStrings(ds dataset.Dataset, col string) ([]string, error) {
	c, err := ds.Column(col)
	if err != nil {
		return nil, fmt.Errorf("facet: %w", err)
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
			out[i] = strconv.FormatInt(v, 10)
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
		return nil, fmt.Errorf("facet: unsupported column type %T for %q: %w", c, col, ErrFacetConfig)
	}

	return out, nil
}

// distinctOrdered returns distinct values preserving first-occurrence order.
func distinctOrdered(vals []string) []string {
	seen := make(map[string]struct{})

	var order []string

	for _, v := range vals {
		if _, exists := seen[v]; !exists {
			seen[v] = struct{}{}
			order = append(order, v)
		}
	}

	return order
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

// maskHasTrue returns true if any element in the mask is true.
func maskHasTrue(mask []bool) bool {
	for _, v := range mask {
		if v {
			return true
		}
	}

	return false
}

// maskCount returns the number of true values in the mask.
func maskCount(mask []bool) int {
	n := 0

	for _, v := range mask {
		if v {
			n++
		}
	}

	return n
}
