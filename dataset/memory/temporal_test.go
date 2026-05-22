package memory_test

import (
	"context"
	"testing"

	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/dataset/memory"
)

func TestNewTimestampColumn(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())
	col := eng.NewTimestampColumn("ts", []int64{0, 1e9, 2e9})

	if col.DType() != dataset.DTypeTimestamp {
		t.Errorf("DType = %v, want DTypeTimestamp", col.DType())
	}

	if col.Name() != "ts" {
		t.Errorf("Name = %q, want %q", col.Name(), "ts")
	}

	if col.Len() != 3 {
		t.Errorf("Len = %d, want 3", col.Len())
	}
}

func TestNewDateColumn(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())
	col := eng.NewDateColumn("d", []int64{0, 365, 730})

	if col.DType() != dataset.DTypeDate {
		t.Errorf("DType = %v, want DTypeDate", col.DType())
	}

	if col.Len() != 3 {
		t.Errorf("Len = %d, want 3", col.Len())
	}
}

func TestNewTimeColumn(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())
	col := eng.NewTimeColumn("t", []int64{0, 3600e9, 7200e9}) // 0h, 1h, 2h in ns

	if col.DType() != dataset.DTypeTime {
		t.Errorf("DType = %v, want DTypeTime", col.DType())
	}

	if col.Len() != 3 {
		t.Errorf("Len = %d, want 3", col.Len())
	}
}

func TestNewTimestampFromString(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())

	tests := []struct {
		name   string
		values []string
	}{
		{"RFC3339", []string{"2024-01-01T00:00:00Z", "2024-06-15T12:30:00Z"}},
		{"date only", []string{"2024-01-01", "2024-12-31"}},
		{"datetime space", []string{"2024-01-01 10:00:00", "2024-06-15 14:30:00"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			col, err := eng.NewTimestampFromString("ts", tt.values)
			if err != nil {
				t.Fatalf("error: %v", err)
			}

			if col.DType() != dataset.DTypeTimestamp {
				t.Errorf("DType = %v, want DTypeTimestamp", col.DType())
			}

			if col.Len() != int64(len(tt.values)) {
				t.Errorf("Len = %d, want %d", col.Len(), len(tt.values))
			}
		})
	}
}

func TestNewTimestampFromString_Error(t *testing.T) {
	t.Parallel()

	_, err := memory.NewEngine(context.Background()).NewTimestampFromString("ts", []string{"not-a-date"})
	if err == nil {
		t.Error("expected error for unparseable string")
	}
}

func TestNewDateFromString(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())

	col, err := eng.NewDateFromString("d", []string{"2024-01-01", "2024-12-31"})
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if col.DType() != dataset.DTypeDate {
		t.Errorf("DType = %v, want DTypeDate", col.DType())
	}

	if col.Len() != 2 {
		t.Errorf("Len = %d, want 2", col.Len())
	}

	// Second date should be > first.
	icol, ok := col.(dataset.Column[int64])
	if !ok {
		t.Fatal("expected Column[int64]")
	}

	vals := icol.Values()
	if vals[1] <= vals[0] {
		t.Errorf("expected vals[1](%d) > vals[0](%d)", vals[1], vals[0])
	}
}

func TestNewDateFromString_Error(t *testing.T) {
	t.Parallel()

	_, err := memory.NewEngine(context.Background()).NewDateFromString("d", []string{"invalid"})
	if err == nil {
		t.Error("expected error for unparseable date string")
	}
}

func TestFloat64_Int64Fallback(t *testing.T) {
	t.Parallel()

	// Float64 accessor should transparently convert int64 columns.
	eng := memory.NewEngine(context.Background())
	col := eng.NewTimestampColumn("ts", []int64{1e9, 2e9, 3e9})

	ds, err := dataset.NewDataset(eng, col)
	if err != nil {
		t.Fatalf("NewDataset: %v", err)
	}

	// Collect so Float64 works.
	ctx := context.Background()

	ds, err = ds.Collect(ctx)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	vals, err := ds.Float64("ts")
	if err != nil {
		t.Fatalf("Float64 fallback: %v", err)
	}

	if len(vals) != 3 {
		t.Fatalf("len = %d, want 3", len(vals))
	}

	if vals[0] != 1e9 {
		t.Errorf("vals[0] = %v, want 1e9", vals[0])
	}
}
