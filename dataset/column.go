package dataset

import "fmt"

// --- Field & Schema (aligned with arrow.Field / arrow.Schema) ---

// Field describes a single column in a dataset — its name, logical type,
// nullability, and optional metadata. This maps directly to arrow.Field.
//
// Metadata carries type-specific parameters that DType alone cannot express:
//   - Timestamp timezone: {"tz": "America/Sao_Paulo"}
//   - Display format:     {"format": "2006-01-02"}
//   - Units:              {"unit": "ns"}
type Field struct {
	Name     string
	Dtype    DType
	Nullable bool
	Metadata map[string]string
}

// Field constructors — return a single field descriptor.

func FloatCol(name string) Field     { return Field{Name: name, Dtype: DTypeFloat64} }
func IntCol(name string) Field       { return Field{Name: name, Dtype: DTypeInt64} }
func StringCol(name string) Field    { return Field{Name: name, Dtype: DTypeString} }
func BoolCol(name string) Field      { return Field{Name: name, Dtype: DTypeBool} }
func TimestampCol(name string) Field { return Field{Name: name, Dtype: DTypeTimestamp} }

// NullableFloatCol creates a nullable float64 field.
func NullableFloatCol(name string) Field {
	return Field{Name: name, Dtype: DTypeFloat64, Nullable: true}
}

// NullableIntCol creates a nullable int64 field.
func NullableIntCol(name string) Field {
	return Field{Name: name, Dtype: DTypeInt64, Nullable: true}
}

// NullableStringCol creates a nullable string field.
func NullableStringCol(name string) Field {
	return Field{Name: name, Dtype: DTypeString, Nullable: true}
}

// WithMetadata returns a copy of the field with the given metadata.
func (f Field) WithMetadata(md map[string]string) Field {
	f.Metadata = md
	return f
}

// WithNullable returns a copy of the field with Nullable set.
func (f Field) WithNullable() Field {
	f.Nullable = true
	return f
}

// Schema describes the complete structure of a dataset — an ordered
// collection of Fields with a name-to-index lookup. This maps directly
// to arrow.Schema.
type Schema struct {
	fields []Field
	index  map[string]int
}

// NewSchema creates a Schema from an ordered list of fields.
// Panics if any two fields share the same name.
func NewSchema(fields ...Field) *Schema {
	s := &Schema{
		fields: make([]Field, len(fields)),
		index:  make(map[string]int, len(fields)),
	}
	copy(s.fields, fields)
	for i, f := range s.fields {
		if _, exists := s.index[f.Name]; exists {
			panic(fmt.Sprintf("dataset: duplicate field name %q", f.Name))
		}
		s.index[f.Name] = i
	}
	return s
}

// Fields returns a copy of the schema's fields.
func (s *Schema) Fields() []Field {
	out := make([]Field, len(s.fields))
	copy(out, s.fields)
	return out
}

// Field returns the field at index i.
func (s *Schema) Field(i int) Field { return s.fields[i] }

// NumFields returns the number of fields.
func (s *Schema) NumFields() int { return len(s.fields) }

// FieldIndex returns the index of the named field, or -1.
func (s *Schema) FieldIndex(name string) int {
	if i, ok := s.index[name]; ok {
		return i
	}
	return -1
}

// HasField returns true if the schema contains a field with the given name.
func (s *Schema) HasField(name string) bool {
	_, ok := s.index[name]
	return ok
}

// --- Two-Tier Column: AnyColumn + Column[T] ---

// AnyColumn is the type-erased column interface.
// This is what Dataset stores, engines operate on, and maps hold.
// Every engine-native column type implements this.
type AnyColumn interface {
	Name() string
	Len() int
	DType() DType
}

// Column is the typed access layer. Engine-specific column types implement
// both AnyColumn and Column[T] for their native type.
//
// Values returns the underlying typed slice — zero-copy for both
// Arrow (returns the Arrow buffer) and memory (returns the Go slice).
//
// IsNull returns the null bitmap. nil means no nulls (common case, zero alloc).
type Column[T any] interface {
	AnyColumn
	Values() []T
	IsNull() []bool
}

// GetColumn retrieves a typed column from a dataset.
// This is the only place a type assertion occurs — call sites get
// compile-time type safety from this point forward.
func GetColumn[T any](ds Dataset, name string) (Column[T], error) {
	raw, err := ds.Column(name)
	if err != nil {
		return nil, err
	}
	typed, ok := raw.(Column[T])
	if !ok {
		return nil, fmt.Errorf("dataset: column %q (%s) is not Column[%T]",
			name, raw.DType(), *new(T))
	}
	return typed, nil
}
