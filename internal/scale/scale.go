package scale

import "github.com/TuSKan/ggplot/internal/dataset"

// Scale defines constraints determining how dataspaces bounds visual aesthetics structurally.
type Scale interface {
	Train(col dataset.Column) error
}

// Compile Time Interface proofs
var _ Scale = (*Continuous)(nil)
var _ Scale = (*Discrete)(nil)
