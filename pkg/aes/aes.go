package aes

import (
	"github.com/TuSKan/ggplot/internal/ast"
	"github.com/TuSKan/ggplot/internal/expr"
)

// X dictates the horizontal axis mapping.
func X(e expr.Expr) func(ast.Aes) {
	return func(m ast.Aes) { m["x"] = e }
}

// Y dictates the vertical axis mapping.
func Y(e expr.Expr) func(ast.Aes) {
	return func(m ast.Aes) { m["y"] = e }
}

// Color dictates the color channel.
func Color(e expr.Expr) func(ast.Aes) {
	return func(m ast.Aes) { m["color"] = e }
}

// Facade builders encapsulating declarative expression generation without leaking internal AST representations.

func Col(name string) expr.Expr                     { return expr.Col(name) }
func Lit(value any) expr.Expr                       { return expr.Lit(value) }
func Add(l, r expr.Expr) expr.Expr                  { return expr.Add(l, r) }
func Sub(l, r expr.Expr) expr.Expr                  { return expr.Sub(l, r) }
func Mul(l, r expr.Expr) expr.Expr                  { return expr.Mul(l, r) }
func Div(l, r expr.Expr) expr.Expr                  { return expr.Div(l, r) }
func Eq(l, r expr.Expr) expr.Expr                   { return expr.Eq(l, r) }
func Gt(l, r expr.Expr) expr.Expr                   { return expr.Gt(l, r) }
func Lt(l, r expr.Expr) expr.Expr                   { return expr.Lt(l, r) }
func And(l, r expr.Expr) expr.Expr                  { return expr.And(l, r) }
func Or(l, r expr.Expr) expr.Expr                   { return expr.Or(l, r) }
func Call(name string, args ...expr.Expr) expr.Expr { return expr.Call(name, args...) }
func Agg(name string, args ...expr.Expr) expr.Expr  { return expr.Agg(name, args...) }
