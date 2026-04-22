package dataset

// JoinType identifies the kind of join to perform.
type JoinType int

const (
	// JoinLeft keeps all rows from the left dataset; unmatched right rows are null-filled.
	JoinLeft JoinType = iota
	// JoinRight keeps all rows from the right dataset; unmatched left rows are null-filled.
	JoinRight
	// JoinInner keeps only rows with matches in both datasets.
	JoinInner
	// JoinFull keeps all rows from both datasets; unmatched sides are null-filled.
	JoinFull
	// JoinSemi keeps rows from the left that have at least one match in the right.
	// No columns from the right are included.
	JoinSemi
	// JoinAnti keeps rows from the left that have NO match in the right.
	// No columns from the right are included.
	JoinAnti
)

// JoinSpec describes how to match rows between two datasets.
type JoinSpec struct {
	Type      JoinType
	LeftCols  []string
	RightCols []string
}

// On creates a JoinSpec matching on columns with the same name in both datasets.
func On(cols ...string) JoinSpec {
	return JoinSpec{LeftCols: cols, RightCols: cols}
}
