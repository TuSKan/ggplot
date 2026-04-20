package dataset

import (
	"fmt"
	"math"
)

// Predicate describes a row-level filter condition that can be lazily
// evaluated against a dataset to produce a boolean mask.
type Predicate interface {
	// Eval computes a boolean mask of length ds.Len(). True entries are kept.
	Eval(ds Dataset) ([]bool, error)
}

// --- Comparison predicates ---

// CmpOp is the comparison operator type.
type CmpOp int

const (
	OpGt  CmpOp = iota // >
	OpGte              // >=
	OpLt               // <
	OpLte              // <=
	OpEq               // ==
)

// CmpPredicate compares a column against a float64 threshold.
// Exported to support SQL pushdown via type-switching.
type CmpPredicate struct {
	Col string
	Val float64
	Op  CmpOp
}

// Gt returns a predicate that keeps rows where col > val.
func Gt(col string, val float64) Predicate {
	return &CmpPredicate{Col: col, Val: val, Op: OpGt}
}

// Gte returns a predicate that keeps rows where col >= val.
func Gte(col string, val float64) Predicate {
	return &CmpPredicate{Col: col, Val: val, Op: OpGte}
}

// Lt returns a predicate that keeps rows where col < val.
func Lt(col string, val float64) Predicate {
	return &CmpPredicate{Col: col, Val: val, Op: OpLt}
}

// Lte returns a predicate that keeps rows where col <= val.
func Lte(col string, val float64) Predicate {
	return &CmpPredicate{Col: col, Val: val, Op: OpLte}
}

// Eq returns a predicate that keeps rows where col == val (within float64 epsilon).
func Eq(col string, val float64) Predicate {
	return &CmpPredicate{Col: col, Val: val, Op: OpEq}
}

// Between returns a predicate that keeps rows where lo <= col <= hi.
func Between(col string, lo, hi float64) Predicate {
	return And(Gte(col, lo), Lte(col, hi))
}

// NullPredicate checks for null/non-null values.
type NullPredicate struct {
	Col      string
	KeepNull bool
}

// IsNotNull returns a predicate that keeps rows where col is not null.
func IsNotNull(col string) Predicate {
	return &NullPredicate{Col: col, KeepNull: false}
}

// IsNull returns a predicate that keeps rows where col is null.
func IsNull(col string) Predicate {
	return &NullPredicate{Col: col, KeepNull: true}
}

// --- Logical combinators ---

// LogicalPredicate combines sub-predicates with AND or OR.
type LogicalPredicate struct {
	Preds []Predicate
	IsAnd bool
}

// And returns a predicate that is true when ALL sub-predicates are true.
func And(preds ...Predicate) Predicate { return &LogicalPredicate{Preds: preds, IsAnd: true} }

// Or returns a predicate that is true when ANY sub-predicate is true.
func Or(preds ...Predicate) Predicate { return &LogicalPredicate{Preds: preds, IsAnd: false} }

// NotPredicate inverts a predicate.
type NotPredicate struct {
	Inner Predicate
}

// Not returns a predicate that inverts the given predicate.
func Not(pred Predicate) Predicate { return &NotPredicate{Inner: pred} }

// --- String predicates ---

// StrEqPredicate checks string equality.
type StrEqPredicate struct {
	Col string
	Val string
}

// EqStr returns a predicate that keeps rows where the string column equals val.
func EqStr(col string, val string) Predicate {
	return &StrEqPredicate{Col: col, Val: val}
}

// StrInPredicate checks set membership.
type StrInPredicate struct {
	Col string
	Set map[string]struct{}
}

// InStr returns a predicate that keeps rows where the string column is in the given set.
func InStr(col string, vals ...string) Predicate {
	set := make(map[string]struct{}, len(vals))
	for _, v := range vals {
		set[v] = struct{}{}
	}
	return &StrInPredicate{Col: col, Set: set}
}

// --- Eval implementations ---

func (p *CmpPredicate) Eval(ds Dataset) ([]bool, error) {
	col, err := ds.Column(p.Col)
	if err != nil {
		return nil, err
	}
	iter, ok := col.(IterableColumn)
	if !ok {
		return nil, fmt.Errorf("dataset: column %q does not support float64 iteration", p.Col)
	}
	flt, err := iter.Float64s()
	if err != nil {
		return nil, err
	}

	mask := make([]bool, ds.Len())
	for i := 0; i < ds.Len(); i++ {
		v, isNull, ok := flt.Next()
		if !ok {
			break
		}
		if isNull {
			mask[i] = false
			continue
		}
		switch p.Op {
		case OpGt:
			mask[i] = v > p.Val
		case OpGte:
			mask[i] = v >= p.Val
		case OpLt:
			mask[i] = v < p.Val
		case OpLte:
			mask[i] = v <= p.Val
		case OpEq:
			mask[i] = math.Abs(v-p.Val) < 1e-12
		}
	}
	return mask, nil
}

func (p *NullPredicate) Eval(ds Dataset) ([]bool, error) {
	col, err := ds.Column(p.Col)
	if err != nil {
		return nil, err
	}
	iter, ok := col.(IterableColumn)
	if !ok {
		return nil, fmt.Errorf("dataset: column %q does not support iteration", p.Col)
	}
	flt, err := iter.Float64s()
	if err != nil {
		return nil, err
	}

	mask := make([]bool, ds.Len())
	for i := 0; i < ds.Len(); i++ {
		_, isNull, ok := flt.Next()
		if !ok {
			break
		}
		if p.KeepNull {
			mask[i] = isNull
		} else {
			mask[i] = !isNull
		}
	}
	return mask, nil
}

func (p *LogicalPredicate) Eval(ds Dataset) ([]bool, error) {
	n := ds.Len()
	result := make([]bool, n)

	// Initialize: And starts true, Or starts false.
	for i := range result {
		result[i] = p.IsAnd
	}

	for _, pred := range p.Preds {
		mask, err := pred.Eval(ds)
		if err != nil {
			return nil, err
		}
		for i := 0; i < n && i < len(mask); i++ {
			if p.IsAnd {
				result[i] = result[i] && mask[i]
			} else {
				result[i] = result[i] || mask[i]
			}
		}
	}
	return result, nil
}

func (p *NotPredicate) Eval(ds Dataset) ([]bool, error) {
	mask, err := p.Inner.Eval(ds)
	if err != nil {
		return nil, err
	}
	for i := range mask {
		mask[i] = !mask[i]
	}
	return mask, nil
}

func (p *StrEqPredicate) Eval(ds Dataset) ([]bool, error) {
	col, err := ds.Column(p.Col)
	if err != nil {
		return nil, err
	}
	iter, ok := col.(IterableColumn)
	if !ok {
		return nil, fmt.Errorf("dataset: column %q does not support string iteration", p.Col)
	}
	sit, err := iter.Strings()
	if err != nil {
		return nil, err
	}

	mask := make([]bool, ds.Len())
	for i := 0; i < ds.Len(); i++ {
		v, isNull, ok := sit.Next()
		if !ok {
			break
		}
		mask[i] = !isNull && v == p.Val
	}
	return mask, nil
}

func (p *StrInPredicate) Eval(ds Dataset) ([]bool, error) {
	col, err := ds.Column(p.Col)
	if err != nil {
		return nil, err
	}
	iter, ok := col.(IterableColumn)
	if !ok {
		return nil, fmt.Errorf("dataset: column %q does not support string iteration", p.Col)
	}
	sit, err := iter.Strings()
	if err != nil {
		return nil, err
	}

	mask := make([]bool, ds.Len())
	for i := 0; i < ds.Len(); i++ {
		v, isNull, ok := sit.Next()
		if !ok {
			break
		}
		if isNull {
			mask[i] = false
			continue
		}
		_, mask[i] = p.Set[v]
	}
	return mask, nil
}
