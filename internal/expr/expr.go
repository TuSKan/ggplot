package expr

// Expr defines a generic, lazily evaluated expression.
type Expr interface {
	// Eval computes the expression.
	Eval() (any, error)
}
