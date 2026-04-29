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
- **Schemas**: YAML dashboards, semantic models, and themes match their declared Bruin schema IDs
- **Rows**: at least one widget per row
- **Widgets**: `type` and `name` are required
- **Grid**: column spans are 1-12, row totals don't exceed 12
- **Query references**: named queries referenced by widgets exist in the `queries` map
- **Filter types**: must be `select`, `date-range`, or `text`
- **Chart types**: valid chart type names
- **Semantic layer**:
  - model files under `semantic/` have a `name` and `source.table`
  - metric expressions are present and derived metric references exist
  - referenced dashboard models or aliases exist
  - semantic widgets reference valid metrics, dimensions, segments, filters, and sort fields
  - invalid semantic models only fail dashboards that reference them

## Exit Code

- `0` — all dashboards valid
- `1` — validation errors found
