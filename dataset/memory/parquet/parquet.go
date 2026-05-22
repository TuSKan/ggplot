// Package parquet provides the Memory Parquet engine driver.
package parquet

import (
	"context"
	"fmt"
	"io"
	"math"

	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/dataset/memory"

	pq "github.com/parquet-go/parquet-go"
)

type memoryParquetHandler struct{}

// ReadParquet reads Parquet data using parquet-go (row-based reader).
func (h *memoryParquetHandler) ReadParquet(_ context.Context, eng dataset.Engine, r io.ReaderAt, size int64, _ dataset.ParquetConfig) (dataset.Table, error) {
	e, ok := eng.(*memory.Engine)
	if !ok {
		return nil, fmt.Errorf("memory/parquet: expected *memory.Engine, got %T: %w", eng, dataset.ErrUnsupportedEngine)
	}

	f, err := pq.OpenFile(r, size)
	if err != nil {
		return nil, fmt.Errorf("memory: parquet open: %w", err)
	}

	schema := f.Schema()
	leafColumns := schema.Columns()
	nCols := len(leafColumns)
	nRows := int(f.NumRows())

	if nCols == 0 || nRows == 0 {
		tbl, err := e.FromColumns(dataset.NewSchema())
		if err != nil {
			return nil, fmt.Errorf("memory/parquet: %w", err)
		}

		return tbl, nil
	}

	// Map parquet column types → dataset DTypes.
	type colInfo struct {
		name  string
		dtype dataset.DType
		leaf  pq.Node
	}

	cols := make([]colInfo, nCols)

	for i, path := range leafColumns {
		name := path[len(path)-1] // leaf name

		node, ok := schema.Lookup(path...)
		if !ok {
			return nil, fmt.Errorf("memory: parquet column not found: %v: %w", path, memory.ErrUnsupportedType)
		}

		cols[i] = colInfo{
			name:  name,
			dtype: parquetNodeToDType(node.Node),
			leaf:  node.Node,
		}
	}

	// Read all rows using the Row-based API.
	reader := pq.NewReader(f)
	defer func() { _ = reader.Close() }()

	rows := make([]pq.Row, 0, nRows)

	rowBuf := make([]pq.Row, 256)
	for {
		n, err := reader.ReadRows(rowBuf)
		for i := range n {
			row := make(pq.Row, len(rowBuf[i]))
			copy(row, rowBuf[i])
			rows = append(rows, row)
		}

		if err != nil {
			break
		}
	}

	// Build dataset columns from rows.
	var (
		fields []dataset.Field
		dsCols []dataset.AnyColumn
	)

	for colIdx, ci := range cols {
		switch ci.dtype { //nolint:exhaustive // intentional subset; default case handles the rest.
		case dataset.DTypeFloat64:
			data := parseParquetFloat64Column(rows, colIdx)

			fields = append(fields, dataset.FloatCol(ci.name))
			dsCols = append(dsCols, e.NewFloat64Column(ci.name, data))

		case dataset.DTypeInt64:
			data := parseParquetInt64Column(rows, colIdx)

			fields = append(fields, dataset.IntCol(ci.name))
			dsCols = append(dsCols, e.NewInt64Column(ci.name, data))

		case dataset.DTypeBool:
			data := parseParquetBoolColumn(rows, colIdx)

			fields = append(fields, dataset.BoolCol(ci.name))
			dsCols = append(dsCols, e.NewBoolColumn(ci.name, data))

		default: // string
			data := parseParquetStringColumn(rows, colIdx)

			fields = append(fields, dataset.StringCol(ci.name))
			dsCols = append(dsCols, e.NewStringColumn(ci.name, data))
		}
	}

	tbl, err := e.FromColumns(dataset.NewSchema(fields...), dsCols...)
	if err != nil {
		return nil, fmt.Errorf("memory/parquet: %w", err)
	}

	return tbl, nil
}

// WriteParquet writes a Dataset as Parquet using parquet-go.
func (h *memoryParquetHandler) WriteParquet(_ context.Context, eng dataset.Engine, w io.Writer, ds dataset.Table, _ dataset.ParquetConfig) error {
	if _, ok := eng.(*memory.Engine); !ok {
		return fmt.Errorf("memory/parquet: expected *memory.Engine, got %T: %w", eng, dataset.ErrUnsupportedEngine)
	}

	schema := ds.Schema()
	nCols := schema.NumFields()
	nRows := int(ds.NumRows())

	// Build parquet schema.
	pqSchema := buildParquetSchema(schema)

	// Resolve the actual column indices assigned by the parquet schema
	// (pq.Group is a map — ordering is NOT guaranteed to match our field order).
	colIndices := make([]int, nCols)

	colData := make([]dataset.AnyColumn, nCols)
	for i := range nCols {
		f := schema.Field(i)

		col, err := ds.Column(f.Name)
		if err != nil {
			return fmt.Errorf("memory: %w", err)
		}

		colData[i] = col

		leaf, ok := pqSchema.Lookup(f.Name)
		if !ok {
			return fmt.Errorf("memory: parquet schema missing column %q: %w", f.Name, memory.ErrUnsupportedType)
		}

		colIndices[i] = leaf.ColumnIndex
	}

	// Build all rows.
	rows := make([]pq.Row, nRows)
	for r := range nRows {
		row := make(pq.Row, nCols)
		for c := range nCols {
			row[c] = makeParquetValue(colData[c], r, colIndices[c])
		}

		rows[r] = row
	}

	// Write via Buffer → row group → writer.
	buf := pq.NewBuffer(pqSchema)
	if _, err := buf.WriteRows(rows); err != nil {
		return fmt.Errorf("memory: parquet write rows: %w", err)
	}

	writer := pq.NewWriter(w)
	if _, err := writer.WriteRowGroup(buf); err != nil {
		return fmt.Errorf("memory: parquet write row group: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("memory: %w", err)
	}

	return nil
}

// parquetNodeToDType maps a parquet-go Node to our DType.
func parquetNodeToDType(node pq.Node) dataset.DType {
	if node.Type() == nil {
		return dataset.DTypeString
	}

	kind := node.Type().Kind()
	switch kind { //nolint:exhaustive // intentional subset; default case handles the rest.
	case pq.Double, pq.Float:
		return dataset.DTypeFloat64
	case pq.Int64, pq.Int32:
		return dataset.DTypeInt64
	case pq.Boolean:
		return dataset.DTypeBool
	default:
		return dataset.DTypeString
	}
}

func dtypeToParquetNode(dt dataset.DType) pq.Node {
	switch dt { //nolint:exhaustive // intentional subset; default case handles the rest.
	case dataset.DTypeFloat64:
		return pq.Optional(pq.Leaf(pq.DoubleType))
	case dataset.DTypeInt64:
		return pq.Leaf(pq.Int64Type)
	case dataset.DTypeBool:
		return pq.Leaf(pq.BooleanType)
	default:
		return pq.String()
	}
}

func buildParquetSchema(schema *dataset.Schema) *pq.Schema {
	nCols := schema.NumFields()

	groupFields := make(pq.Group, nCols)
	for i := range nCols {
		f := schema.Field(i)
		groupFields[f.Name] = dtypeToParquetNode(f.Dtype)
	}

	return pq.NewSchema("dataset", groupFields)
}

func makeParquetValue(col dataset.AnyColumn, row, colIdx int) pq.Value {
	switch c := col.(type) {
	case dataset.Column[float64]:
		v := c.Values()[row]
		if math.IsNaN(v) {
			return pq.Value{}.Level(0, 0, colIdx) // def=0 → null
		}

		return pq.DoubleValue(v).Level(0, 1, colIdx) // def=1 → present
	case dataset.Column[int64]:
		return pq.Int64Value(c.Values()[row]).Level(0, 0, colIdx)
	case dataset.Column[bool]:
		return pq.BooleanValue(c.Values()[row]).Level(0, 0, colIdx)
	case dataset.Column[string]:
		return pq.ValueOf(c.Values()[row]).Level(0, 0, colIdx)
	default:
		return pq.Value{}.Level(0, 0, colIdx)
	}
}

func init() {
	h := &memoryParquetHandler{}
	dataset.RegisterParquetReader("memory", h)
	dataset.RegisterParquetWriter("memory", h)
}

func parseParquetFloat64Column(rows []pq.Row, colIdx int) []float64 {
	data := make([]float64, len(rows))
	for i, row := range rows {
		if colIdx < len(row) && !row[colIdx].IsNull() {
			data[i] = row[colIdx].Double()
		} else {
			data[i] = math.NaN()
		}
	}

	return data
}

func parseParquetInt64Column(rows []pq.Row, colIdx int) []int64 {
	data := make([]int64, len(rows))
	for i, row := range rows {
		if colIdx < len(row) && !row[colIdx].IsNull() {
			data[i] = row[colIdx].Int64()
		}
	}

	return data
}

func parseParquetBoolColumn(rows []pq.Row, colIdx int) []bool {
	data := make([]bool, len(rows))
	for i, row := range rows {
		if colIdx < len(row) && !row[colIdx].IsNull() {
			data[i] = row[colIdx].Boolean()
		}
	}

	return data
}

func parseParquetStringColumn(rows []pq.Row, colIdx int) []string {
	data := make([]string, len(rows))
	for i, row := range rows {
		if colIdx < len(row) && !row[colIdx].IsNull() {
			data[i] = row[colIdx].String()
		}
	}

	return data
}
