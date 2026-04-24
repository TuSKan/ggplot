package bigquery

import (
	"fmt"
	"io"
	"log"
	"sync"

	storagepb "cloud.google.com/go/bigquery/storage/apiv1/storagepb"
	"github.com/apache/arrow-go/v18/arrow/memory"

	"github.com/TuSKan/ggplot/dataset"
	arrowEngine "github.com/TuSKan/ggplot/dataset/arrow"
)

// bqDataset is a lazy Dataset pointing at a BigQuery table or pending SQL.
//
// Two tiers of laziness:
//   - Tier 1 (API params): selectedFields + rowRestriction are pushed into
//     the Storage Read API CreateReadSession call. Zero-cost, no SQL Job.
//   - Tier 2 (pending SQL): complex operations (JOIN, GROUP BY, WINDOW, CAST)
//     store a SQL query string. Execution is deferred until materialize()
//     or download() is called.
//
// Data only reaches local memory when download() runs — triggered by Values().
type bqDataset struct {
	engine  *Engine
	schema  *dataset.Schema
	table   tableRef
	numRows int64

	// Tier 1 — applied via Storage Read API params (zero-cost)
	selectedFields []string // column projection (nil = all)
	rowRestriction string   // WHERE predicate

	// Tier 2 — deferred SQL query (executed on materialize/download)
	pendingSQL string // if set, replaces table + Tier 1 params

	// Cached materialized result
	matOnce sync.Once
	matDS   *bqDataset
	matErr  error
}

// --- Dataset interface ---

func (d *bqDataset) Schema() *dataset.Schema { return d.schema }
func (d *bqDataset) NumRows() int64          { return d.numRows }
func (d *bqDataset) NumCols() int64          { return int64(d.schema.NumFields()) }

// Engine returns the BigQuery engine.
func (d *bqDataset) Engine() dataset.Engine { return d.engine }

// Column returns a lazy bqColumn — no download happens here.
func (d *bqDataset) Column(name string) (dataset.AnyColumn, error) {
	idx := d.schema.FieldIndex(name)
	if idx < 0 {
		return nil, &dataset.ErrColumnNotFound{Name: name}
	}
	field := d.schema.Field(idx)
	return &bqColumn{
		ds:    d,
		name:  name,
		dtype: field.Dtype,
	}, nil
}

// --- Laziness control ---

// needsMaterialization returns true if there's accumulated state to resolve.
func (d *bqDataset) needsMaterialization() bool {
	return d.pendingSQL != "" || d.rowRestriction != "" || d.selectedFields != nil
}

// materialize creates a temp table in BigQuery resolving all pending state.
// Returns a clean bqDataset pointing at the temp table. Data stays in BQ.
func (d *bqDataset) materialize() (*bqDataset, error) {
	if !d.needsMaterialization() {
		return d, nil
	}

	d.matOnce.Do(func() {
		d.matDS, d.matErr = d.executeMaterialize()
	})
	return d.matDS, d.matErr
}

// executeMaterialize runs the SQL Job → temp table.
func (d *bqDataset) executeMaterialize() (*bqDataset, error) {
	sql := d.resolveSQL()

	ref, meta, err := d.engine.execToTempTable(sql)
	if err != nil {
		return nil, fmt.Errorf("bigquery: materialize failed: %w", err)
	}

	schema := bqSchemaToDataset(meta.Schema)
	return &bqDataset{
		engine:  d.engine,
		schema:  schema,
		table:   ref,
		numRows: int64(meta.NumRows),
	}, nil
}

// resolveSQL returns the SQL that represents this dataset's current state.
// If pendingSQL is set, it takes priority. Otherwise, build from table + Tier 1.
func (d *bqDataset) resolveSQL() string {
	if d.pendingSQL != "" {
		return d.pendingSQL
	}
	return d.buildSelectSQL()
}

// buildSelectSQL generates SELECT ... FROM ... WHERE ... from Tier 1 state.
func (d *bqDataset) buildSelectSQL() string {
	cols := "*"
	if len(d.selectedFields) > 0 {
		cols = ""
		for i, f := range d.selectedFields {
			if i > 0 {
				cols += ", "
			}
			cols += "`" + f + "`"
		}
	}

	sql := fmt.Sprintf("SELECT %s FROM `%s`", cols, d.table.FullyQualified())
	if d.rowRestriction != "" {
		sql += " WHERE " + d.rowRestriction
	}
	return sql
}

// sourceRef returns the SQL-safe reference to this dataset's data source.
// For clean tables: backtick-quoted fully-qualified name.
// For pending SQL: a subquery wrapped in parentheses.
func (d *bqDataset) sourceRef() string {
	if d.pendingSQL != "" {
		return "(" + d.pendingSQL + ")"
	}
	src := "`" + d.table.FullyQualified() + "`"
	// If there are Tier 1 restrictions, wrap as subquery
	if d.rowRestriction != "" || d.selectedFields != nil {
		return "(" + d.buildSelectSQL() + ")"
	}
	return src
}

// --- Immutable transformations ---

// withRestriction returns a new bqDataset with appended RowRestriction.
func (d *bqDataset) withRestriction(expr string) *bqDataset {
	// If there's a pending SQL, wrap it as a subquery and apply WHERE
	if d.pendingSQL != "" {
		return &bqDataset{
			engine:     d.engine,
			schema:     d.schema,
			numRows:    d.numRows, // estimate
			pendingSQL: fmt.Sprintf("SELECT * FROM (%s) WHERE %s", d.pendingSQL, expr),
		}
	}

	restriction := expr
	if d.rowRestriction != "" {
		restriction = d.rowRestriction + " AND " + expr
	}
	return &bqDataset{
		engine:         d.engine,
		schema:         d.schema,
		table:          d.table,
		numRows:        d.numRows,
		selectedFields: d.selectedFields,
		rowRestriction: restriction,
	}
}

// withFields returns a new bqDataset with set SelectedFields.
func (d *bqDataset) withFields(fields []string) *bqDataset {
	selected := make([]string, len(fields))
	copy(selected, fields)

	schema := d.schema
	selectedFieldDefs := make([]dataset.Field, 0, len(fields))
	for _, name := range fields {
		idx := d.schema.FieldIndex(name)
		if idx >= 0 {
			selectedFieldDefs = append(selectedFieldDefs, d.schema.Field(idx))
		}
	}
	if len(selectedFieldDefs) > 0 {
		schema = dataset.NewSchema(selectedFieldDefs...)
	}

	// If there's a pending SQL, wrap it as a subquery and apply SELECT
	if d.pendingSQL != "" {
		cols := ""
		for i, f := range selected {
			if i > 0 {
				cols += ", "
			}
			cols += "`" + f + "`"
		}
		return &bqDataset{
			engine:     d.engine,
			schema:     schema,
			numRows:    d.numRows,
			pendingSQL: fmt.Sprintf("SELECT %s FROM (%s)", cols, d.pendingSQL),
		}
	}

	return &bqDataset{
		engine:         d.engine,
		schema:         schema,
		table:          d.table,
		numRows:        d.numRows,
		selectedFields: selected,
		rowRestriction: d.rowRestriction,
	}
}

// withSQL returns a new bqDataset backed by pending SQL (Tier 2 lazy).
// No execution happens — the SQL runs only when materialize() or download() is called.
func (d *bqDataset) withSQL(sql string, schema *dataset.Schema, estimatedRows int64) *bqDataset {
	return &bqDataset{
		engine:     d.engine,
		schema:     schema,
		numRows:    estimatedRows,
		pendingSQL: sql,
	}
}

// --- Lazy bqColumn ---

// bqColumn is a lazy column reference. Knows name + type but holds no data.
// Data only arrives via download() when Values() is called.
type bqColumn struct {
	ds    *bqDataset
	name  string
	dtype dataset.DType
}

func (c *bqColumn) Name() string         { return c.name }
func (c *bqColumn) Len() int64           { return c.ds.NumRows() }
func (c *bqColumn) DType() dataset.DType { return c.dtype }

// --- Download (only on Values()) ---

// download fetches the dataset from BigQuery via Storage Read API.
// This is the ONLY path that pulls data to local memory.
func (d *bqDataset) download() (dataset.Table, error) {
	ctx := d.engine.ctx

	// If there's pending SQL or restrictions, materialize first
	src := d
	if d.needsMaterialization() {
		mat, err := d.materialize()
		if err != nil {
			return nil, err
		}
		src = mat
	}

	// Quota guard
	if src.engine.quota.WarnDownloadRows > 0 && src.numRows > src.engine.quota.WarnDownloadRows {
		log.Printf("bigquery: WARNING — downloading %d rows from %s",
			src.numRows, src.table.FullyQualified())
	}
	if src.engine.quota.MaxDownloadRows > 0 && src.numRows > src.engine.quota.MaxDownloadRows {
		return nil, fmt.Errorf("bigquery: download exceeds quota (%d rows > limit %d)",
			src.numRows, src.engine.quota.MaxDownloadRows)
	}

	// Storage Read API — src is a clean table (no pending state)
	readSession := &storagepb.ReadSession{
		Table:      src.table.StoragePath(),
		DataFormat: storagepb.DataFormat_ARROW,
	}

	req := &storagepb.CreateReadSessionRequest{
		Parent:         fmt.Sprintf("projects/%s", src.engine.projectID),
		ReadSession:    readSession,
		MaxStreamCount: 1,
	}

	session, err := src.engine.readClient.CreateReadSession(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("bigquery: CreateReadSession failed: %w", err)
	}

	eng := arrowEngine.NewEngine(memory.DefaultAllocator)

	if len(session.GetStreams()) == 0 {
		return buildEmptyDataset(eng, src.schema)
	}

	streamName := session.GetStreams()[0].GetName()
	stream, err := src.engine.readClient.ReadRows(ctx, &storagepb.ReadRowsRequest{
		ReadStream: streamName,
	})
	if err != nil {
		return nil, fmt.Errorf("bigquery: ReadRows failed: %w", err)
	}

	var resultDS dataset.Table
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("bigquery: stream recv failed: %w", err)
		}

		arrowRows := resp.GetArrowRecordBatch()
		if arrowRows == nil {
			continue
		}

		arrowSchema := session.GetArrowSchema()
		if arrowSchema == nil {
			return nil, fmt.Errorf("bigquery: session has no Arrow schema")
		}

		batchTbl, err := decodeArrowBatch(eng, arrowSchema.GetSerializedSchema(), arrowRows.GetSerializedRecordBatch())
		if err != nil {
			return nil, fmt.Errorf("bigquery: failed to decode Arrow batch: %w", err)
		}

		if resultDS == nil {
			resultDS = batchTbl
		} else {
			var engIface dataset.Engine = eng
			composer, ok := engIface.(dataset.Composer)
			if !ok {
				return nil, fmt.Errorf("bigquery: arrow engine does not support Composer")
			}
			resultDS, err = composer.Stack(resultDS, batchTbl)
			if err != nil {
				return nil, fmt.Errorf("bigquery: failed to stack batches: %w", err)
			}
		}
	}

	if resultDS == nil {
		return buildEmptyDataset(eng, src.schema)
	}
	return resultDS, nil
}

// buildEmptyDataset creates an empty dataset with the given schema.
func buildEmptyDataset(eng *arrowEngine.Engine, schema *dataset.Schema) (dataset.Table, error) {
	cols := make([]dataset.AnyColumn, schema.NumFields())
	for i := 0; i < schema.NumFields(); i++ {
		f := schema.Field(i)
		switch f.Dtype {
		case dataset.DTypeFloat64:
			cols[i] = eng.NewFloat64Column(f.Name, nil)
		case dataset.DTypeInt64:
			cols[i] = eng.NewInt64Column(f.Name, nil)
		case dataset.DTypeString:
			cols[i] = eng.NewStringColumn(f.Name, nil)
		case dataset.DTypeBool:
			cols[i] = eng.NewBoolColumn(f.Name, nil)
		default:
			cols[i] = eng.NewFloat64Column(f.Name, nil)
		}
	}
	return eng.FromColumns(schema, cols...)
}
