# dac check

Validate dashboards **and** execute all widget queries. This is a deeper check than `validate` — it verifies that queries actually run against your database.

```shell
dac check [flags]
```

## Flags

| Flag | Alias | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--dir` | `-d` | string | `.` | Dashboard definitions directory |

## Examples

```shell
# Check all dashboards
dac check

# Check dashboards in a specific directory
dac check --dir ./dashboards
```

## Output

For each widget, the output shows:

```
Sales Analytics
  ✓ Total Revenue          1 row, 1 col    (45ms)
  ✓ Total Orders           1 row, 1 col    (32ms)
  ✓ Revenue Trend          12 rows, 2 cols (128ms)
  ✗ Broken Widget          ERROR: relation "missing_table" does not exist
```

- **Row and column counts** for successful queries
- **Execution time** for each query
- **Error messages** for failed queries

## Use Cases

- **CI/CD validation**: Run `dac check` in your pipeline to catch broken queries before deploying
- **Post-migration testing**: After a database migration, verify all dashboards still work
- **Development**: Quick feedback on whether your SQL is correct
