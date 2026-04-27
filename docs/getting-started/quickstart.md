# Quickstart

Build your first dashboard in under 5 minutes.

DAC uses Bruin connections for query execution, so make sure `bruin` is installed and available on your `PATH` before starting.

## 0. Try a Bundled Example

If you cloned the repository, you can run one of the example projects immediately:

```shell
make deps
make build
./bin/dac serve --dir examples/basic-yaml
```

The curated examples live under `examples/`:

- `examples/basic-yaml`
- `examples/basic-tsx`
- `examples/semantic-yaml`
- `examples/semantic-tsx`

## 1. Create a Project

Use `dac init` to create a runnable starter project:

```shell
dac init my-dashboards
cd my-dashboards
```

The starter includes a SQL-backed YAML dashboard, a semantic YAML dashboard, a `semantic/sales.yml` model, and a `.bruin.yml` DuckDB connection. The generated queries include inline sample data, so there is no separate seed step.

## 2. Start the Server

```shell
dac serve --dir . --open
```

The dashboards will be available at `http://localhost:8321`.

## 3. Validate the Project

```shell
dac validate --dir .
```

## 4. Check Queries

```shell
dac check --dir .
```

## 5. Try Other Templates

`dac init` can also generate smaller starter projects:

```shell
dac init sql-only --template sql
dac init semantic-yaml --template semantic
dac init semantic-tsx --template tsx
```

## 6. Semantic Model Layout

Semantic models live in a sibling `semantic/` directory and are referenced from dashboard widgets by model name:

```text
my-dashboards/
├── .bruin.yml
├── dashboards/
│   └── semantic-sales.yml
└── semantic/
    └── sales.yml
```

You can try the bundled semantic example:

```shell
./bin/dac validate --dir examples/semantic-yaml
./bin/dac query --dir examples/semantic-yaml --dashboard "Semantic Sales Example" --widget "Revenue"
```

## Next Steps

- Learn the full [YAML format](/dashboards/yaml)
- Use [TSX](/dashboards/tsx) for programmatic dashboards
- Add [filters](/dashboards/filters) for interactivity
- Add a `semantic/` directory and define a [semantic layer](/dashboards/semantic-layer)
- Explore the runnable projects in [`examples/`](https://github.com/bruin-data/dac/tree/main/examples)
