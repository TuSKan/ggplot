package dataset

import (
	"fmt"
	"math"
	"regexp"
	"strings"
)

// validSQLIdentifier matches safe BigQuery column identifiers.
var validSQLIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// validateColName sanitises a column name for use in SQL.
// If the name contains characters outside [A-Za-z0-9_] it strips them
// to prevent SQL injection via crafted column names.
func validateColName(name string) string {
	if validSQLIdentifier.MatchString(name) {
		return name
	}
	return strings.Map(func(r rune) rune {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') ||
			(r >= '0' && r <= '9') || r == '_' {
			return r
		}
		return -1
	}, name)
}

// Masker describes a row-level filter condition that can be lazily
// evaluated against a dataset to produce a boolean mask.
type Masker interface {
	// Mask computes a boolean mask of length int(ds.NumRows()). True entries are kept.
	Mask(ds Table) ([]bool, error)
}

// --- Comparison operators ---

// Op identifies a comparison operator.
type Op int

const (
	OpGt        Op = iota // col > val
	OpLt                  // col < val
	OpGe                  // col >= val
	OpLe                  // col <= val
	OpEq                  // col == val
	OpNe                  // col != val
	OpBetween             // lo <= col <= hi
	OpIn                  // col IN (vals...)
	OpIsNull              // col IS NULL
	OpIsNotNull           // col IS NOT NULL
)

func (o Op) sql() string {
	switch o {
	case OpGt:
		return ">"
	case OpLt:
		return "<"
	case OpGe:
		return ">="
	case OpLe:
		return "<="
	case OpEq:
		return "="
	case OpNe:
		return "!="
	default:
		return "?"
	}
}

// --- Comparison predicates ---

// CompPred compares a column against a scalar value.
// Implements both Masker (local eval) and Expr() (SQL pushdown).
type CompPred struct {
	Col string
	Op  Op
	Val any
}

func Gt(col string, val any) CompPred { return CompPred{Col: col, Op: OpGt, Val: val} }
func Lt(col string, val any) CompPred { return CompPred{Col: col, Op: OpLt, Val: val} }
func Ge(col string, val any) CompPred { return CompPred{Col: col, Op: OpGe, Val: val} }
func Le(col string, val any) CompPred { return CompPred{Col: col, Op: OpLe, Val: val} }
func Eq(col string, val any) CompPred { return CompPred{Col: col, Op: OpEq, Val: val} }
func Ne(col string, val any) CompPred { return CompPred{Col: col, Op: OpNe, Val: val} }

func (p CompPred) Expr() string {
	return fmt.Sprintf("`%s` %s %s", validateColName(p.Col), p.Op.sql(), sqlVal(p.Val))
}

func (p CompPred) Mask(ds Table) ([]bool, error) {
	col, err := ds.Column(p.Col)
	if err != nil {
		return nil, err
	}
	n := int(col.Len())
	mask := make([]bool, n)

	switch c := col.(type) {
	case Column[float64]:
		v := toFloat64(p.Val)
		vals := c.Values()
		for i, x := range vals {
			if math.IsNaN(x) {
				continue
			}
			mask[i] = cmpFloat64(x, v, p.Op)
		}
	case Column[int64]:
		v := toInt64(p.Val)
		vals := c.Values()
		for i, x := range vals {
			mask[i] = cmpInt64(x, v, p.Op)
		}
	case Column[string]:
		v := fmt.Sprintf("%v", p.Val)
		vals := c.Values()
		for i, x := range vals {
			mask[i] = cmpString(x, v, p.Op)
		}
	default:
		return nil, fmt.Errorf("dataset: CompPred unsupported column type %T", col)
	}
	return mask, nil
}

// --- Between ---

type BetweenPred struct {
	Col    string
	Lo, Hi any
}

func Between(col string, lo, hi any) BetweenPred {
	return BetweenPred{Col: col, Lo: lo, Hi: hi}
}

func (p BetweenPred) Expr() string {
	return fmt.Sprintf("`%s` BETWEEN %s AND %s", validateColName(p.Col), sqlVal(p.Lo), sqlVal(p.Hi))
}

func (p BetweenPred) Mask(ds Table) ([]bool, error) {
	col, err := ds.Column(p.Col)
	if err != nil {
		return nil, err
	}
	n := int(col.Len())
	mask := make([]bool, n)

	switch c := col.(type) {
	case Column[float64]:
		lo, hi := toFloat64(p.Lo), toFloat64(p.Hi)
		for i, x := range c.Values() {
			mask[i] = x >= lo && x <= hi
		}
	case Column[int64]:
		lo, hi := toInt64(p.Lo), toInt64(p.Hi)
		for i, x := range c.Values() {
			mask[i] = x >= lo && x <= hi
		}
	case Column[string]:
		lo, hi := fmt.Sprintf("%v", p.Lo), fmt.Sprintf("%v", p.Hi)
		for i, x := range c.Values() {
			mask[i] = x >= lo && x <= hi
		}
	default:
		return nil, fmt.Errorf("dataset: BetweenPred unsupported column type %T", col)
	}
	return mask, nil
}

// --- In ---

type InPred struct {
	Col  string
	Vals []any
}

func In(col string, vals ...any) InPred {
	return InPred{Col: col, Vals: vals}
}

func (p InPred) Expr() string {
	parts := make([]string, len(p.Vals))
	for i, v := range p.Vals {
		parts[i] = sqlVal(v)
	}
	return fmt.Sprintf("`%s` IN (%s)", validateColName(p.Col), strings.Join(parts, ", "))
}

func (p InPred) Mask(ds Table) ([]bool, error) {
	col, err := ds.Column(p.Col)
	if err != nil {
		return nil, err
	}
	n := int(col.Len())
	mask := make([]bool, n)

	switch c := col.(type) {
	case Column[float64]:
		set := make(map[float64]bool, len(p.Vals))
		for _, v := range p.Vals {
			set[toFloat64(v)] = true
		}
		for i, x := range c.Values() {
			mask[i] = set[x]
		}
	case Column[int64]:
		set := make(map[int64]bool, len(p.Vals))
		for _, v := range p.Vals {
			set[toInt64(v)] = true
		}
		for i, x := range c.Values() {
			mask[i] = set[x]
		}
	case Column[string]:
		set := make(map[string]bool, len(p.Vals))
		for _, v := range p.Vals {
			set[fmt.Sprintf("%v", v)] = true
		}
		for i, x := range c.Values() {
			mask[i] = set[x]
		}
	default:
		return nil, fmt.Errorf("dataset: InPred unsupported column type %T", col)
	}
	return mask, nil
}

// --- Null checks ---

type IsNullPred struct{ Col string }
type IsNotNullPred struct{ Col string }

func IsNull(col string) IsNullPred       { return IsNullPred{Col: col} }
func IsNotNull(col string) IsNotNullPred { return IsNotNullPred{Col: col} }

func (p IsNullPred) Expr() string    { return fmt.Sprintf("`%s` IS NULL", validateColName(p.Col)) }
func (p IsNotNullPred) Expr() string { return fmt.Sprintf("`%s` IS NOT NULL", validateColName(p.Col)) }

func (p IsNullPred) Mask(ds Table) ([]bool, error) {
	return nullMask(ds, p.Col, true)
}

func (p IsNotNullPred) Mask(ds Table) ([]bool, error) {
	return nullMask(ds, p.Col, false)
}

func nullMask(ds Table, colName string, wantNull bool) ([]bool, error) {
	col, err := ds.Column(colName)
	if err != nil {
		return nil, err
	}
	n := int(col.Len())
	mask := make([]bool, n)

	// Check for IsNull interface on the column
	type nuller interface{ IsNull() []bool }
	if nc, ok := col.(nuller); ok {
		nulls := nc.IsNull()
		if nulls == nil {
			// No nulls at all
			for i := range mask {
				mask[i] = !wantNull // all non-null
			}
			return mask, nil
		}
		for i, isNull := range nulls {
			if wantNull {
				mask[i] = isNull
			} else {
				mask[i] = !isNull
			}
		}
		return mask, nil
	}

	// Check for NaN in float64 columns as null proxy
	if fc, ok := col.(Column[float64]); ok {
		for i, v := range fc.Values() {
			isNull := math.IsNaN(v)
			if wantNull {
				mask[i] = isNull
			} else {
				mask[i] = !isNull
			}
		}
		return mask, nil
	}

	// Default: all non-null
	for i := range mask {
		mask[i] = !wantNull
	}
	return mask, nil
}

// --- Logical combinators ---

// AndPred combines masks with AND.
type AndPred struct{ Preds []Masker }

func And(preds ...Masker) AndPred { return AndPred{Preds: preds} }

func (p AndPred) Expr() string {
	parts := make([]string, len(p.Preds))
	for i, sub := range p.Preds {
		if e, ok := sub.(interface{ Expr() string }); ok {
			parts[i] = e.Expr()
		} else {
			parts[i] = "?"
		}
	}
	return "(" + strings.Join(parts, " AND ") + ")"
}

func (p AndPred) Mask(ds Table) ([]bool, error) {
	n := int(ds.NumRows())
	result := make([]bool, n)
	for i := range result {
		result[i] = true
	}
	for _, sub := range p.Preds {
		m, err := sub.Mask(ds)
		if err != nil {
			return nil, err
		}
		for i := range result {
			result[i] = result[i] && m[i]
		}
	}
	return result, nil
}

// OrPred combines masks with OR.
type OrPred struct{ Preds []Masker }

func Or(preds ...Masker) OrPred { return OrPred{Preds: preds} }

func (p OrPred) Expr() string {
	parts := make([]string, len(p.Preds))
	for i, sub := range p.Preds {
		if e, ok := sub.(interface{ Expr() string }); ok {
			parts[i] = e.Expr()
		} else {
			parts[i] = "?"
		}
	}
	return "(" + strings.Join(parts, " OR ") + ")"
}

func (p OrPred) Mask(ds Table) ([]bool, error) {
	n := int(ds.NumRows())
	result := make([]bool, n)
	for _, sub := range p.Preds {
		m, err := sub.Mask(ds)
		if err != nil {
			return nil, err
		}
		for i := range result {
			result[i] = result[i] || m[i]
		}
	}
	return result, nil
}

// NotPred inverts a mask.
type NotPred struct{ Pred Masker }

func Not(pred Masker) NotPred { return NotPred{Pred: pred} }

func (p NotPred) Expr() string {
	if e, ok := p.Pred.(interface{ Expr() string }); ok {
		return "NOT(" + e.Expr() + ")"
	}
	return "NOT(?)"
}

func (p NotPred) Mask(ds Table) ([]bool, error) {
	m, err := p.Pred.Mask(ds)
	if err != nil {
		return nil, err
	}
	for i := range m {
		m[i] = !m[i]
	}
	return m, nil
}

// --- Helpers ---

func sqlVal(v any) string {
	switch val := v.(type) {
	case string:
		// Escape single quotes to prevent SQL injection.
		escaped := strings.ReplaceAll(val, "'", "''")
		escaped = strings.ReplaceAll(escaped, "\\", "\\\\")
		return fmt.Sprintf("'%s'", escaped)
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%g", val)
	case float32:
		return fmt.Sprintf("%g", val)
	case int:
		return fmt.Sprintf("%d", val)
	case int64:
		return fmt.Sprintf("%d", val)
	case int32:
		return fmt.Sprintf("%d", val)
	case bool:
		if val {
			return "TRUE"
		}
		return "FALSE"
	default:
		return fmt.Sprintf("%v", val)
	}
}

func toFloat64(v any) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case int32:
		return float64(val)
	default:
		return math.NaN()
	}
}

func toInt64(v any) int64 {
	switch val := v.(type) {
	case int64:
		return val
	case int:
		return int64(val)
	case int32:
		return int64(val)
	case float64:
		return int64(val)
	default:
		return 0
	}
}

func cmpFloat64(a, b float64, op Op) bool {
	switch op {
	case OpGt:
		return a > b
	case OpLt:
		return a < b
	case OpGe:
		return a >= b
	case OpLe:
		return a <= b
	case OpEq:
		return a == b
	case OpNe:
		return a != b
	default:
		return false
	}
}

func cmpInt64(a, b int64, op Op) bool {
	switch op {
	case OpGt:
		return a > b
	case OpLt:
		return a < b
	case OpGe:
		return a >= b
	case OpLe:
		return a <= b
	case OpEq:
		return a == b
	case OpNe:
		return a != b
	default:
		return false
	}
}

func cmpString(a, b string, op Op) bool {
	switch op {
	case OpGt:
		return a > b
	case OpLt:
		return a < b
	case OpGe:
		return a >= b
	case OpLe:
		return a <= b
	case OpEq:
		return a == b
	case OpNe:
		return a != b
	default:
		return false
	}
}

// Compile-time assertions
var (
	_ Masker = CompPred{}
	_ Masker = BetweenPred{}
	_ Masker = InPred{}
	_ Masker = IsNullPred{}
	_ Masker = IsNotNullPred{}
	_ Masker = AndPred{}
	_ Masker = OrPred{}
	_ Masker = NotPred{}
	_ Masker = BoolMask(nil)
)

// BoolMask is a pre-computed boolean mask that implements Masker.
// Useful when the filter has already been computed externally (e.g. faceting).
type BoolMask []bool

func (m BoolMask) Mask(_ Table) ([]bool, error) {
	return []bool(m), nil
}

func (m BoolMask) Expr() string {
	return "TRUE" // fallback for SQL — not meaningful for pre-computed masks
}
