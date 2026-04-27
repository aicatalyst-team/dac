# dac init

Create a new DAC project scaffold.

```shell
dac init [path] [flags]
```

If `path` is omitted, DAC initializes the current directory.

## Flags

| Flag | Alias | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--template` | `-t` | string | `starter` | Project template: `starter`, `sql`, `semantic`, or `tsx` |
| `--force` | `-f` | bool | `false` | Overwrite scaffold files if they already exist |

## Examples

```shell
# Create the default starter project
dac init my-dashboards

# Create only a SQL-backed YAML dashboard
dac init my-sql-dashboard --template sql

# Create a semantic YAML dashboard and model
dac init my-semantic-dashboard --template semantic

# Create a semantic TSX dashboard and model
dac init my-tsx-dashboard --template tsx
```

## Generated Project

The default `starter` template creates:

```text
my-dashboards/
├── .bruin.yml
├── README.md
├── data/
│   └── .gitkeep
├── dashboards/
│   ├── sales.yml
│   └── semantic-sales.yml
└── semantic/
    └── sales.yml
```

The generated dashboards use a local DuckDB connection named `local_duckdb`. Starter queries include inline sample data, so there is no separate seed step.

## Next Steps

```shell
cd my-dashboards
dac validate --dir .
dac serve --dir . --open
```

To inspect a generated semantic widget from the command line:

```shell
dac query --dir . --dashboard "Semantic Sales" --widget "Revenue"
```
