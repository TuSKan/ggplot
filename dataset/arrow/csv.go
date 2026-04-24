package arrow

import (
	"context"
	"fmt"
	"io"
	"math"
	"strconv"

	"github.com/TuSKan/ggplot/dataset"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	arrowcsv "github.com/apache/arrow-go/v18/arrow/csv"

	simdcsv "github.com/nnnkkk7/go-simdcsv"
)

// ReadCSV reads CSV data using arrow/csv.NewInferringReader with chunked
// streaming. Default chunk is 64K rows per batch to bound memory for large files.
func (e *Engine) ReadCSV(ctx context.Context, r io.Reader, cfg dataset.CSVConfig) (dataset.Table, error) {
	chunkSize := cfg.ChunkSize
	if chunkSize <= 0 {
		chunkSize = 1 << 16 // 65 536 rows default
	}

	var opts []arrowcsv.Option
	opts = append(opts, arrowcsv.WithHeader(cfg.HasHeader))
	opts = append(opts, arrowcsv.WithComma(cfg.Comma))
	if cfg.Comment != 0 {
		opts = append(opts, arrowcsv.WithComment(cfg.Comment))
	}
	opts = append(opts, arrowcsv.WithNullReader(true, cfg.NullValues...))
	opts = append(opts, arrowcsv.WithChunk(chunkSize))
	opts = append(opts, arrowcsv.WithAllocator(e.alloc))

	rr := arrowcsv.NewInferringReader(r, opts...)
	defer rr.Release()

	// Accumulate columns across chunks.
	type colAcc struct {
		name    string
		typeID  arrow.Type
		floats  []float64
		ints    []int64
		bools   []bool
		strings []string
	}

	var accums []colAcc
	var fields []dataset.Field

	for rr.Next() {
		rec := rr.RecordBatch()
		nCols := int(rec.NumCols())
		nRows := int(rec.NumRows())

		// First batch: initialize accumulators from schema.
		if accums == nil {
			schema := rec.Schema()
			accums = make([]colAcc, nCols)
			for i := 0; i < nCols; i++ {
				f := schema.Field(i)
				accums[i] = colAcc{name: f.Name, typeID: f.Type.ID()}
			}
		}

		// Append chunk data to accumulators.
		for i := 0; i < nCols; i++ {
			col := rec.Column(i)
			a := &accums[i]

			switch a.typeID {
			case arrow.FLOAT64:
				arr := col.(*array.Float64)
				start := len(a.floats)
				a.floats = append(a.floats, arr.Float64Values()...)
				for j := 0; j < nRows; j++ {
					if arr.IsNull(j) {
						a.floats[start+j] = math.NaN()
					}
				}

			case arrow.INT64:
				arr := col.(*array.Int64)
				a.ints = append(a.ints, arr.Int64Values()...)

			case arrow.BOOL:
				arr := col.(*array.Boolean)
				for j := 0; j < nRows; j++ {
					a.bools = append(a.bools, arr.Value(j))
				}

			default: // string
				arr := col.(*array.String)
				for j := 0; j < nRows; j++ {
					a.strings = append(a.strings, arr.Value(j))
				}
			}
		}
	}

	if err := rr.Err(); err != nil {
		return nil, fmt.Errorf("arrow: csv read: %w", err)
	}

	if accums == nil {
		return e.FromColumns(dataset.NewSchema())
	}

	// Build dataset from accumulated columns.
	var dsCols []dataset.AnyColumn
	for _, a := range accums {
		switch a.typeID {
		case arrow.FLOAT64:
			fields = append(fields, dataset.FloatCol(a.name))
			dsCols = append(dsCols, e.NewFloat64Column(a.name, a.floats))
		case arrow.INT64:
			fields = append(fields, dataset.IntCol(a.name))
			dsCols = append(dsCols, e.NewInt64Column(a.name, a.ints))
		case arrow.BOOL:
			fields = append(fields, dataset.BoolCol(a.name))
			dsCols = append(dsCols, e.NewBoolColumn(a.name, a.bools))
		default:
			fields = append(fields, dataset.StringCol(a.name))
			dsCols = append(dsCols, e.NewStringColumn(a.name, a.strings))
		}
	}

	return e.FromColumns(dataset.NewSchema(fields...), dsCols...)
}

// WriteCSV writes a Dataset as CSV using go-simdcsv (generic string-based output).
func (e *Engine) WriteCSV(ctx context.Context, w io.Writer, ds dataset.Table, cfg dataset.CSVConfig) error {
	writer := simdcsv.NewWriter(w)
	writer.Comma = cfg.Comma
	defer writer.Flush()

	schema := ds.Schema()
	nCols := schema.NumFields()
	nRows := int(ds.NumRows())

	// Write header.
	if cfg.HasHeader {
		header := make([]string, nCols)
		for i := 0; i < nCols; i++ {
			header[i] = schema.Field(i).Name
		}
		if err := writer.Write(header); err != nil {
			return err
		}
	}

	// Build column formatters.
	formatters := make([]func(int) string, nCols)
	for i := 0; i < nCols; i++ {
		f := schema.Field(i)
		col, err := ds.Column(f.Name)
		if err != nil {
			return err
		}
		formatters[i] = arrowMakeFormatter(col)
	}

	// Write data rows.
	row := make([]string, nCols)
	for r := 0; r < nRows; r++ {
		for c := 0; c < nCols; c++ {
			row[c] = formatters[c](r)
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

func arrowMakeFormatter(col dataset.AnyColumn) func(int) string {
	switch c := col.(type) {
	case dataset.Column[float64]:
		vals := c.Values()
		return func(row int) string {
			v := vals[row]
			if math.IsNaN(v) {
				return "NA"
			}
			return strconv.FormatFloat(v, 'g', -1, 64)
		}
	case dataset.Column[int64]:
		vals := c.Values()
		return func(row int) string {
			return strconv.FormatInt(vals[row], 10)
		}
	case dataset.Column[bool]:
		vals := c.Values()
		return func(row int) string {
			return strconv.FormatBool(vals[row])
		}
	case dataset.Column[string]:
		vals := c.Values()
		return func(row int) string {
			return vals[row]
		}
	default:
		return func(row int) string { return "" }
	}
}
