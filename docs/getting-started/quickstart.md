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

Create a DAC project with a `dashboards/` directory:

```shell
mkdir -p my-dashboards/dashboards && cd my-dashboards
```

Create `.bruin.yml`:

```yaml
default_environment: default

environments:
  default:
    connections:
      duckdb:
        - name: my_db
          path: data.duckdb
```

## 2. Create a Dashboard

Create `dashboards/sales.yml`:

```yaml
name: Sales Overview
description: A simple sales dashboard
connection: my_db

rows:
  - widgets:
      - name: Total Revenue
        type: metric
        col: 4
        sql: SELECT SUM(amount) AS value FROM sales
        column: value
        prefix: "$"
        format: number

      - name: Order Count
        type: metric
        col: 4
        sql: SELECT COUNT(*) AS value FROM orders
        column: value
        format: number

      - name: Avg Order Value
        type: metric
        col: 4
        sql: SELECT ROUND(AVG(amount), 2) AS value FROM orders
        column: value
        prefix: "$"
        format: number

  - widgets:
      - name: Revenue Over Time
        type: chart
        chart: area
        col: 8
        sql: |
          SELECT
            STRFTIME(DATE_TRUNC('month', created_at), '%Y-%m') AS month,
            SUM(amount) AS revenue
          FROM sales
          GROUP BY 1
          ORDER BY 1
        x: month
        y: [revenue]

      - name: Revenue by Region
        type: chart
        chart: pie
        col: 4
        sql: |
          SELECT region, SUM(amount) AS total
          FROM sales
          GROUP BY 1
        label: region
        value: total
```

## 3. Start the Server

```shell
dac serve --dir . --open
```

The dashboard will be available at `http://localhost:8321`.

## 4. Validate the Project

```shell
dac validate --dir .
```

## 5. Check Queries

```shell
dac check --dir .
```

## 6. Try Semantic Models

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
