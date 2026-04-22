# dac connections

Test all database connections defined in `.bruin.yml`. Connections are tested in parallel.

```shell
dac connections [flags]
```

## Flags

| Flag | Alias | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--dir` | `-d` | string | `.` | Dashboard definitions directory |

## Examples

```shell
dac connections
```

## Output

```
Name          Type       Status
my_db         duckdb     ✓ Connected
warehouse     bigquery   ✓ Connected
legacy_db     postgres   ✗ connection refused
```
