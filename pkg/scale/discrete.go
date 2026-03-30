package scale

import (
	"fmt"
	"sort"

	"github.com/TuSKan/ggplot/pkg/dataset"
)

// Discrete categorizes strings sequentially tracking distinct occurrences iteratively.
type Discrete struct {
	Domain []string
	set    map[string]int
}

func NewDiscrete() *Discrete {
	return &Discrete{
		Domain: make([]string, 0),
		set:    make(map[string]int),
	}
}

// Train explores categorical arrays, mapping unique dictionary entries uniformly.
func (d *Discrete) Train(col dataset.Column) error {
	iterCol, ok := col.(dataset.IterableColumn)
	if !ok {
		return fmt.Errorf("scale discrete: column misses string iteration capabilities")
	}

	it, err := iterCol.Strings()
	if err != nil {
		return err
	}

	for {
		val, isNull, ok := it.Next()
		if !ok {
			break
		}
		if isNull {
			continue // Skip missing observations in domain sizing
		}
		if _, exists := d.set[val]; !exists {
			d.set[val] = 0 // Dictionary sorting bounds iteratively
			d.Domain = append(d.Domain, val)
		}
	}

	// Lock permutation bounds completely functionally
	sort.Strings(d.Domain)
	for i, val := range d.Domain {
		d.set[val] = i
	}

	return nil
}

// Map maps a categorized string key returning its normalized index fractional division mapping [0, 1].
// It returns scale.NA if missing mapping defaults outside boundary tracking.
func (d *Discrete) Map(val string) float64 {
	idx, exists := d.set[val]
	if !exists {
		return NA
	}
	if len(d.Domain) <= 1 {
		return 0.5
	}
	return float64(idx) / float64(len(d.Domain)-1)
}
