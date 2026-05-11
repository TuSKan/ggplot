package bigquery

import (
	"bytes"
	"errors"
	"fmt"

	"cloud.google.com/go/bigquery"
	"github.com/apache/arrow-go/v18/arrow"
	arrowarray "github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/ipc"

	"github.com/TuSKan/ggplot/dataset"
	arrowEngine "github.com/TuSKan/ggplot/dataset/arrow"
)

// bqSchemaToDataset converts a BigQuery TableSchema to a dataset.Schema.
func bqSchemaToDataset(bqSchema bigquery.Schema) *dataset.Schema {
	fields := make([]dataset.Field, 0, len(bqSchema))
	for _, fs := range bqSchema {
		f := bqFieldToDataset(fs)
		if f.Dtype != dataset.DTypeUnknown {
			fields = append(fields, f)
		}
	}

	return dataset.NewSchema(fields...)
}

// bqFieldToDataset converts a single BigQuery FieldSchema to a dataset.Field.
func bqFieldToDataset(fs *bigquery.FieldSchema) dataset.Field {
	f := dataset.Field{
		Name:     fs.Name,
		Nullable: !fs.Required,
	}

	switch fs.Type { //nolint:exhaustive // handled by default case.
	case bigquery.FloatFieldType:
		f.Dtype = dataset.DTypeFloat64
	case bigquery.IntegerFieldType:
		f.Dtype = dataset.DTypeInt64
	case bigquery.NumericFieldType, bigquery.BigNumericFieldType:
		f.Dtype = dataset.DTypeFloat64 // approximate as float64
	case bigquery.StringFieldType:
		f.Dtype = dataset.DTypeString
	case bigquery.BooleanFieldType:
		f.Dtype = dataset.DTypeBool
	case bigquery.TimestampFieldType, bigquery.DateTimeFieldType, bigquery.DateFieldType, bigquery.TimeFieldType:
		f.Dtype = dataset.DTypeTimestamp
	case bigquery.BytesFieldType:
		f.Dtype = dataset.DTypeString // bytes → string fallback
	default:
		f.Dtype = dataset.DTypeUnknown
	}

	return f
}

// datasetSchemaToBQ converts a dataset.Schema to a BigQuery TableSchema.
func datasetSchemaToBQ(schema *dataset.Schema) bigquery.Schema {
	bqFields := make([]*bigquery.FieldSchema, schema.NumFields())
	for i := range schema.NumFields() {
		f := schema.Field(i)
		bqFields[i] = datasetFieldToBQ(f)
	}

	return bqFields
}

// datasetFieldToBQ converts a single dataset.Field to a BigQuery FieldSchema.
func datasetFieldToBQ(f dataset.Field) *bigquery.FieldSchema {
	fs := &bigquery.FieldSchema{
		Name:     f.Name,
		Required: !f.Nullable,
	}

	switch f.Dtype { //nolint:exhaustive // handled by default case.
	case dataset.DTypeFloat64:
		fs.Type = bigquery.FloatFieldType
	case dataset.DTypeInt64:
		fs.Type = bigquery.IntegerFieldType
	case dataset.DTypeString:
		fs.Type = bigquery.StringFieldType
	case dataset.DTypeBool:
		fs.Type = bigquery.BooleanFieldType
	case dataset.DTypeTimestamp:
		fs.Type = bigquery.TimestampFieldType
	default:
		fs.Type = bigquery.StringFieldType // fallback
	}

	return fs
}

// arrowRecordToDataset converts an Arrow Record to a dataset.Dataset.
func arrowRecordToDataset(eng *arrowEngine.Engine, rec arrow.RecordBatch) (dataset.Table, error) {
	nCols := int(rec.NumCols())
	schema := rec.Schema()
	cols := make([]dataset.AnyColumn, nCols)
	fields := make([]dataset.Field, nCols)

	for i := range nCols {
		arrowField := schema.Field(i)
		col := rec.Column(i)
		name := arrowField.Name

		dtype := arrowTypeToDType(arrowField.Type)
		fields[i] = dataset.Field{
			Name:     name,
			Dtype:    dtype,
			Nullable: arrowField.Nullable,
		}

		var err error

		cols[i], err = arrowArrayToColumn(eng, name, col, dtype)
		if err != nil {
			return nil, fmt.Errorf("bigquery: column %q: %w", name, err)
		}
	}

	dsSchema := dataset.NewSchema(fields...)

	result, err := eng.FromColumns(dsSchema, cols...)
	if err != nil {
		return nil, fmt.Errorf("bigquery: %w", err)
	}

	return result, nil
}

// arrowTypeToDType maps Arrow types to dataset DType.
func arrowTypeToDType(dt arrow.DataType) dataset.DType {
	switch dt.ID() { //nolint:exhaustive // handled by default case.
	case arrow.FLOAT64, arrow.FLOAT32, arrow.FLOAT16:
		return dataset.DTypeFloat64
	case arrow.INT64, arrow.INT32, arrow.INT16, arrow.INT8,
		arrow.UINT64, arrow.UINT32, arrow.UINT16, arrow.UINT8:
		return dataset.DTypeInt64
	case arrow.STRING, arrow.LARGE_STRING, arrow.BINARY, arrow.LARGE_BINARY:
		return dataset.DTypeString
	case arrow.BOOL:
		return dataset.DTypeBool
	case arrow.TIMESTAMP, arrow.DATE32, arrow.DATE64, arrow.TIME32, arrow.TIME64:
		return dataset.DTypeTimestamp
	default:
		return dataset.DTypeString // fallback
	}
}

// arrowArrayToColumn wraps an Arrow array into a dataset.AnyColumn.
func arrowArrayToColumn(eng *arrowEngine.Engine, name string, arr arrow.Array, dtype dataset.DType) (dataset.AnyColumn, error) {
	switch dtype { //nolint:exhaustive // handled by default case.
	case dataset.DTypeFloat64:
		vals := make([]float64, arr.Len())
		switch a := arr.(type) {
		case *arrowarray.Float64:
			for i := range a.Len() {
				if a.IsNull(i) {
					vals[i] = 0 // or NaN
				} else {
					vals[i] = a.Value(i)
				}
			}
		case *arrowarray.Float32:
			for i := range a.Len() {
				vals[i] = float64(a.Value(i))
			}
		case *arrowarray.Int64:
			for i := range a.Len() {
				vals[i] = float64(a.Value(i))
			}
		default:
			return nil, fmt.Errorf("unsupported arrow type %T for float64", arr)
		}

		return eng.NewFloat64Column(name, vals), nil

	case dataset.DTypeInt64:
		vals := make([]int64, arr.Len())
		switch a := arr.(type) {
		case *arrowarray.Int64:
			for i := range a.Len() {
				vals[i] = a.Value(i)
			}
		case *arrowarray.Int32:
			for i := range a.Len() {
				vals[i] = int64(a.Value(i))
			}
		default:
			return nil, fmt.Errorf("unsupported arrow type %T for int64", arr)
		}

		return eng.NewInt64Column(name, vals), nil

	case dataset.DTypeString:
		vals := make([]string, arr.Len())
		switch a := arr.(type) {
		case *arrowarray.String:
			for i := range a.Len() {
				vals[i] = a.Value(i)
			}
		case *arrowarray.Binary:
			for i := range a.Len() {
				vals[i] = string(a.Value(i))
			}
		default:
			return nil, fmt.Errorf("unsupported arrow type %T for string", arr)
		}

		return eng.NewStringColumn(name, vals), nil

	case dataset.DTypeBool:
		vals := make([]bool, arr.Len())
		if a, ok := arr.(*arrowarray.Boolean); ok {
			for i := range a.Len() {
				vals[i] = a.Value(i)
			}
		}

		return eng.NewBoolColumn(name, vals), nil

	default:
		// Fallback: treat as string
		vals := make([]string, arr.Len())
		for i := range arr.Len() {
			vals[i] = arr.ValueStr(i)
		}

		return eng.NewStringColumn(name, vals), nil
	}
}

// decodeArrowBatch decodes a serialized Arrow IPC record batch from the
// BigQuery Storage API into a dataset.Dataset.
func decodeArrowBatch(eng *arrowEngine.Engine, schemaBytes, batchBytes []byte) (dataset.Table, error) {
	buf := buildIPCStream(schemaBytes, batchBytes)

	reader, err := ipc.NewReader(buf)
	if err != nil {
		return nil, fmt.Errorf("ipc.NewReader: %w", err)
	}
	defer reader.Release()

	if !reader.Next() {
		return nil, errors.New("no record batches in IPC stream")
	}

	rec := reader.RecordBatch()

	rec.Retain()
	defer rec.Release()

	return arrowRecordToDataset(eng, rec)
}

// buildIPCStream wraps a serialized Arrow schema + record batch into a minimal
// IPC stream (schema message + record batch message) suitable for ipc.NewReader.
func buildIPCStream(schemaBytes, batchBytes []byte) *bytes.Reader {
	// The BigQuery Storage API returns the schema and record batch as
	// serialized Arrow IPC messages. We need to combine them into a
	// complete IPC stream that arrow-go can read.
	//
	// An IPC stream consists of:
	//   1. Schema message (already serialized)
	//   2. One or more RecordBatch messages (already serialized)
	//
	// The bytes from BQ are already valid IPC message payloads.
	combined := make([]byte, 0, len(schemaBytes)+len(batchBytes))
	combined = append(combined, schemaBytes...)
	combined = append(combined, batchBytes...)

	return bytes.NewReader(combined)
}
