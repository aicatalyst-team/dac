# Quickstart

Build your first dashboard in under 5 minutes.

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

## Next Steps

- Learn the full [YAML format](/dashboards/yaml)
- Use [TSX](/dashboards/tsx) for programmatic dashboards
- Add [filters](/dashboards/filters) for interactivity
- Add a `semantic/` directory and define a [semantic layer](/dashboards/semantic-layer)
