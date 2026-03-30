package dataset

// StringIterator navigates through string/categorical columns sequentially.
type StringIterator interface {
	// Next returns the next string, nullity flag, and false if EOF is reached.
	Next() (val string, isNull bool, ok bool)
}

// IterableColumn exposes typed data loops for mapping/training.
type IterableColumn interface {
	Column
	Strings() (StringIterator, error)
	Float64s() (Float64Iterator, error)
	Int64s() (Int64Iterator, error)
}

// Float64Iterator iterates abstract contiguous numeric data.
type Float64Iterator interface {
	Next() (float64, bool, bool)
}

// Int64Iterator functionally functionally statically.
type Int64Iterator interface {
	Next() (int64, bool, bool)
}
