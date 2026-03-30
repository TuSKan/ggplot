package expr

import (
	"fmt"

	"github.com/TuSKan/ggplot/pkg/dataset"
)

// Evaluator executes the AST against a specific dataset.
// For operations producing new data (arithmetic), materialization is unavoidable without external tools.
// We map these to simple slices which can then wrap back to `dataset.Column`.
type Evaluator struct {
	ds dataset.Dataset
}

func NewEvaluator(ds dataset.Dataset) *Evaluator {
	return &Evaluator{ds: ds}
}

// EvalFloat64 runs scalar float expressions resolving nested values functionally.
func (ev *Evaluator) EvalFloat64(e Expr) ([]float64, error) {
	n := ev.ds.Len()
	res := make([]float64, n)

	switch node := e.(type) {
	case *LiteralExpr:
		val, ok := node.Value.(float64)
		if !ok {
			if vInt, ok := node.Value.(int64); ok {
				val = float64(vInt)
			} else {
				return nil, fmt.Errorf("literal not a float64")
			}
		}
		for i := 0; i < n; i++ {
			res[i] = val
		}
		return res, nil

	case *ColumnRef:
		// Attempting native extraction for fallback generic mapping
		return nil, fmt.Errorf("direct column float materialize requires native generic extractor wrapper")

	case *BinaryExpr:
		left, err := ev.EvalFloat64(node.Left)
		if err != nil {
			return nil, err
		}
		right, err := ev.EvalFloat64(node.Right)
		if err != nil {
			return nil, err
		}

		switch node.Op {
		case OpAdd:
			for i := 0; i < n; i++ {
				res[i] = left[i] + right[i]
			}
		case OpSub:
			for i := 0; i < n; i++ {
				res[i] = left[i] - right[i]
			}
		case OpMul:
			for i := 0; i < n; i++ {
				res[i] = left[i] * right[i]
			}
		case OpDiv:
			for i := 0; i < n; i++ {
				if right[i] == 0 {
					return nil, fmt.Errorf("division by zero at row %d", i)
				}
				res[i] = left[i] / right[i]
			}
		default:
			return nil, fmt.Errorf("unsupported binary operator for float evaluation: %s", node.Op)
		}
		return res, nil
	}

	return nil, fmt.Errorf("unsupported expression node structure ")
}
