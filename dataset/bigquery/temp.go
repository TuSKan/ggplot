package bigquery

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"cloud.google.com/go/bigquery"
)

// execToTempTable runs a SQL query as a BigQuery Job and writes the result
// to a temporary table with a 1-hour TTL. Returns the table reference
// and metadata of the created table.
func (e *Engine) execToTempTable(sql string) (tableRef, *bigquery.TableMetadata, error) {
	// Dry-run guard
	if e.quota.DryRun {
		job, err := e.bqClient.Query(sql).Run(e.ctx)
		if err != nil {
			return tableRef{}, nil, fmt.Errorf("dry-run failed: %w", err)
		}

		status := job.LastStatus()
		if status.Statistics != nil {
			total := status.Statistics.TotalBytesProcessed
			slog.Info("bigquery: dry run",
				"sql_prefix", sql[:min(len(sql), 80)],
				"bytes", total,
				"mb", fmt.Sprintf("%.2f", float64(total)/(1<<20)))
		}

		return tableRef{}, nil, errors.New("bigquery: dry-run mode — query not executed")
	}

	// MaxQueryBytes guard
	if e.quota.MaxQueryBytes > 0 {
		q := e.bqClient.Query(sql)
		q.DryRun = true

		job, err := q.Run(e.ctx)
		if err != nil {
			return tableRef{}, nil, fmt.Errorf("cost estimation failed: %w", err)
		}

		status := job.LastStatus()
		if status.Statistics != nil {
			total := status.Statistics.TotalBytesProcessed
			if total > e.quota.MaxQueryBytes {
				return tableRef{}, nil, fmt.Errorf(
					"bigquery: query would process %d bytes (limit: %d bytes)",
					total, e.quota.MaxQueryBytes)
			}
		}
	}

	// Generate temp table name
	tempID := fmt.Sprintf("_bq_tmp_%d", time.Now().UnixNano())
	tempDatasetID := "_temp" // convention: use a _temp dataset

	// Execute the query → write to temp table
	q := e.bqClient.Query(sql)
	q.Dst = e.bqClient.Dataset(tempDatasetID).Table(tempID)
	q.WriteDisposition = bigquery.WriteTruncate
	q.CreateDisposition = bigquery.CreateIfNeeded

	job, err := q.Run(e.ctx)
	if err != nil {
		return tableRef{}, nil, fmt.Errorf("failed to run query job: %w", err)
	}

	status, err := job.Wait(e.ctx)
	if err != nil {
		return tableRef{}, nil, fmt.Errorf("query job failed: %w", err)
	}

	if status.Err() != nil {
		return tableRef{}, nil, fmt.Errorf("query job error: %w", status.Err())
	}

	// Set 1-hour TTL on the temp table
	ref := tableRef{
		ProjectID: e.projectID,
		DatasetID: tempDatasetID,
		TableID:   tempID,
	}

	tbl := e.bqClient.Dataset(tempDatasetID).Table(tempID)
	expiration := time.Now().Add(1 * time.Hour)

	_, err = tbl.Update(e.ctx, bigquery.TableMetadataToUpdate{
		ExpirationTime: expiration,
	}, "")
	if err != nil {
		// Non-fatal — table will remain but without auto-expiration
		slog.Warn("bigquery: failed to set TTL on temp table",
			"table", ref.FullyQualified(),
			"error", err)
	}

	// Register for cleanup on Close
	e.registerTempTable(ref)

	// Fetch metadata
	meta, err := tbl.Metadata(e.ctx)
	if err != nil {
		return tableRef{}, nil, fmt.Errorf("failed to get temp table metadata: %w", err)
	}

	return ref, meta, nil
}
