# Quickstart

Create and run your first DAC project with `dac init`.

DAC uses Bruin connections for query execution, so make sure both `dac` and `bruin` are installed and available on your `PATH` before starting.

## 1. Create a Project

```shell
dac init my-dashboards
cd my-dashboards
```

The default starter creates a complete runnable project:

- `.bruin.yml` with a read-only DuckDB connection named `local_duckdb`
- `data/dac-demo.duckdb` for local testing
- `dashboards/sales.yml` for a regular SQL dashboard
- `dashboards/semantic-sales.yml` for a semantic dashboard
- `semantic/sales.yml` for the semantic model
- `.claude/skills/create-dashboard/SKILL.md` and `.codex/skills/create-dashboard`

The generated dashboards include inline sample data, so there is no separate seed step.

## 2. Validate the Project

```shell
dac validate --dir .
```

Validation checks dashboard YAML, semantic model YAML, schema versions, semantic references, and dashboard structure before anything is served. When `schema` is omitted, DAC assumes v1.

To also validate the generated SQL against your local DuckDB connection without fetching rows:

```shell
dac validate --dir . --with-database
```

## 3. Inspect a Widget Query

```shell
dac query --dir . --dashboard "Semantic Sales" --widget "Revenue"
```

For semantic widgets, DAC compiles the semantic model reference into SQL in the backend before executing the query.

## 4. Check All Widgets

```shell
dac check --dir .
```

`dac check` executes every query-backed widget and reports query failures before a viewer opens the dashboard.

## 5. Start the Server

```shell
dac serve --dir . --open
```

The dashboard app will be available at `http://localhost:8321`.

## 6. Try Other Templates

`dac init` can also generate smaller starter projects:

```shell
dac init sql-only --template sql
dac init semantic-yaml --template semantic
dac init semantic-tsx --template tsx
```

## 7. Existing Projects

If you already have a DAC project, install or refresh the bundled dashboard authoring skills with:

```shell
dac skills install --dir .
```

## Next Steps

- Learn the full [YAML format](/dashboards/yaml)
- Use [TSX](/dashboards/tsx) for programmatic dashboards
- Add [filters](/dashboards/filters) for interactivity
- Add a `semantic/` directory and define a [semantic layer](/dashboards/semantic-layer)
- Explore the runnable projects in [`examples/`](https://github.com/bruin-data/dac/tree/main/examples)
