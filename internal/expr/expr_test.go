package expr_test

import (
	"fmt"
	"testing"

	"github.com/TuSKan/ggplot/internal/expr"
)

type mockSchema map[string]expr.Type

func (m mockSchema) TypeOf(c string) (expr.Type, error) {
	if t, ok := m[c]; ok {
		return t, nil
	}
	return expr.TypeUnknown, fmt.Errorf("column not found: %s", c)
}

func TestBinaryExpr_TypeValidation(t *testing.T) {
	s := mockSchema{
		"price": expr.TypeFloat64,
		"qty":   expr.TypeInt64,
		"name":  expr.TypeString,
		"valid": expr.TypeBool,
	}

	tests := []struct {
		name     string
		e        expr.Expr
		wantErr  bool
		wantType expr.Type
	}{
		{"float + float", expr.Add(expr.Col("price"), expr.Lit(10.0)), false, expr.TypeFloat64},
		{"int + int", expr.Add(expr.Col("qty"), expr.Lit(10)), false, expr.TypeInt64},
		{"float + int", expr.Add(expr.Col("price"), expr.Col("qty")), false, expr.TypeFloat64},
		{"string == string", expr.Eq(expr.Col("name"), expr.Lit("A")), false, expr.TypeBool},
		{"bool AND bool", expr.And(expr.Col("valid"), expr.Lit(true)), false, expr.TypeBool},
		{"string + float (error)", expr.Add(expr.Col("name"), expr.Col("price")), true, expr.TypeUnknown},
		{"unknown function", expr.Call("rand"), true, expr.TypeUnknown}, // function requires arguments error from our basic dummy mock
		{"missing col", expr.Add(expr.Col("missing"), expr.Lit(1)), true, expr.TypeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, err := tt.e.Type(s)
			if (err != nil) != tt.wantErr {
				t.Errorf("Type() error = %v, wantErr %v", err, tt.wantErr)
			}
			if gotType != tt.wantType {
				t.Errorf("Type() = %v, want %v", gotType, tt.wantType)
			}
			err = tt.e.Validate(s)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExpr_Format(t *testing.T) {
	e := expr.Add(expr.Mul(expr.Col("price"), expr.Col("qty")), expr.Lit(5.5))
	got := e.Format()
	want := `((Col("price") * Col("qty")) + 5.5)`
	if got != want {
		t.Errorf("Format() = %v, want %v", got, want)
	}
}

func TestExpr_IsAggregate(t *testing.T) {
	e1 := expr.Add(expr.Col("a"), expr.Lit(1))
	if e1.IsAggregate() {
		t.Error("e1 should not be aggregate")
	}

	e2 := expr.Agg("sum", expr.Col("a"))
	if !e2.IsAggregate() {
		t.Error("e2 should be aggregate")
	}

	e3 := expr.Add(e2, expr.Lit(1))
	if !e3.IsAggregate() {
		t.Error("e3 should be aggregate because it contains an aggregate child")
	}
}
