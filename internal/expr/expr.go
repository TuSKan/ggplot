package expr

import (
	"fmt"
	"strings"
)

// Expr defines a generic, lazily evaluated typed expression AST.
type Expr interface {
	Type(schema Schema) (Type, error)
	Format() string
	Validate(schema Schema) error
	IsAggregate() bool
}

// Op defines binary operators functionally statically.
type Op string

const (
	OpAdd Op = "+"
	OpSub Op = "-"
	OpMul Op = "*"
	OpDiv Op = "/"
	OpEq  Op = "=="
	OpGt  Op = ">"
	OpLt  Op = "<"
	OpAnd Op = "AND"
	OpOr  Op = "OR"
)

func checkBinaryOp(op Op, left, right Type) (Type, error) {
	if left != right {
		// Basic implicit cast: Int64 -> Float64
		if (left == TypeInt64 && right == TypeFloat64) || (left == TypeFloat64 && right == TypeInt64) {
			left = TypeFloat64
		} else {
			return TypeUnknown, fmt.Errorf("type mismatch: %s %s %s", left, op, right)
		}
	}

	switch op {
	case OpAdd, OpSub, OpMul, OpDiv:
		if left != TypeFloat64 && left != TypeInt64 {
			return TypeUnknown, fmt.Errorf("arithmetic operator %s requires numeric types, got %s", op, left)
		}
		return left, nil
	case OpEq:
		return TypeBool, nil
	case OpGt, OpLt:
		if left != TypeFloat64 && left != TypeInt64 && left != TypeString {
			return TypeUnknown, fmt.Errorf("comparison operator %s requires comparable types, got %s", op, left)
		}
		return TypeBool, nil
	case OpAnd, OpOr:
		if left != TypeBool {
			return TypeUnknown, fmt.Errorf("logical operator %s requires boolean types, got %s", op, left)
		}
		return TypeBool, nil
	default:
		return TypeUnknown, fmt.Errorf("unknown operator: %s", op)
	}
}

// LiteralExpr represents a static constant.
type LiteralExpr struct {
	Value any
	T     Type
}

func (e *LiteralExpr) Type(_ Schema) (Type, error) { return e.T, nil }
func (e *LiteralExpr) Format() string {
	if e.T == TypeString {
		return fmt.Sprintf("%q", e.Value)
	}
	return fmt.Sprintf("%v", e.Value)
}
func (e *LiteralExpr) Validate(_ Schema) error { return nil }
func (e *LiteralExpr) IsAggregate() bool       { return false }

// ColumnRef locates data bounds.
type ColumnRef struct {
	Name string
}

func (c *ColumnRef) Type(s Schema) (Type, error) { return s.TypeOf(c.Name) }
func (c *ColumnRef) Format() string              { return fmt.Sprintf("Col(%q)", c.Name) }
func (c *ColumnRef) Validate(s Schema) error {
	_, err := s.TypeOf(c.Name)
	return err
}
func (c *ColumnRef) IsAggregate() bool { return false }

// BinaryExpr joins bounds.
type BinaryExpr struct {
	Op    Op
	Left  Expr
	Right Expr
}

func (b *BinaryExpr) Type(s Schema) (Type, error) {
	lt, err := b.Left.Type(s)
	if err != nil {
		return TypeUnknown, err
	}
	rt, err := b.Right.Type(s)
	if err != nil {
		return TypeUnknown, err
	}
	return checkBinaryOp(b.Op, lt, rt)
}

func (b *BinaryExpr) Format() string {
	return fmt.Sprintf("(%s %s %s)", b.Left.Format(), b.Op, b.Right.Format())
}

func (b *BinaryExpr) Validate(s Schema) error {
	if err := b.Left.Validate(s); err != nil {
		return err
	}
	if err := b.Right.Validate(s); err != nil {
		return err
	}
	_, err := b.Type(s)
	return err
}

func (b *BinaryExpr) IsAggregate() bool {
	return b.Left.IsAggregate() || b.Right.IsAggregate()
}

// CallExpr supports plotting scalars/aggregates.
type CallExpr struct {
	Name string
	Args []Expr
	Agg  bool
}

func (c *CallExpr) Type(s Schema) (Type, error) {
	// Dummy logic for scalars, assumes basic math functions return floats
	if len(c.Args) > 0 {
		return TypeFloat64, nil
	}
	return TypeUnknown, fmt.Errorf("function %s requires arguments", c.Name)
}

func (c *CallExpr) Format() string {
	args := []string{}
	for _, a := range c.Args {
		args = append(args, a.Format())
	}
	return fmt.Sprintf("%s(%s)", c.Name, strings.Join(args, ", "))
}

func (c *CallExpr) Validate(s Schema) error {
	for _, a := range c.Args {
		if err := a.Validate(s); err != nil {
			return err
		}
	}
	_, err := c.Type(s)
	return err
}

func (c *CallExpr) IsAggregate() bool {
	if c.Agg {
		return true
	}
	for _, a := range c.Args {
		if a.IsAggregate() {
			return true
		}
	}
	return false
}
