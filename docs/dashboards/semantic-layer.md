# Semantic Layer

DAC loads semantic models from the project `semantic/` directory and compiles semantic widgets and named queries to SQL in the backend.

## Project Layout

```text
my-project/
├── .bruin.yml
├── dashboards/
│   └── sales.yml
└── semantic/
    └── sales.yml
```

Dashboard files reference semantic models by name. They do not generate SQL themselves.

## Model Files

Each semantic model is a separate `.yml` file in `semantic/`:

```yaml
name: sales
label: Sales
description: Semantic model over the sales table

source:
  table: sales

dimensions:
  - name: created_at
    type: time
    granularities:
      day: date_trunc('day', created_at)
      month: date_trunc('month', created_at)
  - name: region
    type: string
  - name: channel
    type: string

metrics:
  - name: revenue
    expression: sum(amount)
  - name: sales_count
    expression: count(*)
  - name: avg_sale_value
    expression: "{revenue} / {sales_count}"

segments:
  - name: online
    filter: "channel = 'online'"
```

## Model Fields

### Top Level

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Model name used by dashboards |
| `label` | string | Optional display label |
| `description` | string | Optional description |
| `source.table` | string | Base table for the model |

### Dimensions

Dimensions are fields you group and filter by.

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Dimension name |
| `type` | string | `string`, `number`, `boolean`, or `time` |
| `expression` | string | SQL expression. Defaults to the dimension name when omitted |
| `granularities` | map | Optional time bucket expressions such as `day`, `month`, or `year` |
| `label` | string | Optional display label |
| `description` | string | Optional description |
| `hidden` | bool | Hide from UI consumers |
| `group` | string | Optional grouping label |

### Metrics

Metrics are aggregated or derived values.

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Metric name |
| `expression` | string | SQL expression. Metric references use `{metric_name}` |
| `filter` | string | Optional SQL predicate applied to the metric |
| `format` | object | Optional formatting metadata |
| `window` | object | Optional window function metadata |
| `label` | string | Optional display label |
| `description` | string | Optional description |

### Segments

Segments are reusable filter predicates.

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Segment name |
| `filter` | string | SQL predicate |
| `label` | string | Optional display label |
| `description` | string | Optional description |

## Using Semantic Models in Dashboards

Set a default semantic model on the dashboard:

```yaml
name: Semantic Sales Example
connection: local_duckdb
model: sales
```

### Metric Widgets

```yaml
- name: Revenue
  type: metric
  metric: revenue
  filters:
    - dimension: region
      operator: equals
      value: "{{ filters.region }}"
```

### Chart Widgets

```yaml
- name: Revenue Trend
  type: chart
  chart: area
  dimension: created_at
  granularity: month
  metrics: [revenue]
  sort:
    - name: created_at
      direction: asc
```

### Table Widgets

```yaml
- name: Sales Breakdown
  type: table
  dimensions:
    - name: region
    - name: channel
  metrics: [revenue, sales_count]
  limit: 20
```

### Named Semantic Queries

```yaml
queries:
  online_by_region:
    dimensions:
      - name: region
    metrics: [revenue]
    segments: [online]
    sort:
      - name: revenue
        direction: desc
```

Widgets can then reference that query with `query: online_by_region`.

## Semantic Filters

Structured filters target semantic dimensions:

```yaml
filters:
  - dimension: created_at
    operator: between
    value:
      start: "{{ filters.date_range.start }}"
      end: "{{ filters.date_range.end }}"
```

Supported operators include:
- `equals`
- `not_equals`
- `gt`
- `gte`
- `lt`
- `lte`
- `in`
- `not_in`
- `between`
- `is_null`
- `is_not_null`

For advanced cases, filters can use `expression` instead of `dimension` plus `operator`.

## TSX Example

```tsx
export default (
  <Dashboard name="Semantic Sales" connection="local_duckdb" model="sales">
    <Query
      name="onlineByRegion"
      dimensions={[{ name: "region" }]}
      metrics={["revenue"]}
      segments={["online"]}
    />

    <Row>
      <Chart name="Online Revenue by Region" chart="bar" query="onlineByRegion" col={6} />
      <Metric name="Revenue" metric="revenue" col={6} />
    </Row>
  </Dashboard>
)
```

`<Query />` is a declaration node in the dashboard DSL. It defines a named query; it does not render a UI component.

## Backend Compilation

Semantic widgets and named semantic queries go through the backend REST API. The backend:

1. renders templated filter values
2. validates models, dimensions, metrics, segments, and filters
3. compiles the semantic query to SQL
4. executes the generated SQL against the selected connection

This keeps SQL generation in the application backend rather than in dashboard files.
