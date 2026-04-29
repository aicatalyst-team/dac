package query

import "context"

// Backend defines the interface for executing queries against a data source.
type Backend interface {
	Execute(ctx context.Context, connection string, query string) (*QueryResult, error)
}
