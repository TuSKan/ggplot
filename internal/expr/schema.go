package expr

// Type identifies the native logical datatype of an expression.
type Type int

const (
	TypeUnknown Type = iota
	TypeFloat64
	TypeInt64
	TypeBool
	TypeString
)

// String returns the name of the Type for debug and error messages.
func (t Type) String() string {
	switch t {
	case TypeFloat64:
		return "Float64"
	case TypeInt64:
		return "Int64"
	case TypeBool:
		return "Bool"
	case TypeString:
		return "String"
	default:
		return "Unknown"
	}
}

// Schema allows resolving the type of a column before compilation.
type Schema interface {
	TypeOf(column string) (Type, error)
}
