# dac ls

List all discovered dashboards in a directory.

```shell
dac ls [flags]
```

## Flags

| Flag | Alias | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--dir` | `-d` | string | `.` | Dashboard definitions directory |

## Examples

```shell
dac ls
```

## Output

```
Name                Widgets   Filters   Connections
Sales Analytics     14        2         1
Chart Showcase      18        0         1
Business Summary    12        2         1
```
