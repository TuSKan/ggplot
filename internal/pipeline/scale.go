package pipeline

import (
	"github.com/TuSKan/ggplot/internal/ast"
	"github.com/TuSKan/ggplot/internal/expr"
	"github.com/TuSKan/ggplot/pkg/scale"
	"github.com/TuSKan/ggplot/pkg/dataset"
)

// TrainScales iterates the mapping resolving columns against the underlying dataset
// to train standard explicit scale implementations.
func TrainScales(plan *ast.RenderPlan, ds dataset.Dataset, mappings []ast.Aes) {
	for _, m := range mappings {
		for key, e := range m {
			if colRef, ok := e.(*expr.ColumnRef); ok {
				colName := colRef.Name
				col, err := ds.Column(colName)
				if err != nil {
					continue
				}

				var s scale.Scale
				if existing, found := plan.Scales[key]; found {
					s = existing.(scale.Scale)
					s.Train(col)
				} else {
					cs := scale.NewContinuous()
					if err := cs.Train(col); err == nil {
						plan.Scales[key] = cs
					} else {
						d := scale.NewDiscrete()
						d.Train(col)
						plan.Scales[key] = d
					}
				}
			}
		}
	}
}
