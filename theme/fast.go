package theme

func init() { MustRegister(Fast, newFast) }

// newFast mirrors matplotlib's fast.mplstyle. Upstream this style only
// toggles path-simplification rcParams (path.simplify, agg.path.chunksize)
// for performance; it does not change visual chrome. We have no
// path-simplification analog in our renderer, so the preset exists so
// users can write Theme("fast") without an error and inherits the
// default ggplot look.
//
// Source: matplotlib/lib/matplotlib/mpl-data/stylelib/fast.mplstyle
func newFast() Theme {
	t := newGgplot()
	t.Name = "fast"
	return t
}
