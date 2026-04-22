package dataset

// Masker describes a row-level filter condition that can be lazily
// evaluated against a dataset to produce a boolean mask.
type Masker interface {
	// Mask computes a boolean mask of length int(ds.NumRows()). True entries are kept.
	Mask(ds Dataset) ([]bool, error)
}
