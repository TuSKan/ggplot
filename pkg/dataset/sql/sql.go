package sql

// _ "github.com/apache/arrow-adbc/go/adbc/sqldriver/flightsql"

// Client is a stub for an Arrow FlightSQL client.
// It implements dataset.Dataset.
type Client struct{}

// Verify at compile time that Client implements dataset.Dataset
// var _ dataset.Dataset = (*Client)(nil)

func (c *Client) Columns() []string {
	return nil
}
