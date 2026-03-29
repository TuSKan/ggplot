package dataset

// StringIterator navigates through string/categorical columns sequentially.
type StringIterator interface {
	// Next returns the next string, nullity flag, and false if EOF is reached.
	Next() (val string, isNull bool, ok bool)
}

// IterableColumn exposes typed data loops for mapping/training.
type IterableColumn interface {
	Strings() (StringIterator, error)
}
