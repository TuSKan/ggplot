package stat

// Stat computes statistical transformations on datasets (e.g. binning, smoothing).
type Stat interface {
	// Compute transforms the incoming data structure.
	Compute() error
}
