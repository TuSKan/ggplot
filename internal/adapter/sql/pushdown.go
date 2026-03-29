package sql

import (
	"fmt"

	"github.com/TuSKan/ggplot/internal/dataset"
)

// FilterSQL pushes logical string dialect matching conditions executing constraints zero-copy onto the target backend safely.
func FilterSQL(ds dataset.Dataset, sqlPredicate string) (dataset.Dataset, error) {
	if rt, ok := ds.(*RemoteTable); ok {
		derived := &RemoteTable{
			DB:           rt.DB,
			TableName:    rt.TableName,
			whereClauses: append([]string(nil), rt.whereClauses...),
			groupBy:      append([]string(nil), rt.groupBy...),
		}
		derived.whereClauses = append(derived.whereClauses, fmt.Sprintf("(%s)", sqlPredicate))
		return derived, nil
	}
	return nil, fmt.Errorf("SQL Pushdown: target Dataset relies natively outside backend mapping limits (requires RemoteTable)")
}

// GroupBySQL injects exact aggregation targets seamlessly onto remote servers executing safely natively.
func GroupBySQL(ds dataset.Dataset, cols ...string) (dataset.Dataset, error) {
	if rt, ok := ds.(*RemoteTable); ok {
		derived := &RemoteTable{
			DB:           rt.DB,
			TableName:    rt.TableName,
			whereClauses: append([]string(nil), rt.whereClauses...),
			groupBy:      append([]string(nil), rt.groupBy...),
		}
		derived.groupBy = append(derived.groupBy, cols...)
		return derived, nil
	}
	return nil, fmt.Errorf("SQL Pushdown: target mapping relies fully heavily outside boundary backend capabilities (requires RemoteTable)")
}
