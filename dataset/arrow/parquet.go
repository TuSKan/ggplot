package arrow

import (
	"context"
	"fmt"
	"io"
	"math"

	"github.com/TuSKan/ggplot/dataset"

	"github.com/apache/arrow-go/v18/arrow"
	arrowarray "github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/parquet/file"
	"github.com/apache/arrow-go/v18/parquet/pqarrow"
)

// ReadParquet reads Parquet data using pqarrow for zero-copy columnar ingest.
func (e *Engine) ReadParquet(r io.ReaderAt, size int64, cfg dataset.ParquetConfig) (dataset.Dataset, error) {
	sr := io.NewSectionReader(r, 0, size)
	pf, err := file.NewParquetReader(sr)
	if err != nil {
		return nil, fmt.Errorf("arrow: parquet open: %w", err)
	}
	defer pf.Close()

	arrowRdr, err := pqarrow.NewFileReader(pf, pqarrow.ArrowReadProperties{}, e.alloc)
	if err != nil {
		return nil, fmt.Errorf("arrow: parquet reader: %w", err)
	}

	tbl, err := arrowRdr.ReadTable(context.Background())
	if err != nil {
		return nil, fmt.Errorf("arrow: parquet read table: %w", err)
	}
	defer tbl.Release()

	return arrowTableToDataset(e, tbl)
}

// arrowTableToDataset converts an arrow.Table into a dataset.Dataset.
func arrowTableToDataset(eng *Engine, tbl arrow.Table) (dataset.Dataset, error) {
	schema := tbl.Schema()
	nCols := int(tbl.NumCols())
	nRows := int(tbl.NumRows())

	if nCols == 0 || nRows == 0 {
		return eng.FromColumns(dataset.NewSchema())
	}

	var fields []dataset.Field
	var cols []dataset.AnyColumn

	for i := 0; i < nCols; i++ {
		f := schema.Field(i)
		chunked := tbl.Column(i)
		name := f.Name

		switch f.Type.ID() {
		case arrow.FLOAT64:
			data := make([]float64, 0, nRows)
			for _, chunk := range chunked.Data().Chunks() {
				arr := chunk.(*arrowarray.Float64)
				start := len(data)
				data = append(data, arr.Float64Values()...)
				for j := 0; j < arr.Len(); j++ {
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
				arr := chunk.(*arrowarray.Int64)
				data = append(data, arr.Int64Values()...)
			}
			fields = append(fields, dataset.IntCol(name))
			cols = append(cols, eng.NewInt64Column(name, data))

		case arrow.BOOL:
			data := make([]bool, 0, nRows)
			for _, chunk := range chunked.Data().Chunks() {
				arr := chunk.(*arrowarray.Boolean)
				for j := 0; j < arr.Len(); j++ {
					data = append(data, arr.Value(j))
				}
			}
			fields = append(fields, dataset.BoolCol(name))
			cols = append(cols, eng.NewBoolColumn(name, data))

		default: // string
			data := make([]string, 0, nRows)
			for _, chunk := range chunked.Data().Chunks() {
				arr := chunk.(*arrowarray.String)
				for j := 0; j < arr.Len(); j++ {
					data = append(data, arr.Value(j))
				}
			}
			fields = append(fields, dataset.StringCol(name))
			cols = append(cols, eng.NewStringColumn(name, data))
		}
	}

	return eng.FromColumns(dataset.NewSchema(fields...), cols...)
}

// WriteParquet writes a Dataset as Parquet using pqarrow.WriteTable.
func (e *Engine) WriteParquet(w io.Writer, ds dataset.Dataset, cfg dataset.ParquetConfig) error {
	schema := ds.Schema()
	nCols := schema.NumFields()
	nRows := int(ds.NumRows())

	// Build arrow schema.
	arrowFields := make([]arrow.Field, nCols)
	for i := 0; i < nCols; i++ {
		f := schema.Field(i)
		arrowFields[i] = arrow.Field{Name: f.Name, Type: dtypeToArrowType(f.Dtype), Nullable: true}
	}
	arrowSchema := arrow.NewSchema(arrowFields, nil)

	// Build arrow record batch.
	bld := arrowarray.NewRecordBuilder(e.alloc, arrowSchema)
	defer bld.Release()

	for i := 0; i < nCols; i++ {
		f := schema.Field(i)
		col, err := ds.Column(f.Name)
		if err != nil {
			return err
		}
		appendToBuilder(bld.Field(i), col, nRows)
	}

	rec := bld.NewRecord()
	defer rec.Release()

	// Build table from record.
	tbl := arrowarray.NewTableFromRecords(arrowSchema, []arrow.Record{rec})
	defer tbl.Release()

	// Write table to parquet via pqarrow.
	return pqarrow.WriteTable(tbl, w, int64(nRows), nil, pqarrow.DefaultWriterProps())
}

func dtypeToArrowType(dt dataset.DType) arrow.DataType {
	switch dt {
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

func appendToBuilder(bldr arrowarray.Builder, col dataset.AnyColumn, nRows int) {
	switch c := col.(type) {
	case dataset.Column[float64]:
		fb := bldr.(*arrowarray.Float64Builder)
		vals := c.Values()
		for _, v := range vals {
			if math.IsNaN(v) {
				fb.AppendNull()
			} else {
				fb.Append(v)
			}
		}
	case dataset.Column[int64]:
		ib := bldr.(*arrowarray.Int64Builder)
		ib.AppendValues(c.Values(), nil)
	case dataset.Column[bool]:
		bb := bldr.(*arrowarray.BooleanBuilder)
		for _, v := range c.Values() {
			bb.Append(v)
		}
	case dataset.Column[string]:
		sb := bldr.(*arrowarray.StringBuilder)
		for _, v := range c.Values() {
			sb.Append(v)
		}
	}
}
