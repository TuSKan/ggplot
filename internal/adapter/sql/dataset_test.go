package sql_test

import (
	"testing"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/TuSKan/ggplot/internal/adapter/sql"
)

func TestSQLPushdown_LenAndColumns(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed attaching sqlmock: %v", err)
	}
	defer db.Close()

	rt := sql.NewTable(db, "flight_metrics")

	// Verify Introspection
	mock.ExpectQuery(`^SELECT \* FROM flight_metrics LIMIT 0$`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "altitude", "velocity"}))

	cols := rt.Columns()
	if len(cols) != 3 || cols[1] != "altitude" {
		t.Errorf("Introspection failed returning bounds. Got: %v", cols)
	}

	// Verify Fast-path COUNT query exactly avoiding Materialize dynamically!
	mock.ExpectQuery(`^SELECT COUNT\(\*\) FROM flight_metrics$`).
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(1420))

	if l := rt.Len(); l != 1420 {
		t.Errorf("expected fast-path SELECT COUNT(*) to natively return 1420, got %d", l)
	}
}

func TestSQLPushdown_Filters(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed attaching mock smoothly: %v", err)
	}
	defer db.Close()

	baseTable := sql.NewTable(db, "weather")

	filtered, err := sql.FilterSQL(baseTable, "temperature > 30.5")
	if err != nil {
		t.Fatalf("Pushdown bounds smoothly generated err: %v", err)
	}

	// Double apply testing stack mappings
	doubleFiltered, err := sql.FilterSQL(filtered, "humidity < 50")
	if err != nil {
		t.Fatalf("Pushdown bounds smoothly generated err: %v", err)
	}

	// Assert final stack executes correctly
	mock.ExpectQuery(`^SELECT COUNT\(\*\) FROM weather WHERE \(temperature > 30.5\) AND \(humidity < 50\)$`).
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(42))

	if c := doubleFiltered.Len(); c != 42 {
		t.Errorf("Expected length 42, got %d natively mapping queries precisely", c)
	}
}

func TestSQLPushdown_FallbackMaterialization(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed mocked setup natively: %v", err)
	}
	defer db.Close()

	ds := sql.NewTable(db, "telemetry")
	mock.ExpectQuery(`^SELECT sensor_x FROM telemetry$`).
		WillReturnRows(sqlmock.NewRows([]string{"sensor_x"}).
			AddRow(12.5).
			AddRow(13.2).
			AddRow(15.9))

	col, err := ds.Column("sensor_x")
	if err != nil {
		t.Fatalf("Expected strictly parsed bounds yielding correctly arrow targets, fell exactly short: %v", err)
	}

	if col.Len() != 3 {
		t.Errorf("Expected mocked fallback natively buffering 3 elements properly, got %d", col.Len())
	}
	
	// Ensure mocked tracking resolved explicitly zero-copy execution constraints smoothly!
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Mock bounds not explicitly satisfied cleanly: %v", err)
	}
}
