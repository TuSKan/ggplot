package arrow

import (
	"fmt"
	"reflect"

	"github.com/TuSKan/ggplot/pkg/dataset"
)

// FromStructs uses reflection to transpose a slice of identically-typed structs into columnar arrays.
// It maps the extracted columns to a zero-copy Arrow dataset via FromMap.
// Struct fields can be optionally tagged with `arrow:"columnName"`.
func FromStructs[T any](rows []T) (dataset.Dataset, error) {
	if len(rows) == 0 {
		return nil, fmt.Errorf("dataset struct slice is empty")
	}

	rowType := reflect.TypeOf(rows[0])
	if rowType.Kind() == reflect.Ptr {
		rowType = rowType.Elem()
	}
	if rowType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("FromStructs requires a slice of structs, got %v", rowType.Kind())
	}

	numFields := rowType.NumField()
	columns := make(map[string]any, numFields)
	fieldNames := make([]string, numFields)
	fieldBuilders := make([]reflect.Value, numFields)

	// Pre-allocate columnar slices based on the struct's defined types
	for i := 0; i < numFields; i++ {
		field := rowType.Field(i)

		// Only consider exported fields
		if !field.IsExported() {
			continue
		}

		// Determine target column name using the 'arrow' struct tag, falling back to field name.
		colName := field.Tag.Get("arrow")
		if colName == "" {
			colName = field.Name
		}
		fieldNames[i] = colName

		// Create a dynamic slice of the field's type with length matching the rows count
		sliceType := reflect.SliceOf(field.Type)
		sliceVal := reflect.MakeSlice(sliceType, len(rows), len(rows))
		fieldBuilders[i] = sliceVal
		columns[colName] = sliceVal.Interface()
	}

	// Transpose rows into columnar structure
	for r, row := range rows {
		val := reflect.ValueOf(row)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}

		for c := 0; c < numFields; c++ {
			field := rowType.Field(c)
			if !field.IsExported() {
				continue
			}
			fieldVal := val.Field(c)
			fieldBuilders[c].Index(r).Set(fieldVal)
		}
	}

	// The map of extracted slices can now be fed into FromMap.
	return FromMap(columns)
}
