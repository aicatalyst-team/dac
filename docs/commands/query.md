# dac query

Run SQL queries against configured connections. Supports inline SQL, SQL files, SQL-backed dashboard widgets, and semantic dashboard widgets.

```shell
dac query [SQL] [flags]
```

## Flags

| Flag | Alias | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--connection` | `-c` | string | | Connection name from `.bruin.yml` |
| `--file` | `-f` | string | | Path to `.sql` file |
| `--dashboard` | | string | | Dashboard name (for widget queries) |
| `--widget` | `-w` | string | | Widget name within dashboard |
| `--output` | `-o` | string | `table` | Output format: `table`, `json`, `csv` |
| `--dir` | `-d` | string | `.` | Dashboard definitions directory |

## Three Modes

### 1. Inline SQL

```shell
dac query "SELECT * FROM sales LIMIT 10" --connection my_db
```

### 2. SQL File

```shell
dac query --file queries/report.sql --connection my_db
```

### 3. Dashboard Widget

Execute a specific widget's query with its filter defaults. If the widget uses a semantic model, DAC resolves the model from `semantic/` and compiles the widget to SQL before execution:

```shell
dac query --dashboard "Sales Analytics" --widget "Revenue Trend"
```

## Output Formats

```shell
# Table (default)
dac query "SELECT region, COUNT(*) as n FROM sales GROUP BY 1" -c my_db

# JSON
dac query "SELECT * FROM sales LIMIT 5" -c my_db -o json

# CSV
dac query "SELECT * FROM sales" -c my_db -o csv > export.csv
```

## Examples

```shell
# Quick ad-hoc query
dac query "SELECT COUNT(*) FROM orders" -c my_db

# Test a widget's query in isolation
dac query --dashboard "Sales Analytics" --widget "Total Revenue"

# Export to CSV
dac query --file queries/full_export.sql -c warehouse -o csv > report.csv
```
