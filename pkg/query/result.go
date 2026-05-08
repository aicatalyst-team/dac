package query

// QueryResult represents the tabular result of executing a query.
type QueryResult struct {
	Columns []ColumnInfo    `json:"columns"`
	Rows    [][]interface{} `json:"rows"`
}

type ColumnInfo struct {
	Name string `json:"name"`
	Type string `json:"type,omitempty"`
}

// DryRunResult captures metadata returned by a query validation run.
type DryRunResult struct {
	Query          string          `json:"query,omitempty"`
	ConnectionName string          `json:"connectionName,omitempty"`
	ConnectionType string          `json:"connectionType,omitempty"`
	Valid          bool            `json:"valid"`
	Columns        []ColumnInfo    `json:"columns,omitempty"`
	Rows           [][]interface{} `json:"rows,omitempty"`
}
