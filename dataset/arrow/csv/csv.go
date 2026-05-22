// Package csv provides the Arrow CSV engine driver.
package csv

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"strconv"

	"github.com/TuSKan/ggplot/dataset"
	ggplotarrow "github.com/TuSKan/ggplot/dataset/arrow"

	"github.com/apache/arrow-go/v18/arrow"
	arrowarray "github.com/apache/arrow-go/v18/arrow/array"
	arrowcsv "github.com/apache/arrow-go/v18/arrow/csv"
)

type arrowCSVHandler struct{}

// ReadCSV reads CSV data using arrow/csv.NewInferringReader with chunked
// streaming. Default chunk is 64K rows per batch to bound memory for large files.
func (h *arrowCSVHandler) ReadCSV(_ context.Context, eng dataset.Engine, r io.Reader, cfg dataset.CSVConfig) (dataset.Table, error) {
	e, ok := eng.(*ggplotarrow.Engine)
	if !ok {
		return nil, fmt.Errorf("arrow/csv: expected *arrow.Engine, got %T: %w", eng, dataset.ErrUnsupportedEngine)
	}

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
	opts = append(opts, arrowcsv.WithAllocator(e.Alloc()))

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

	var (
		accums []colAcc
		fields []dataset.Field
	)

	for rr.Next() {
		rec := rr.RecordBatch()
		nCols := int(rec.NumCols())
		nRows := int(rec.NumRows())

		// First batch: initialize accumulators from schema.
		if accums == nil {
			schema := rec.Schema()

			accums = make([]colAcc, nCols)
			for i := range nCols {
				f := schema.Field(i)
				accums[i] = colAcc{name: f.Name, typeID: f.Type.ID()}
			}
		}

		// Append chunk data to accumulators.
		for i := range nCols {
			col := rec.Column(i)
			a := &accums[i]

			switch a.typeID { //nolint:exhaustive // intentional subset; default case handles the rest.
			case arrow.FLOAT64:
				arr := col.(*arrowarray.Float64) //nolint:errcheck,forcetypeassert // type guaranteed by dispatch.
				start := len(a.floats)

				a.floats = append(a.floats, arr.Float64Values()...)
				for j := range nRows {
					if arr.IsNull(j) {
						a.floats[start+j] = math.NaN()
					}
				}

			case arrow.INT64:
				arr := col.(*arrowarray.Int64) //nolint:errcheck,forcetypeassert // type guaranteed by dispatch.
				a.ints = append(a.ints, arr.Int64Values()...)

			case arrow.BOOL:
				arr := col.(*arrowarray.Boolean) //nolint:errcheck,forcetypeassert // type guaranteed by dispatch.
				for j := range nRows {
					a.bools = append(a.bools, arr.Value(j))
				}

			default: // string
				arr := col.(*arrowarray.String) //nolint:errcheck,forcetypeassert // type guaranteed by dispatch.
				for j := range nRows {
					a.strings = append(a.strings, arr.Value(j))
				}
			}
		}
	}

	if err := rr.Err(); err != nil {
		return nil, fmt.Errorf("arrow: csv read: %w", err)
	}

	if accums == nil {
		tbl, err := e.FromColumns(dataset.NewSchema())
		if err != nil {
			return nil, fmt.Errorf("arrow/csv: %w", err)
		}

		return tbl, nil
	}

	// Build dataset from accumulated columns.
	var dsCols []dataset.AnyColumn

	for _, a := range accums {
		switch a.typeID { //nolint:exhaustive // intentional subset; default case handles the rest.
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

	tbl, err := e.FromColumns(dataset.NewSchema(fields...), dsCols...)
	if err != nil {
		return nil, fmt.Errorf("arrow/csv: %w", err)
	}

	return tbl, nil
}

// WriteCSV writes a Dataset as CSV using stdlib encoding/csv (generic string-based output).
func (h *arrowCSVHandler) WriteCSV(_ context.Context, eng dataset.Engine, w io.Writer, ds dataset.Table, cfg dataset.CSVConfig) error {
	if _, ok := eng.(*ggplotarrow.Engine); !ok {
		return fmt.Errorf("arrow/csv: expected *arrow.Engine, got %T: %w", eng, dataset.ErrUnsupportedEngine)
	}

	writer := csv.NewWriter(w)

	writer.Comma = cfg.Comma
	defer writer.Flush()

	schema := ds.Schema()
	nCols := schema.NumFields()
	nRows := int(ds.NumRows())

	// Write header.
	if cfg.HasHeader {
		header := make([]string, nCols)
		for i := range nCols {
			header[i] = schema.Field(i).Name
		}

		if err := writer.Write(header); err != nil {
			return fmt.Errorf("arrow: %w", err)
		}
	}

	// Build column formatters.
	formatters := make([]func(int) string, nCols)
	for i := range nCols {
		f := schema.Field(i)

		col, err := ds.Column(f.Name)
		if err != nil {
			return fmt.Errorf("arrow: %w", err)
		}

		formatters[i] = arrowMakeFormatter(col)
	}

	// Write data rows.
	row := make([]string, nCols)
	for r := range nRows {
		for c := range nCols {
			row[c] = formatters[c](r)
		}

		if err := writer.Write(row); err != nil {
			return fmt.Errorf("arrow: %w", err)
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
		return func(_ int) string { return "" }
	}
}

func init() {
	h := &arrowCSVHandler{}
	dataset.RegisterCSVReader("arrow", h)
	dataset.RegisterCSVWriter("arrow", h)
}
