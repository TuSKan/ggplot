package dataset

// FillDirection specifies the direction for filling missing values.
type FillDirection int

const (
	// FillDown fills missing values with the previous non-null value (carry forward).
	FillDown FillDirection = iota
	// FillUp fills missing values with the next non-null value (carry backward).
	FillUp
)
