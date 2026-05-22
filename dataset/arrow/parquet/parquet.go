// Package parquet provides the Arrow Parquet engine driver.
package parquet

import (
	"context"
	"fmt"
	"io"
	"math"

	"github.com/TuSKan/ggplot/dataset"
	ggplotarrow "github.com/TuSKan/ggplot/dataset/arrow"

	"github.com/apache/arrow-go/v18/arrow"
	arrowarray "github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/parquet/file"
	"github.com/apache/arrow-go/v18/parquet/pqarrow"
)

type arrowParquetHandler struct{}

// ReadParquet reads Parquet data using pqarrow for zero-copy columnar ingest.
func (h *arrowParquetHandler) ReadParquet(ctx context.Context, eng dataset.Engine, r io.ReaderAt, size int64, _ dataset.ParquetConfig) (dataset.Table, error) {
	e, ok := eng.(*ggplotarrow.Engine)
	if !ok {
		return nil, fmt.Errorf("arrow/parquet: expected *arrow.Engine, got %T: %w", eng, dataset.ErrUnsupportedEngine)
	}

	sr := io.NewSectionReader(r, 0, size)

	pf, err := file.NewParquetReader(sr)
	if err != nil {
		return nil, fmt.Errorf("arrow: parquet open: %w", err)
	}
	defer func() { _ = pf.Close() }()

	arrowRdr, err := pqarrow.NewFileReader(pf, pqarrow.ArrowReadProperties{}, e.Alloc())
	if err != nil {
		return nil, fmt.Errorf("arrow: parquet reader: %w", err)
	}

	tbl, err := arrowRdr.ReadTable(ctx)
	if err != nil {
		return nil, fmt.Errorf("arrow: parquet read table: %w", err)
	}
	defer tbl.Release()

	return arrowTableToDataset(e, tbl)
}

// arrowTableToDataset converts an arrow.Table into a dataset.Dataset.
func arrowTableToDataset(eng *ggplotarrow.Engine, tbl arrow.Table) (dataset.Table, error) {
	schema := tbl.Schema()
	nCols := int(tbl.NumCols())
	nRows := int(tbl.NumRows())

	if nCols == 0 || nRows == 0 {
		res, err := eng.FromColumns(dataset.NewSchema())
		if err != nil {
			return nil, fmt.Errorf("arrow/parquet: %w", err)
		}

		return res, nil
	}

	var (
		fields []dataset.Field
		cols   []dataset.AnyColumn
	)

	for i := range nCols {
		f := schema.Field(i)
		chunked := tbl.Column(i)
		name := f.Name

		switch f.Type.ID() { //nolint:exhaustive // intentional subset; default case handles the rest.
		case arrow.FLOAT64:
			data := make([]float64, 0, nRows)

			for _, chunk := range chunked.Data().Chunks() {
				arr := chunk.(*arrowarray.Float64) //nolint:errcheck,forcetypeassert // type guaranteed by dispatch.
				start := len(data)

				data = append(data, arr.Float64Values()...)
				for j := range arr.Len() {
					if arr.IsNull(j) {
						data[start+j] = math.NaN()
					}
				}
			}

			fields = append(fields, dataset.FloatCol(name))
			cols = append(cols, eng.NewFloat64Column(name, data))

		case arrow.INT64:
			data := make([]int64, 0, nRows)

			for _, chunk := range chunked.Data().Chunks() {
				arr := chunk.(*arrowarray.Int64) //nolint:errcheck,forcetypeassert // type guaranteed by dispatch.
				data = append(data, arr.Int64Values()...)
			}

			fields = append(fields, dataset.IntCol(name))
			cols = append(cols, eng.NewInt64Column(name, data))

		case arrow.BOOL:
			data := make([]bool, 0, nRows)

			for _, chunk := range chunked.Data().Chunks() {
				arr := chunk.(*arrowarray.Boolean) //nolint:errcheck,forcetypeassert // type guaranteed by dispatch.
				for j := range arr.Len() {
					data = append(data, arr.Value(j))
				}
			}

			fields = append(fields, dataset.BoolCol(name))
			cols = append(cols, eng.NewBoolColumn(name, data))

		default: // string
			data := make([]string, 0, nRows)

			for _, chunk := range chunked.Data().Chunks() {
				arr := chunk.(*arrowarray.String) //nolint:errcheck,forcetypeassert // type guaranteed by dispatch.
				for j := range arr.Len() {
					data = append(data, arr.Value(j))
				}
			}

			fields = append(fields, dataset.StringCol(name))
			cols = append(cols, eng.NewStringColumn(name, data))
		}
	}

	res, err := eng.FromColumns(dataset.NewSchema(fields...), cols...)
	if err != nil {
		return nil, fmt.Errorf("arrow/parquet: %w", err)
	}

	return res, nil
}

// WriteParquet writes a Dataset as Parquet using pqarrow.WriteTable.
func (h *arrowParquetHandler) WriteParquet(_ context.Context, eng dataset.Engine, w io.Writer, ds dataset.Table, _ dataset.ParquetConfig) error {
	e, ok := eng.(*ggplotarrow.Engine)
	if !ok {
		return fmt.Errorf("arrow/parquet: expected *arrow.Engine, got %T: %w", eng, dataset.ErrUnsupportedEngine)
	}

	schema := ds.Schema()
	nCols := schema.NumFields()
	nRows := int(ds.NumRows())

	// Build arrow schema.
	arrowFields := make([]arrow.Field, nCols)
	for i := range nCols {
		f := schema.Field(i)
		arrowFields[i] = arrow.Field{Name: f.Name, Type: dtypeToArrowType(f.Dtype), Nullable: true}
	}

	arrowSchema := arrow.NewSchema(arrowFields, nil)

	// Build arrow record batch.
	bld := arrowarray.NewRecordBuilder(e.Alloc(), arrowSchema)
	defer bld.Release()

	for i := range nCols {
		f := schema.Field(i)

		col, err := ds.Column(f.Name)
		if err != nil {
			return fmt.Errorf("arrow: %w", err)
		}

		appendToBuilder(bld.Field(i), col, nRows)
	}

	rec := bld.NewRecordBatch()
	defer rec.Release()

	// Build table from record.
	tbl := arrowarray.NewTableFromRecords(arrowSchema, []arrow.RecordBatch{rec})
	defer tbl.Release()

	// Write table to parquet via pqarrow.
	if err := pqarrow.WriteTable(tbl, w, int64(nRows), nil, pqarrow.DefaultWriterProps()); err != nil {
		return fmt.Errorf("arrow: %w", err)
	}

	return nil
}

func dtypeToArrowType(dt dataset.DType) arrow.DataType {
	switch dt { //nolint:exhaustive // intentional subset; default case handles the rest.
	case dataset.DTypeFloat64:
		return arrow.PrimitiveTypes.Float64
	case dataset.DTypeInt64:
		return arrow.PrimitiveTypes.Int64
	case dataset.DTypeBool:
		return arrow.FixedWidthTypes.Boolean
	default:
		return arrow.BinaryTypes.String
	}
}

func appendToBuilder(bldr arrowarray.Builder, col dataset.AnyColumn, _ int) {
	switch c := col.(type) {
	case dataset.Column[float64]:
		fb := bldr.(*arrowarray.Float64Builder) //nolint:errcheck,forcetypeassert // type guaranteed by dispatch.

		vals := c.Values()
		for _, v := range vals {
			if math.IsNaN(v) {
				fb.AppendNull()
			} else {
				fb.Append(v)
			}
		}
	case dataset.Column[int64]:
		ib := bldr.(*arrowarray.Int64Builder) //nolint:errcheck,forcetypeassert // type guaranteed by dispatch.
		ib.AppendValues(c.Values(), nil)
	case dataset.Column[bool]:
		bb := bldr.(*arrowarray.BooleanBuilder) //nolint:errcheck,forcetypeassert // type guaranteed by dispatch.
		for _, v := range c.Values() {
			bb.Append(v)
		}
	case dataset.Column[string]:
		sb := bldr.(*arrowarray.StringBuilder) //nolint:errcheck,forcetypeassert // type guaranteed by dispatch.
		for _, v := range c.Values() {
			sb.Append(v)
		}
	}
}

func init() {
	h := &arrowParquetHandler{}
	dataset.RegisterParquetReader("arrow", h)
	dataset.RegisterParquetWriter("arrow", h)
}
