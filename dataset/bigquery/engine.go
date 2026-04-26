// Package bigquery implements a BigQuery SQL pushdown engine for the dataset library.
//
// All computation stays in BigQuery. The engine creates lazy datasets that
// accumulate SelectedFields and RowRestriction on the Storage Read API.
// Complex operations (GROUP BY, JOIN, WINDOW) execute SQL Jobs that write
// to temporary BigQuery tables. Data only reaches local memory when
// Column().Values() is called.
//
// Usage:
//
//	eng, _ := bigquery.NewEngine(ctx, "my-project")
//	defer eng.Close()
//
//	ds := eng.Table("analytics", "events")
//
//	result, _ := dataset.From(ds).
//	    Select("region", "revenue").
//	    Filter(dataset.Gt("revenue", 1000)).
//	    Collect()
package bigquery

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"cloud.google.com/go/bigquery"
	bqstorage "cloud.google.com/go/bigquery/storage/apiv1"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"google.golang.org/api/option"

	"github.com/TuSKan/ggplot/dataset"
	arrowEngine "github.com/TuSKan/ggplot/dataset/arrow"
)

// Engine is the BigQuery SQL pushdown engine.
// It holds three GCP clients:
//   - bqClient: for SQL Jobs (GROUP BY, JOIN, etc.) and table metadata
//   - readClient: for Storage Read API (Arrow IPC download)
//   - writeClient: (Phase 4) for Storage Write API (managed writer upload)
type Engine struct {
	projectID  string
	bqClient   *bigquery.Client
	readClient *bqstorage.BigQueryReadClient
	quota      Quota
	ctx        context.Context // engine-scoped context for background ops

	mu          sync.Mutex
	tempTables  []tableRef // temp tables to cleanup on Close
	tempCounter int64      // atomic counter for unique temp table names

	// Cached local Arrow engine for post-download operations.
	_localOnce  sync.Once
	_local      *arrowEngine.Engine
	_clientOpts []option.ClientOption // for deferred client construction
}

// tableRef identifies a BQ table.
type tableRef struct {
	ProjectID string
	DatasetID string
	TableID   string
}

func (t tableRef) FullyQualified() string {
	return fmt.Sprintf("%s.%s.%s", t.ProjectID, t.DatasetID, t.TableID)
}

func (t tableRef) StoragePath() string {
	return fmt.Sprintf("projects/%s/datasets/%s/tables/%s", t.ProjectID, t.DatasetID, t.TableID)
}

// Quota controls download limits and billing guards.
type Quota struct {
	// MaxDownloadRows limits how many rows can be pulled locally via Storage Read API.
	// 0 = unlimited. Default: 1_000_000.
	MaxDownloadRows int64

	// MaxDownloadBytes limits download size in bytes. 0 = unlimited. Default: 1 GB.
	MaxDownloadBytes int64

	// WarnDownloadRows triggers a log warning above this threshold.
	// Default: 100_000.
	WarnDownloadRows int64

	// DryRun if true, estimates cost before executing SQL Jobs.
	// Logs estimated bytes processed. Does NOT execute.
	DryRun bool

	// MaxQueryBytes limits SQL Job size in bytes. 0 = unlimited.
	// Jobs exceeding this are rejected before execution.
	MaxQueryBytes int64
}

func defaultQuota() Quota {
	return Quota{
		MaxDownloadRows:  1_000_000,
		MaxDownloadBytes: 1 << 30, // 1 GB
		WarnDownloadRows: 100_000,
	}
}

// Option configures the BigQuery engine.
type Option func(*Engine)

// WithMaxDownloadRows sets the maximum number of rows that can be downloaded.
func WithMaxDownloadRows(n int64) Option {
	return func(e *Engine) { e.quota.MaxDownloadRows = n }
}

// WithMaxDownloadBytes sets the maximum download size in bytes.
func WithMaxDownloadBytes(n int64) Option {
	return func(e *Engine) { e.quota.MaxDownloadBytes = n }
}

// WithWarnDownloadRows sets the row count threshold for download warnings.
func WithWarnDownloadRows(n int64) Option {
	return func(e *Engine) { e.quota.WarnDownloadRows = n }
}

// WithDryRun enables dry-run mode for SQL Jobs.
func WithDryRun(v bool) Option {
	return func(e *Engine) { e.quota.DryRun = v }
}

// WithMaxQueryBytes limits the maximum bytes processed by SQL Jobs.
func WithMaxQueryBytes(n int64) Option {
	return func(e *Engine) { e.quota.MaxQueryBytes = n }
}

// WithBQClient injects a pre-configured BigQuery client (e.g. for emulator tests).
func WithBQClient(c *bigquery.Client) Option {
	return func(e *Engine) { e.bqClient = c }
}

// WithClientOptions passes google.api.option.ClientOption to the GCP client constructors.
func WithClientOptions(opts ...option.ClientOption) Option {
	return func(e *Engine) { e._clientOpts = append(e._clientOpts, opts...) }
}

// NewEngine creates a BigQuery engine for the given GCP project.
// Uses Application Default Credentials unless overridden via opts.
func NewEngine(ctx context.Context, projectID string, opts ...Option) (*Engine, error) {
	e := &Engine{
		projectID: projectID,
		quota:     defaultQuota(),
		ctx:       ctx,
	}

	// Apply options first so WithBQClient / WithClientOptions take effect
	for _, o := range opts {
		o(e)
	}

	// Create BQ client if not injected
	if e.bqClient == nil {
		client, err := bigquery.NewClient(ctx, projectID, e._clientOpts...)
		if err != nil {
			return nil, fmt.Errorf("bigquery: failed to create client: %w", err)
		}
		e.bqClient = client
	}

	// Create Storage Read client if not injected
	if e.readClient == nil {
		rc, err := bqstorage.NewBigQueryReadClient(ctx, e._clientOpts...)
		if err != nil {
			e.bqClient.Close()
			return nil, fmt.Errorf("bigquery: failed to create storage read client: %w", err)
		}
		e.readClient = rc
	}

	return e, nil
}

// Name returns "bigquery".
func (e *Engine) Name() string { return "bigquery" }

// Close releases all clients and cleans up temporary tables.
func (e *Engine) Close() error {
	e.mu.Lock()
	temps := make([]tableRef, len(e.tempTables))
	copy(temps, e.tempTables)
	e.mu.Unlock()

	// Best-effort cleanup of temp tables
	for _, t := range temps {
		ref := e.bqClient.Dataset(t.DatasetID).Table(t.TableID)
		if err := ref.Delete(e.ctx); err != nil {
			slog.Warn("bigquery: failed to delete temp table",
				"table", t.FullyQualified(),
				"error", err)
		}
	}

	var errs []error
	if err := e.readClient.Close(); err != nil {
		errs = append(errs, err)
	}
	if err := e.bqClient.Close(); err != nil {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return fmt.Errorf("bigquery: close errors: %v", errs)
	}
	return nil
}

// Table returns a lazy [dataset.Dataset] pointing at a BigQuery table.
// No data is downloaded — only table metadata (schema + row count) is fetched.
func (e *Engine) Table(datasetID, tableID string) (dataset.Table, error) {
	ref := tableRef{
		ProjectID: e.projectID,
		DatasetID: datasetID,
		TableID:   tableID,
	}

	// Fetch table metadata (schema + row count) — this is free
	meta, err := e.bqClient.Dataset(datasetID).Table(tableID).Metadata(e.ctx)
	if err != nil {
		return nil, fmt.Errorf("bigquery: failed to get table metadata for %s: %w",
			ref.FullyQualified(), err)
	}

	schema := bqSchemaToDataset(meta.Schema)
	numRows := int64(meta.NumRows)

	return &bqDataset{
		engine:  e,
		schema:  schema,
		table:   ref,
		numRows: numRows,
	}, nil
}

// registerTempTable tracks a temp table for cleanup on Close.
func (e *Engine) registerTempTable(ref tableRef) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.tempTables = append(e.tempTables, ref)
}

// nextTempID returns a unique incrementing ID for temp table names.
func (e *Engine) nextTempID() int64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.tempCounter++
	return e.tempCounter
}

// localEngine returns a cached Arrow engine for post-download local operations.
func (e *Engine) localEngine() *arrowEngine.Engine {
	e._localOnce.Do(func() {
		e._local = arrowEngine.NewEngine(memory.DefaultAllocator)
	})
	return e._local
}

// --- ColumnFactory ---
//
// When Frame.Select calls FromColumns with bqColumns, we build a new bqDataset
// with withFields(). When given local (arrow) columns, we delegate to the
// arrow engine.

// NewFloat64Column creates a local float64 column via the arrow engine.
func (e *Engine) NewFloat64Column(name string, data []float64) dataset.AnyColumn {
	return e.localEngine().NewFloat64Column(name, data)
}

// NewInt64Column creates a local int64 column via the arrow engine.
func (e *Engine) NewInt64Column(name string, data []int64) dataset.AnyColumn {
	return e.localEngine().NewInt64Column(name, data)
}

// NewStringColumn creates a local string column via the arrow engine.
func (e *Engine) NewStringColumn(name string, data []string) dataset.AnyColumn {
	return e.localEngine().NewStringColumn(name, data)
}

// NewBoolColumn creates a local bool column via the arrow engine.
func (e *Engine) NewBoolColumn(name string, data []bool) dataset.AnyColumn {
	return e.localEngine().NewBoolColumn(name, data)
}

// NewTimestampColumn creates a local timestamp column via the arrow engine.
func (e *Engine) NewTimestampColumn(name string, data []int64) dataset.AnyColumn {
	return e.localEngine().NewTimestampColumn(name, data)
}

// FromColumns builds a dataset from the given columns.
// If all columns are bqColumns from the same table, returns a new bqDataset
// with SelectedFields (column projection — zero-cost).
// Otherwise, delegates to the arrow engine for local data.
func (e *Engine) FromColumns(schema *dataset.Schema, cols ...dataset.AnyColumn) (dataset.Table, error) {
	// Check if all columns are bqColumns from the same dataset
	var bqDS *bqDataset
	allBQ := true
	names := make([]string, len(cols))
	for i, col := range cols {
		bqCol, ok := col.(*bqColumn)
		if !ok {
			allBQ = false
			break
		}
		names[i] = bqCol.name
		if bqDS == nil {
			bqDS = bqCol.ds
		}
	}

	if allBQ && bqDS != nil {
		// Column projection — zero cost, just update SelectedFields
		return bqDS.withFields(names), nil
	}

	// Local data — delegate to arrow engine
	return e.localEngine().FromColumns(schema, cols...)
}

// Compile-time interface assertions — ensures all sub-interfaces are satisfied.
var (
	_ dataset.Engine         = (*Engine)(nil)
	_ dataset.ColumnFactory  = (*Engine)(nil)
	_ dataset.BuilderFactory = (*Engine)(nil)
	_ dataset.Aggregator     = (*Engine)(nil)
	_ dataset.Caster         = (*Engine)(nil)
	_ dataset.Selector       = (*Engine)(nil)
	_ dataset.Filterer       = (*Engine)(nil)
	_ dataset.Joiner         = (*Engine)(nil)
	_ dataset.Windower       = (*Engine)(nil)
	_ dataset.Reshaper       = (*Engine)(nil)
	_ dataset.Filler         = (*Engine)(nil)
	_ dataset.Composer       = (*Engine)(nil)
	_ dataset.MathKernel     = (*Engine)(nil)
)
