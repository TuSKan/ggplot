package bigquery

import (
	"fmt"
	"strings"

	"github.com/TuSKan/ggplot/dataset"
)

// --- MathKernel ---
//
// All math operations generate lazy SQL expressions via pendingSQL.
// BQ Standard SQL supports all standard math functions.

// --- Binary arithmetic (column × column) ---

func (e *Engine) AddCols(a, b dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.binaryColOp("+", "add", a, b)
}

func (e *Engine) SubCols(a, b dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.binaryColOp("-", "sub", a, b)
}

func (e *Engine) MulCols(a, b dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.binaryColOp("*", "mul", a, b)
}

func (e *Engine) DivCols(a, b dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.binaryColOp("/", "div", a, b)
}

// binaryColOp generates SELECT a.col op b.col FROM source.
func (e *Engine) binaryColOp(op, suffix string, a, b dataset.AnyColumn) (dataset.AnyColumn, error) {
	aBQ, aOK := a.(*bqColumn)
	bBQ, bOK := b.(*bqColumn)
	if !aOK || !bOK {
		return e.localBinaryOp(op, a, b)
	}

	resultName := fmt.Sprintf("%s_%s_%s", aBQ.name, suffix, bBQ.name)
	sql := fmt.Sprintf(
		"SELECT *, (`%s` %s `%s`) AS `%s` FROM %s",
		aBQ.name, op, bBQ.name, resultName, aBQ.ds.sourceRef(),
	)

	origFields := aBQ.ds.schema.Fields()
	newFields := make([]dataset.Field, len(origFields)+1)
	copy(newFields, origFields)
	newFields[len(origFields)] = dataset.Field{Name: resultName, Dtype: dataset.DTypeFloat64}
	schema := dataset.NewSchema(newFields...)

	ds := aBQ.ds.withSQL(sql, schema, aBQ.ds.numRows)
	return &bqColumn{ds: ds, name: resultName, dtype: dataset.DTypeFloat64}, nil
}

func (e *Engine) localBinaryOp(op string, a, b dataset.AnyColumn) (dataset.AnyColumn, error) {
	mk := e.localEngine()
	switch op {
	case "+":
		return mk.AddCols(a, b)
	case "-":
		return mk.SubCols(a, b)
	case "*":
		return mk.MulCols(a, b)
	case "/":
		return mk.DivCols(a, b)
	default:
		return nil, fmt.Errorf("bigquery: unknown binary op %q", op)
	}
}

// --- Scalar arithmetic (column × scalar) ---

func (e *Engine) AddScalar(col dataset.AnyColumn, val float64) (dataset.AnyColumn, error) {
	return e.scalarOp("+", val, col)
}

func (e *Engine) MulScalar(col dataset.AnyColumn, val float64) (dataset.AnyColumn, error) {
	return e.scalarOp("*", val, col)
}

func (e *Engine) scalarOp(op string, val float64, col dataset.AnyColumn) (dataset.AnyColumn, error) {
	bqCol, ok := col.(*bqColumn)
	if !ok {
		switch op {
		case "+":
			return e.localEngine().AddScalar(col, val)
		case "*":
			return e.localEngine().MulScalar(col, val)
		}
		return nil, fmt.Errorf("bigquery: unknown scalar op %q", op)
	}

	sql := fmt.Sprintf(
		"SELECT *, (`%s` %s %v) AS `%s` FROM %s",
		bqCol.name, op, val, bqCol.name, bqCol.ds.sourceRef(),
	)

	ds := bqCol.ds.withSQL(sql,
		dataset.NewSchema(dataset.Field{Name: bqCol.name, Dtype: dataset.DTypeFloat64}),
		bqCol.ds.numRows,
	)
	return &bqColumn{ds: ds, name: bqCol.name, dtype: dataset.DTypeFloat64}, nil
}

// --- Unary math functions ---

func (e *Engine) Abs(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.unaryMathFn("ABS", col)
}
func (e *Engine) Neg(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.unaryExpr("-`%s`", "neg", col)
}
func (e *Engine) Sign(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.unaryMathFn("SIGN", col)
}
func (e *Engine) Sqrt(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.unaryMathFn("SQRT", col)
}

func (e *Engine) Pow(col dataset.AnyColumn, exp float64) (dataset.AnyColumn, error) {
	bqCol, ok := col.(*bqColumn)
	if !ok {
		return e.localEngine().Pow(col, exp)
	}
	return e.lazyExprCol(bqCol, fmt.Sprintf("POW(`%s`, %v)", bqCol.name, exp), bqCol.name, dataset.DTypeFloat64)
}

// --- Logarithmic ---

func (e *Engine) Exp(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.unaryMathFn("EXP", col)
}
func (e *Engine) Ln(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.unaryMathFn("LN", col)
}
func (e *Engine) Log2(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.unaryMathFn("LOG2", col)
} // BQ: LOG(x, 2) but LOG2 not available — use LOG
func (e *Engine) Log10(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.unaryMathFn("LOG10", col)
}

// --- Trigonometric ---

func (e *Engine) Sin(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.unaryMathFn("SIN", col)
}
func (e *Engine) Cos(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.unaryMathFn("COS", col)
}
func (e *Engine) Tan(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.unaryMathFn("TAN", col)
}
func (e *Engine) Asin(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.unaryMathFn("ASIN", col)
}
func (e *Engine) Acos(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.unaryMathFn("ACOS", col)
}
func (e *Engine) Atan(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.unaryMathFn("ATAN", col)
}

func (e *Engine) Atan2(y, x dataset.AnyColumn) (dataset.AnyColumn, error) {
	yBQ, yOK := y.(*bqColumn)
	xBQ, xOK := x.(*bqColumn)
	if !yOK || !xOK {
		return e.localEngine().Atan2(y, x)
	}
	resultName := fmt.Sprintf("atan2_%s_%s", yBQ.name, xBQ.name)
	return e.lazyExprCol(yBQ, fmt.Sprintf("ATAN2(`%s`, `%s`)", yBQ.name, xBQ.name), resultName, dataset.DTypeFloat64)
}

// --- Hyperbolic / special ---

func (e *Engine) Tanh(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.unaryMathFn("TANH", col)
}

func (e *Engine) Sigmoid(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	bqCol, ok := col.(*bqColumn)
	if !ok {
		return e.localEngine().Sigmoid(col)
	}
	// sigmoid(x) = 1 / (1 + EXP(-x))
	return e.lazyExprCol(bqCol, fmt.Sprintf("1.0 / (1.0 + EXP(-`%s`))", bqCol.name), bqCol.name, dataset.DTypeFloat64)
}

func (e *Engine) Erf(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	// BQ doesn't have ERF — download and delegate
	if bqCol, ok := col.(*bqColumn); ok {
		ds, err := bqCol.ds.download()
		if err != nil {
			return nil, err
		}
		localCol, err := ds.Column(bqCol.name)
		if err != nil {
			return nil, err
		}
		return e.localEngine().Erf(localCol)
	}
	return e.localEngine().Erf(col)
}

// --- Rounding ---

func (e *Engine) Round(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.unaryMathFn("ROUND", col)
}
func (e *Engine) Floor(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.unaryMathFn("FLOOR", col)
}
func (e *Engine) Ceil(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.unaryMathFn("CEIL", col)
}

// --- Bitwise (int64 columns) ---

func (e *Engine) BitAnd(a, b dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.binaryColOp("&", "bitand", a, b)
}

func (e *Engine) BitOr(a, b dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.binaryColOp("|", "bitor", a, b)
}

func (e *Engine) BitXor(a, b dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.binaryColOp("^", "bitxor", a, b)
}

func (e *Engine) BitNot(col dataset.AnyColumn) (dataset.AnyColumn, error) {
	return e.unaryExpr("~`%s`", "bitnot", col)
}

func (e *Engine) BitShiftLeft(col dataset.AnyColumn, n int) (dataset.AnyColumn, error) {
	bqCol, ok := col.(*bqColumn)
	if !ok {
		return e.localEngine().BitShiftLeft(col, n)
	}
	return e.lazyExprCol(bqCol, fmt.Sprintf("`%s` << %d", bqCol.name, n), bqCol.name, dataset.DTypeInt64)
}

func (e *Engine) BitShiftRight(col dataset.AnyColumn, n int) (dataset.AnyColumn, error) {
	bqCol, ok := col.(*bqColumn)
	if !ok {
		return e.localEngine().BitShiftRight(col, n)
	}
	return e.lazyExprCol(bqCol, fmt.Sprintf("`%s` >> %d", bqCol.name, n), bqCol.name, dataset.DTypeInt64)
}

// --- Shared helpers ---

// unaryMathFn generates SELECT FN(col) AS col FROM source.
func (e *Engine) unaryMathFn(fn string, col dataset.AnyColumn) (dataset.AnyColumn, error) {
	bqCol, ok := col.(*bqColumn)
	if !ok {
		return e.localUnaryMath(fn, col)
	}
	return e.lazyExprCol(bqCol, fmt.Sprintf("%s(`%s`)", fn, bqCol.name), bqCol.name, dataset.DTypeFloat64)
}

// unaryExpr generates SELECT expr AS col FROM source (for -col, ~col etc).
func (e *Engine) unaryExpr(exprFmt, suffix string, col dataset.AnyColumn) (dataset.AnyColumn, error) {
	bqCol, ok := col.(*bqColumn)
	if !ok {
		return e.localUnaryMath(suffix, col)
	}
	expr := fmt.Sprintf(exprFmt, bqCol.name)
	return e.lazyExprCol(bqCol, expr, bqCol.name, bqCol.dtype)
}

// lazyExprCol creates a lazy bqColumn backed by SELECT expr AS resultName FROM source.
// If resultName matches an existing column, the SQL uses a replacement SELECT
// (no SELECT *) to avoid duplicate field panics.
func (e *Engine) lazyExprCol(bqCol *bqColumn, expr, resultName string, dtype dataset.DType) (*bqColumn, error) {
	origFields := bqCol.ds.schema.Fields()
	replaced := false

	// Check if we're replacing an existing column
	for _, f := range origFields {
		if f.Name == resultName {
			replaced = true
			break
		}
	}

	var sql string
	var schema *dataset.Schema

	if replaced {
		// Build explicit SELECT list, replacing the target column with the expression
		var selectParts []string
		var outFields []dataset.Field
		for _, f := range origFields {
			if f.Name == resultName {
				selectParts = append(selectParts, fmt.Sprintf("(%s) AS `%s`", expr, resultName))
				outFields = append(outFields, dataset.Field{Name: resultName, Dtype: dtype})
			} else {
				selectParts = append(selectParts, "`"+f.Name+"`")
				outFields = append(outFields, f)
			}
		}
		sql = fmt.Sprintf("SELECT %s FROM %s", strings.Join(selectParts, ", "), bqCol.ds.sourceRef())
		schema = dataset.NewSchema(outFields...)
	} else {
		// New column — append via SELECT *
		sql = fmt.Sprintf("SELECT *, (%s) AS `%s` FROM %s", expr, resultName, bqCol.ds.sourceRef())
		newFields := make([]dataset.Field, len(origFields)+1)
		copy(newFields, origFields)
		newFields[len(origFields)] = dataset.Field{Name: resultName, Dtype: dtype}
		schema = dataset.NewSchema(newFields...)
	}

	ds := bqCol.ds.withSQL(sql, schema, bqCol.ds.numRows)
	return &bqColumn{ds: ds, name: resultName, dtype: dtype}, nil
}

func (e *Engine) localUnaryMath(fn string, col dataset.AnyColumn) (dataset.AnyColumn, error) {
	mk := e.localEngine()
	switch fn {
	case "ABS":
		return mk.Abs(col)
	case "neg":
		return mk.Neg(col)
	case "SIGN":
		return mk.Sign(col)
	case "SQRT":
		return mk.Sqrt(col)
	case "EXP":
		return mk.Exp(col)
	case "LN":
		return mk.Ln(col)
	case "LOG2":
		return mk.Log2(col)
	case "LOG10":
		return mk.Log10(col)
	case "SIN":
		return mk.Sin(col)
	case "COS":
		return mk.Cos(col)
	case "TAN":
		return mk.Tan(col)
	case "ASIN":
		return mk.Asin(col)
	case "ACOS":
		return mk.Acos(col)
	case "ATAN":
		return mk.Atan(col)
	case "TANH":
		return mk.Tanh(col)
	case "ROUND":
		return mk.Round(col)
	case "FLOOR":
		return mk.Floor(col)
	case "CEIL":
		return mk.Ceil(col)
	case "bitnot":
		return mk.BitNot(col)
	default:
		return nil, fmt.Errorf("bigquery: unsupported unary math %q", fn)
	}
}
