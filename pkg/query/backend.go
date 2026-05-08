package query

import "context"

// Backend defines the interface for executing queries against a data source.
type Backend interface {
	Execute(ctx context.Context, connection string, query string) (*QueryResult, error)
}

// DryRunner is implemented by backends that can validate a query without
// returning result rows.
type DryRunner interface {
	DryRun(ctx context.Context, connection string, query string) (*DryRunResult, error)
}
