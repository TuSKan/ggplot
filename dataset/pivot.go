package dataset

// PivotLonger and PivotWider reshape datasets between wide and long formats.
// These are the Go equivalents of tidyr::pivot_longer and tidyr::pivot_wider.

// PivotLongerSpec configures a PivotLonger operation.
type PivotLongerSpec struct {
	// Cols are the column names to pivot from wide to long format.
	// These columns are "gathered" into a single name+value pair.
	Cols []string
	// NamesTo is the output column name that will hold the original column names.
	NamesTo string
	// ValuesTo is the output column name that will hold the values.
	ValuesTo string
}

// PivotWiderSpec configures a PivotWider operation.
type PivotWiderSpec struct {
	// NamesFrom is the column whose unique values become new column names.
	NamesFrom string
	// ValuesFrom is the column whose values fill the new columns.
	ValuesFrom string
}
