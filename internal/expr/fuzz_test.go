package expr

import (
	"fmt"
	"testing"
)

// FuzzExpressionAST verifies syntactic bounds mapping randomly!
func FuzzExpressionAST(f *testing.F) {
	// Seed representative valid & edge case expressions
	f.Add("age", 50.0, true)
	f.Add("income", 1000.0, false)
	f.Add("", 0.0, true)
	f.Add("score", -5.5, false)

	f.Fuzz(func(t *testing.T, colName string, litVal float64, useAdd bool) {
		// Construct dynamic AST
		col := Col(colName)
		lit := Lit(litVal)

		var astExpr Expr
		if useAdd {
			astExpr = Add(col, lit)
		} else {
			astExpr = Gt(col, lit)
		}

		schema := DummySchema{cols: map[string]Type{colName: TypeFloat64}}

		_ = astExpr.Validate(schema)
		_, _ = astExpr.Type(schema)
		_ = astExpr.Format()
	})
}

type DummySchema struct {
	cols map[string]Type
}

func (d DummySchema) TypeOf(col string) (Type, error) {
	if t, ok := d.cols[col]; ok {
		return t, nil
	}
	return TypeUnknown, fmt.Errorf("missing ")
}
func (d DummySchema) Columns() []string { return []string{} }
