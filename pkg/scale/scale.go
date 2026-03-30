package scale

import "github.com/TuSKan/ggplot/pkg/dataset"

// Scale defines constraints determining how dataspaces bounds visual aesthetics.
type Scale interface {
	Train(col dataset.Column) error
}

// Manager resolves aesthetic bindings into concrete Scale limits.
type Manager interface {
	GetScale(aesthetic string) (Scale, bool)
	GetContinuous(aesthetic string) (ContinuousMapper, bool)
	GetDiscrete(aesthetic string) (DiscreteMapper, bool)
}

// ContinuousMapper maps a float64 value into a normalized visual bound [0, 1].
type ContinuousMapper interface {
	Scale
	Map(val float64) float64
}

// DiscreteMapper maps a string category into a normalized visual bound [0, 1].
type DiscreteMapper interface {
	Scale
	Map(val string) float64
}

// TemporalMapper maps an int64 timestamp into a normalized visual bound [0, 1].
type TemporalMapper interface {
	Scale
	Map(val int64) float64
}

// Compile Time Interface proofs
var _ ContinuousMapper = (*Continuous)(nil)
var _ DiscreteMapper = (*Discrete)(nil)
