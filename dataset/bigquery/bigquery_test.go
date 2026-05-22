package bigquery_test

import (
	"testing"

	"github.com/TuSKan/ggplot/dataset"
	bq "github.com/TuSKan/ggplot/dataset/bigquery"
)

// --- Predicate Expr() ---

func TestCompPredExpr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		pred interface{ Expr() string }
		want string
	}{
		{"Gt", dataset.Gt("x", 5), "`x` > 5"},
		{"Lt", dataset.Lt("y", 3.14), "`y` < 3.14"},
		{"Ge", dataset.Ge("x", 0), "`x` >= 0"},
		{"Le", dataset.Le("x", 100), "`x` <= 100"},
		{"Eq_string", dataset.Eq("name", "Alice"), "`name` = 'Alice'"},
		{"Ne", dataset.Ne("x", 0), "`x` != 0"},
		{"IsNull", dataset.IsNull("x"), "`x` IS NULL"},
		{"IsNotNull", dataset.IsNotNull("x"), "`x` IS NOT NULL"},
		{"Between", dataset.Between("x", 1, 10), "`x` BETWEEN 1 AND 10"},
		{"In", dataset.In("x", 1, 2, 3), "`x` IN (1, 2, 3)"},
		{"And", dataset.And(dataset.Gt("x", 0), dataset.Lt("y", 10)), "(`x` > 0 AND `y` < 10)"},
		{"Or", dataset.Or(dataset.Eq("x", 1), dataset.Eq("x", 2)), "(`x` = 1 OR `x` = 2)"},
		{"Not", dataset.Not(dataset.Eq("x", 0)), "NOT(`x` = 0)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.pred.Expr()
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// --- DType SQL mapping ---

func TestDtypeToBQSQL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		dt   dataset.DType
		want string
	}{
		{dataset.DTypeFloat64, "FLOAT64"},
		{dataset.DTypeInt64, "INT64"},
		{dataset.DTypeString, "STRING"},
		{dataset.DTypeBool, "BOOL"},
		{dataset.DTypeTimestamp, "TIMESTAMP"},
		{dataset.DTypeDate, "DATE"},
		{dataset.DTypeTime, "TIME"},
	}
	for _, tt := range tests {
		got := bq.DtypeToBQSQL(tt.dt)
		if got != tt.want {
			t.Errorf("dtypeToBQSQL(%v) = %q, want %q", tt.dt, got, tt.want)
		}
	}
}
