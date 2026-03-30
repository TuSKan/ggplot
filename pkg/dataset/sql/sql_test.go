//go:build flightsql

package flightsql

import (
	"testing"
)

func TestFlightSQLClient(t *testing.T) {
	c := &Client{}
	if c == nil {
		t.Fatal("expected flightsql Client")
	}
}
