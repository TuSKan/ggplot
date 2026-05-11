package memory

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/TuSKan/ggplot/dataset"
)

// ReadCSV reads CSV data using go-simdcsv with schema inference.
func (e *Engine) ReadCSV(_ context.Context, r io.Reader, cfg dataset.CSVConfig) (dataset.Table, error) {
	reader := csv.NewReader(r)

	reader.Comma = cfg.Comma
	if cfg.Comment != 0 {
		reader.Comment = cfg.Comment
	}

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("memory: csv read: %w", err)
	}

	if len(records) == 0 {
		return e.FromColumns(dataset.NewSchema())
	}

	// Extract headers.
	startRow := 0

	var headers []string
	if cfg.HasHeader {
		headers = records[0]
		startRow = 1
	} else {
		nCols := len(records[0])

		headers = make([]string, nCols)
		for i := range headers {
			headers[i] = fmt.Sprintf("V%d", i+1)
		}
	}

	nCols := len(headers)
	dataRows := records[startRow:]
	nRows := len(dataRows)

	if nRows == 0 {
		fields := make([]dataset.Field, nCols)
		cols := make([]dataset.AnyColumn, nCols)

		for i, name := range headers {
			fields[i] = dataset.StringCol(name)
			cols[i] = e.NewStringColumn(name, nil)
		}

		return e.FromColumns(dataset.NewSchema(fields...), cols...)
	}

	// Build null set.
	nullSet := map[string]bool{}
	for _, v := range cfg.NullValues {
		nullSet[v] = true
	}

	// Collect raw column data.
	rawCols := make([][]string, nCols)
	for i := range rawCols {
		rawCols[i] = make([]string, nRows)
	}

	for row, rec := range dataRows {
		for col := 0; col < nCols && col < len(rec); col++ {
			rawCols[col][row] = rec[col]
		}
	}

	// Infer types and build columns.
	var (
		fields []dataset.Field
		cols   []dataset.AnyColumn
	)

	for i, name := range headers {
		dtype := inferType(rawCols[i], nullSet)
		switch dtype { //nolint:exhaustive // handled by default case.
		case dataset.DTypeFloat64:
			data := make([]float64, nRows)

			for j, s := range rawCols[i] {
				if nullSet[s] {
					data[j] = math.NaN()
				} else {
					v, err := strconv.ParseFloat(s, 64)
					if err != nil {
						data[j] = math.NaN()
					} else {
						data[j] = v
					}
				}
			}

			fields = append(fields, dataset.FloatCol(name))
			cols = append(cols, e.NewFloat64Column(name, data))

		case dataset.DTypeInt64:
			data := make([]int64, nRows)

			for j, s := range rawCols[i] {
				if !nullSet[s] {
					v, _ := strconv.ParseInt(s, 10, 64)
					data[j] = v
				}
			}

			fields = append(fields, dataset.IntCol(name))
			cols = append(cols, e.NewInt64Column(name, data))

		case dataset.DTypeBool:
			data := make([]bool, nRows)

			for j, s := range rawCols[i] {
				if !nullSet[s] {
					v, _ := strconv.ParseBool(s)
					data[j] = v
				}
			}

			fields = append(fields, dataset.BoolCol(name))
			cols = append(cols, e.NewBoolColumn(name, data))

		default:
			data := make([]string, nRows)
			copy(data, rawCols[i])

			fields = append(fields, dataset.StringCol(name))
			cols = append(cols, e.NewStringColumn(name, data))
		}
	}

	schema := dataset.NewSchema(fields...)

	return e.FromColumns(schema, cols...)
}

// WriteCSV writes a Dataset as CSV using go-simdcsv.
func (e *Engine) WriteCSV(_ context.Context, w io.Writer, ds dataset.Table, cfg dataset.CSVConfig) error {
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
			return fmt.Errorf("memory: %w", err)
		}
	}

	// Build column formatters.
	formatters := make([]func(int) string, nCols)
	for i := range nCols {
		f := schema.Field(i)

		col, err := ds.Column(f.Name)
		if err != nil {
			return fmt.Errorf("memory: %w", err)
		}

		formatters[i] = makeFormatter(col)
	}

	// Write data rows.
	row := make([]string, nCols)
	for r := range nRows {
		for c := range nCols {
			row[c] = formatters[c](r)
		}

		if err := writer.Write(row); err != nil {
			return fmt.Errorf("memory: %w", err)
		}
	}

	return nil
}

// makeFormatter returns a function that formats a column value at a row index.
func makeFormatter(col dataset.AnyColumn) func(int) string {
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

// inferType inspects a column of strings and returns the most specific DType.
// Priority: bool → int64 → float64 → string.
func inferType(vals []string, nullSet map[string]bool) dataset.DType {
	canBool := true
	canInt := true
	canFloat := true
	nonNull := 0

	for _, s := range vals {
		if nullSet[s] {
			continue
		}

		nonNull++

		if canBool {
			low := strings.ToLower(s)
			if low != "true" && low != "false" && low != "1" && low != "0" {
				canBool = false
			}
		}

		if canInt {
			if _, err := strconv.ParseInt(s, 10, 64); err != nil {
				canInt = false
			}
		}

		if canFloat {
			if _, err := strconv.ParseFloat(s, 64); err != nil {
				canFloat = false
			}
		}

		if !canBool && !canInt && !canFloat {
			return dataset.DTypeString
		}
	}

	if nonNull == 0 {
		return dataset.DTypeString
	}

	// Bool columns that use 1/0 are ambiguous; prefer int.
	if canBool && !canInt {
		return dataset.DTypeBool
	}

	if canInt {
		return dataset.DTypeInt64
	}

	if canFloat {
		return dataset.DTypeFloat64
	}

	return dataset.DTypeString
}
