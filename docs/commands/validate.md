# dac validate

Validate dashboard definitions without executing any queries. Catches structural and configuration errors before you run the server.

```shell
dac validate [flags]
```

## Flags

| Flag | Alias | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--dir` | `-d` | string | `.` | Dashboard definitions directory |

## Examples

```shell
# Validate all dashboards in current directory
dac validate

# Validate dashboards in a specific directory
dac validate --dir ./dashboards
```

## What It Checks

- **Dashboard**: `name` is required, at least one row
- **Rows**: at least one widget per row
- **Widgets**: `type` and `name` are required
- **Grid**: column spans are 1-12, row totals don't exceed 12
- **Query references**: named queries referenced by widgets exist in the `queries` map
- **Filter types**: must be `select`, `date-range`, or `text`
- **Chart types**: valid chart type names
- **Semantic layer**:
  - `source.table` is required when metrics/dimensions are defined
  - Metrics need either `aggregate` or `expression`
  - Expression metrics must reference valid metric names
  - Dimensions need a `column`

## Exit Code

- `0` — all dashboards valid
- `1` — validation errors found
