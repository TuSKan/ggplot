package dataset

import "testing"

func TestSqlVal_StringEscaping(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  string
	}{
		{"normal", "hello", "'hello'"},
		{"apostrophe", "O'Brien", "'O''Brien'"},
		{"injection_or", "x' OR 1=1 --", "'x'' OR 1=1 --'"},
		{"injection_drop", "'; DROP TABLE users; --", "'''; DROP TABLE users; --'"},
		{"backslash", `path\to\file`, `'path\\to\\file'`},
		{"double_quote", `say "hello"`, `'say "hello"'`},
		{"empty", "", "''"},
		{"int", 42, "42"},
		{"float", 3.14, "3.14"},
		{"bool_true", true, "TRUE"},
		{"bool_false", false, "FALSE"},
		{"int64", int64(99), "99"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sqlVal(tt.input)
			if got != tt.want {
				t.Errorf("sqlVal(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidateColName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"name", "name"},
		{"user_id", "user_id"},
		{"col123", "col123"},
		{"_private", "_private"},
		{"x`; DROP", "xDROP"},   // backtick, semicolon, space stripped
		{"a'b", "ab"},           // apostrophe stripped
		{"col name", "colname"}, // space stripped
		{"日本語", ""},             // non-ASCII stripped entirely
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := validateColName(tt.input)
			if got != tt.want {
				t.Errorf("validateColName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCompPred_Expr_Escaping(t *testing.T) {
	// Normal usage
	expr := Eq("name", "test").Expr()

	want := "`name` = 'test'"
	if expr != want {
		t.Errorf("Eq(name, test).Expr() = %q, want %q", expr, want)
	}

	// SQL injection attempt
	expr = Eq("name", "x' OR 1=1 --").Expr()

	want = "`name` = 'x'' OR 1=1 --'"
	if expr != want {
		t.Errorf("Eq(name, injection).Expr() = %q, want %q", expr, want)
	}

	// Column name injection attempt — non-identifier chars stripped
	expr = Eq("x`; DROP TABLE", "test").Expr()

	want = "`xDROPTABLE` = 'test'"
	if expr != want {
		t.Errorf("Eq(malicious_col, test).Expr() = %q, want %q", expr, want)
	}
}

func TestBetweenPred_Expr_Escaping(t *testing.T) {
	expr := Between("age", "10' OR 1=1 --", 20).Expr()
	if expr != "`age` BETWEEN '10'' OR 1=1 --' AND 20" {
		t.Errorf("Between escaping failed: %q", expr)
	}
}

func TestInPred_Expr_Escaping(t *testing.T) {
	expr := In("name", "Alice", "O'Brien", "x'; DROP--").Expr()

	want := "`name` IN ('Alice', 'O''Brien', 'x''; DROP--')"
	if expr != want {
		t.Errorf("In escaping: got %q, want %q", expr, want)
	}
}
