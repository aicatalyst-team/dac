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
