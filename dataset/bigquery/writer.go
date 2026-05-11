package bigquery

import (
	"fmt"
	"math"
	"strings"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/bigquery/storage/managedwriter"
	"cloud.google.com/go/bigquery/storage/managedwriter/adapt"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/TuSKan/ggplot/dataset"
)

// --- BuilderFactory ---

// NewBuilder creates a bqBuilder that accumulates rows locally and writes
// them to BigQuery via the Storage Write API (managedwriter) on Build().
//
// Usage:
//
//	b := eng.NewBuilder(schema)
//	b.Float64("x").Append(1.5)
//	b.String("name").Append("Alice")
//	ds, _ := b.Build() // streams to BQ, returns lazy bqDataset
func (e *Engine) NewBuilder(schema *dataset.Schema) dataset.Builder {
	return &bqBuilder{
		engine: e,
		schema: schema,
		cols:   make(map[string]*bqAppender),
	}
}

// bqBuilder accumulates rows in memory, then streams to BigQuery
// via the Storage Write API on Build().
type bqBuilder struct {
	engine *Engine
	schema *dataset.Schema
	cols   map[string]*bqAppender
	nRows  int
}

// bqAppender accumulates values for a single column.
type bqAppender struct {
	name    string
	dtype   dataset.DType
	floats  []float64
	ints    []int64
	strings []string
	bools   []bool
	nulls   []bool // null bitmap
}

func (b *bqBuilder) Float64(col string) dataset.Float64Appender {
	return &float64Appender{a: b.getOrCreate(col, dataset.DTypeFloat64), builder: b}
}

func (b *bqBuilder) Int64(col string) dataset.Int64Appender {
	return &int64Appender{a: b.getOrCreate(col, dataset.DTypeInt64), builder: b}
}

func (b *bqBuilder) String(col string) dataset.StringAppender {
	return &stringAppender{a: b.getOrCreate(col, dataset.DTypeString), builder: b}
}

func (b *bqBuilder) Bool(col string) dataset.BoolAppender {
	return &boolAppender{a: b.getOrCreate(col, dataset.DTypeBool), builder: b}
}

func (b *bqBuilder) getOrCreate(name string, dtype dataset.DType) *bqAppender {
	if a, ok := b.cols[name]; ok {
		return a
	}

	a := &bqAppender{name: name, dtype: dtype}
	b.cols[name] = a

	return a
}

// Build streams accumulated rows to BigQuery via the Storage Write API.
// Creates a temp table, writes all rows using managedwriter, returns lazy bqDataset.
func (b *bqBuilder) Build() (dataset.Table, error) {
	if b.nRows == 0 {
		result, fErr := b.engine.localEngine().FromColumns(b.schema)
		if fErr != nil {
			return nil, fmt.Errorf("bigquery: %w", fErr)
		}

		return result, nil
	}

	ctx := b.engine.ctx
	bqSchema := datasetSchemaToBQ(b.schema)

	// Create the destination temp table
	tempID := fmt.Sprintf("_bq_write_%d", b.engine.nextTempID())
	tempDatasetID := "_temp"

	tbl := b.engine.bqClient.Dataset(tempDatasetID).Table(tempID)
	if err := tbl.Create(ctx, &bigquery.TableMetadata{Schema: bqSchema}); err != nil {
		return nil, fmt.Errorf("bigquery: failed to create write table: %w", err)
	}

	ref := tableRef{
		ProjectID: b.engine.projectID,
		DatasetID: tempDatasetID,
		TableID:   tempID,
	}
	b.engine.registerTempTable(ref)

	// Convert BQ schema → Storage TableSchema → proto descriptor
	storageSchema, err := adapt.BQSchemaToStorageTableSchema(bqSchema)
	if err != nil {
		return nil, fmt.Errorf("bigquery: schema conversion failed: %w", err)
	}

	descriptor, err := adapt.StorageSchemaToProto2Descriptor(storageSchema, "row")
	if err != nil {
		return nil, fmt.Errorf("bigquery: proto descriptor creation failed: %w", err)
	}

	messageDescriptor, ok := descriptor.(protoreflect.MessageDescriptor)
	if !ok {
		return nil, fmt.Errorf("descriptor is not a MessageDescriptor: %w", ErrUnsupportedType)
	}

	// Convert to DescriptorProto for WithSchemaDescriptor
	descriptorProto := protodesc.ToDescriptorProto(messageDescriptor)

	// Create managedwriter client
	writeClient, err := managedwriter.NewClient(ctx, b.engine.projectID)
	if err != nil {
		return nil, fmt.Errorf("bigquery: failed to create write client: %w", err)
	}
	defer func() { _ = writeClient.Close() }()

	// Table path for the writer
	tablePath := fmt.Sprintf(
		"projects/%s/datasets/%s/tables/%s",
		b.engine.projectID, tempDatasetID, tempID,
	)

	// Open managed stream (default stream — auto-committed)
	ms, err := writeClient.NewManagedStream(ctx,
		managedwriter.WithDestinationTable(tablePath),
		managedwriter.WithType(managedwriter.DefaultStream),
		managedwriter.WithSchemaDescriptor(descriptorProto),
	)
	if err != nil {
		return nil, fmt.Errorf("bigquery: failed to create managed stream: %w", err)
	}
	defer func() { _ = ms.Close() }()

	// Serialize all rows as protobuf messages
	serializedRows := make([][]byte, b.nRows)
	for i := range b.nRows {
		msg := dynamicpb.NewMessage(messageDescriptor)

		for _, field := range b.schema.Fields() {
			a, aOk := b.cols[field.Name]
			if !aOk {
				continue
			}

			// Proto field names are lowercase with underscores
			fdName := protoreflect.Name(strings.ToLower(field.Name))

			fd := messageDescriptor.Fields().ByName(fdName)
			if fd == nil {
				// Try exact match
				fd = messageDescriptor.Fields().ByName(protoreflect.Name(field.Name))
			}

			if fd == nil {
				continue
			}

			// Check null
			if a.nulls != nil && i < len(a.nulls) && a.nulls[i] {
				continue // skip — proto default value
			}

			switch field.Dtype { //nolint:exhaustive // intentional subset; default case handles the rest.
			case dataset.DTypeFloat64:
				if i < len(a.floats) && !math.IsNaN(a.floats[i]) {
					msg.Set(fd, protoreflect.ValueOfFloat64(a.floats[i]))
				}
			case dataset.DTypeInt64:
				if i < len(a.ints) {
					msg.Set(fd, protoreflect.ValueOfInt64(a.ints[i]))
				}
			case dataset.DTypeString:
				if i < len(a.strings) {
					msg.Set(fd, protoreflect.ValueOfString(a.strings[i]))
				}
			case dataset.DTypeBool:
				if i < len(a.bools) {
					msg.Set(fd, protoreflect.ValueOfBool(a.bools[i]))
				}
			default:
			}
		}

		data, err := proto.Marshal(msg)
		if err != nil {
			return nil, fmt.Errorf("bigquery: failed to marshal row %d: %w", i, err)
		}

		serializedRows[i] = data
	}

	// Append all rows
	result, err := ms.AppendRows(ctx, serializedRows)
	if err != nil {
		return nil, fmt.Errorf("bigquery: AppendRows failed: %w", err)
	}

	// Wait for write to complete
	_, err = result.FullResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("bigquery: write commit failed: %w", err)
	}

	// Return lazy bqDataset pointing at the written table
	meta, err := tbl.Metadata(ctx)
	if err != nil {
		return nil, fmt.Errorf("bigquery: failed to get table metadata: %w", err)
	}

	schema := bqSchemaToDataset(meta.Schema)

	return &bqDataset{
		engine:  b.engine,
		schema:  schema,
		table:   ref,
		numRows: int64(meta.NumRows), //nolint:gosec // G115: safe — metadata values bounded by platform.
	}, nil
}

// --- Typed appenders ---

type float64Appender struct {
	a       *bqAppender
	builder *bqBuilder
}

func (f *float64Appender) Append(v float64) {
	f.a.floats = append(f.a.floats, v)
	f.builder.nRows = max(f.builder.nRows, len(f.a.floats))
}
func (f *float64Appender) AppendNull() {
	f.a.floats = append(f.a.floats, math.NaN())
	f.a.nulls = appendNull(f.a.nulls, len(f.a.floats)-1)
}
func (f *float64Appender) AppendValues(vs []float64) {
	f.a.floats = append(f.a.floats, vs...)
	f.builder.nRows = max(f.builder.nRows, len(f.a.floats))
}
func (f *float64Appender) Reserve(n int) { f.a.floats = grow(f.a.floats, n) }

type int64Appender struct {
	a       *bqAppender
	builder *bqBuilder
}

func (f *int64Appender) Append(v int64) {
	f.a.ints = append(f.a.ints, v)
	f.builder.nRows = max(f.builder.nRows, len(f.a.ints))
}
func (f *int64Appender) AppendNull() {
	f.a.ints = append(f.a.ints, 0)
	f.a.nulls = appendNull(f.a.nulls, len(f.a.ints)-1)
}
func (f *int64Appender) AppendValues(vs []int64) {
	f.a.ints = append(f.a.ints, vs...)
	f.builder.nRows = max(f.builder.nRows, len(f.a.ints))
}
func (f *int64Appender) Reserve(n int) { f.a.ints = grow(f.a.ints, n) }

type stringAppender struct {
	a       *bqAppender
	builder *bqBuilder
}

func (f *stringAppender) Append(v string) {
	f.a.strings = append(f.a.strings, v)
	f.builder.nRows = max(f.builder.nRows, len(f.a.strings))
}
func (f *stringAppender) AppendNull() {
	f.a.strings = append(f.a.strings, "")
	f.a.nulls = appendNull(f.a.nulls, len(f.a.strings)-1)
}
func (f *stringAppender) AppendValues(vs []string) {
	f.a.strings = append(f.a.strings, vs...)
	f.builder.nRows = max(f.builder.nRows, len(f.a.strings))
}
func (f *stringAppender) Reserve(n int) { f.a.strings = grow(f.a.strings, n) }

type boolAppender struct {
	a       *bqAppender
	builder *bqBuilder
}

func (f *boolAppender) Append(v bool) {
	f.a.bools = append(f.a.bools, v)
	f.builder.nRows = max(f.builder.nRows, len(f.a.bools))
}
func (f *boolAppender) AppendNull() {
	f.a.bools = append(f.a.bools, false)
	f.a.nulls = appendNull(f.a.nulls, len(f.a.bools)-1)
}
func (f *boolAppender) AppendValues(vs []bool) {
	f.a.bools = append(f.a.bools, vs...)
	f.builder.nRows = max(f.builder.nRows, len(f.a.bools))
}
func (f *boolAppender) Reserve(n int) { f.a.bools = grow(f.a.bools, n) }

// --- Helpers ---

func appendNull(nulls []bool, idx int) []bool {
	for len(nulls) <= idx {
		nulls = append(nulls, false)
	}

	nulls[idx] = true

	return nulls
}

func grow[T any](s []T, n int) []T {
	if cap(s)-len(s) >= n {
		return s
	}

	newCap := len(s) + n
	ns := make([]T, len(s), newCap)
	copy(ns, s)

	return ns
}
