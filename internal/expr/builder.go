package expr

func Col(name string) Expr {
	return &ColumnRef{Name: name}
}

func Lit(value any) Expr {
	switch v := value.(type) {
	case float64:
		return &LiteralExpr{Value: v, T: TypeFloat64}
	case int:
		return &LiteralExpr{Value: int64(v), T: TypeInt64}
	case int64:
		return &LiteralExpr{Value: v, T: TypeInt64}
	case bool:
		return &LiteralExpr{Value: v, T: TypeBool}
	case string:
		return &LiteralExpr{Value: v, T: TypeString}
	default:
		return &LiteralExpr{Value: v, T: TypeUnknown}
	}
}

func Add(l, r Expr) Expr { return &BinaryExpr{Op: OpAdd, Left: l, Right: r} }
func Sub(l, r Expr) Expr { return &BinaryExpr{Op: OpSub, Left: l, Right: r} }
func Mul(l, r Expr) Expr { return &BinaryExpr{Op: OpMul, Left: l, Right: r} }
func Div(l, r Expr) Expr { return &BinaryExpr{Op: OpDiv, Left: l, Right: r} }
func Eq(l, r Expr) Expr  { return &BinaryExpr{Op: OpEq, Left: l, Right: r} }
func Gt(l, r Expr) Expr  { return &BinaryExpr{Op: OpGt, Left: l, Right: r} }
func Lt(l, r Expr) Expr  { return &BinaryExpr{Op: OpLt, Left: l, Right: r} }
func And(l, r Expr) Expr { return &BinaryExpr{Op: OpAnd, Left: l, Right: r} }
func Or(l, r Expr) Expr  { return &BinaryExpr{Op: OpOr, Left: l, Right: r} }

func Call(name string, args ...Expr) Expr {
	return &CallExpr{Name: name, Args: args, Agg: false}
}

func Agg(name string, args ...Expr) Expr {
	return &CallExpr{Name: name, Args: args, Agg: true}
}
