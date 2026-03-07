---
name: create-dashboard
description: Create dac dashboards by writing YAML definition files and SQL queries. Use when the user wants to create, modify, or understand dashboard YAML files, widget configuration, filters, query templating, or CLI usage.
argument-hint: "[description of the dashboard to create]"
---

# Create Dashboard

Create dac dashboards by writing YAML files and SQL queries. This skill covers the full dashboard YAML schema, widget types, filters, query templating, and project setup.

When invoked, use `$ARGUMENTS` as the description of what dashboard to create.

---

## Project Structure

```
my-project/
  .bruin.yml                    # Database connections (required for queries)
  dashboards/
    my-dashboard.yml            # Any *.yml file = a dashboard
    another.yml
    queries/
      my_query.sql              # Shared SQL files (referenced from YAML)
  themes/
    corporate.yml               # Optional custom themes (token overrides)
```

- Any `*.yml` file in the dashboard directory is auto-discovered as a dashboard.
- Files starting with `.` are ignored (e.g. `.bruin.yml`).
- SQL files can live in `queries/` or any subdirectory — referenced by relative path.

---

## .bruin.yml — Connection Config

The `.bruin.yml` file defines database connections. It must exist somewhere above or in the dashboard directory. The CLI auto-discovers it by walking up the directory tree.

```yaml
default_environment: default
environments:
  default:
    connections:
      duckdb:
        - name: my_duckdb
          path: /absolute/path/to/data.db
          read_only: true       # recommended for DuckDB to avoid lock issues

      postgres:
        - name: my_postgres
          host: localhost
          port: 5432
          database: analytics
          username: user
          password: pass

      # Other connection types supported by bruin CLI
```

Queries are executed via `bruin query` under the hood. Any connection type supported by bruin works.

---

## Dashboard YAML Schema

```yaml
# Required
name: My Dashboard                    # Display name, also used as URL slug
rows: []                              # At least one row required

# Optional
description: A description            # Shown on the dashboard list page
connection: my_duckdb                 # Default connection for all queries
theme: bruin                          # Template name (bruin, bruin-dark, or custom)

# Optional: refresh config
refresh:
  interval: 5m                        # Cache TTL for query results

# Optional: interactive filters
filters: []

# Optional: named queries (reusable across widgets)
queries: {}
```

---

## Filters

Filters create interactive controls in the UI. Filter values are injected into SQL queries via Jinja templating.

```yaml
filters:
  # Select dropdown
  - name: region
    type: select
    default: "All"                     # Initial value
    multiple: false                    # true for multi-select
    options:
      values: ["All", "North America", "Europe", "APAC"]   # Static options
      # OR dynamic options from a query:
      # query: SELECT DISTINCT region FROM dim_regions ORDER BY region
      # connection: my_postgres        # Optional connection override

  # Date range picker (with preset default)
  - name: date_range
    type: date-range
    default: last_30_days              # Preset name OR explicit {start, end}
    # default:                         # Explicit dates also work:
    #   start: "2025-01-01"
    #   end: "2025-12-31"
    options:
      presets:                         # Optional: control which presets appear
        - last_7_days
        - last_30_days
        - last_90_days
        - this_month
        - this_year

  # Free text input
  - name: search
    type: text
    default: ""
```

**Filter types:** `select`, `date-range`, `text`

**Date range presets:** `today`, `yesterday`, `last_7_days`, `last_30_days`, `last_90_days`, `this_month`, `last_month`, `this_quarter`, `this_year`, `year_to_date`, `all_time`. If `options.presets` is omitted, a default set is shown. Users can always pick "Custom range" for arbitrary dates.

---

## Named Queries

Define reusable queries that multiple widgets can reference:

```yaml
queries:
  total_revenue:
    sql: |
      SELECT SUM(amount) as value FROM sales
      WHERE created_at >= '{{ filters.date_range.start }}'

  revenue_by_month:
    file: queries/revenue_by_month.sql       # Relative to the YAML file
    connection: my_postgres                  # Optional connection override
```

---

## Rows and Grid

Dashboards use a **12-column grid**. Each row contains widgets whose `col` values should sum to 12 (or less). If `col` is omitted, widgets share space equally.

```yaml
rows:
  - widgets:
      - name: Widget A
        col: 8                # Takes 8/12 columns
        # ...
      - name: Widget B
        col: 4                # Takes 4/12 columns
        # ...

  - widgets:
      - name: Full Width
        col: 12               # Full width
        # ...
```

---

## Widget Types

Every widget (except `text`, `divider`, and `image`) needs a query source. **Priority order:**
1. `query: <name>` — reference a named query from the `queries:` map
2. `sql: |` — inline SQL
3. `file: path/to/query.sql` — external SQL file (relative to the YAML)

### Metric Widget

Single KPI number card.

```yaml
- name: Total Revenue
  type: metric
  query: total_revenue          # or sql: / file:
  column: value                 # REQUIRED: which result column to display
  prefix: "$"                   # Optional: shown before the number
  suffix: "%"                   # Optional: shown after the number
  format: number                # Optional: "number" for locale formatting
  col: 3
```

The SQL must return at least one row. The value from `column` in the first row is displayed.

### Chart Widget

Visualizations using Recharts. **17 chart types** available.

#### Line / Bar / Area

```yaml
- name: Revenue Trend
  type: chart
  chart: line                   # line | bar | area
  sql: |
    SELECT month, revenue, target FROM monthly_data ORDER BY month
  x: month                     # REQUIRED: column for X axis
  y: [revenue, target]         # REQUIRED: column(s) for Y axis (array)
  col: 8
```

#### Stacked Bar / Area

```yaml
- name: Sales by Region
  type: chart
  chart: bar
  stacked: true                 # Stacks the Y series
  sql: |
    SELECT month,
      SUM(CASE WHEN region='NA' THEN amount ELSE 0 END) AS "North America",
      SUM(CASE WHEN region='EU' THEN amount ELSE 0 END) AS "Europe"
    FROM sales GROUP BY 1 ORDER BY 1
  x: month
  y: ["North America", "Europe"]
  col: 6
```

#### Pie

```yaml
- name: Revenue by Region
  type: chart
  chart: pie
  sql: |
    SELECT region, SUM(amount) as total FROM sales GROUP BY 1
  label: region                 # REQUIRED: category column
  value: total                  # REQUIRED: numeric column
  col: 4
```

#### Scatter

```yaml
- name: Price vs Quantity
  type: chart
  chart: scatter
  sql: SELECT price, quantity FROM orders
  x: price
  y: [quantity]
  col: 6
```

X axis auto-detects numeric vs category data.

#### Bubble

```yaml
- name: Sales Bubble
  type: chart
  chart: bubble
  sql: SELECT region, revenue, profit, order_count FROM summary
  x: region                     # X axis
  y: [revenue]                  # Y axis
  size: order_count             # REQUIRED: bubble size column
  col: 6
```

#### Combo (mixed bar + line)

```yaml
- name: Revenue vs Growth
  type: chart
  chart: combo
  sql: SELECT month, revenue, growth_pct FROM monthly
  x: month
  y: [revenue, growth_pct]
  lines: [growth_pct]           # Which y series render as lines (rest are bars)
  col: 8
```

#### Histogram

```yaml
- name: Order Distribution
  type: chart
  chart: histogram
  sql: SELECT amount FROM orders
  x: amount                     # REQUIRED: column to bin
  bins: 20                      # Optional: number of bins (default: 10)
  col: 6
```

Client-side binning of raw data values.

#### Boxplot

```yaml
- name: Amount by Status
  type: chart
  chart: boxplot
  sql: SELECT status, amount FROM orders
  x: status                     # Category column
  y: [amount]                   # Numeric column
  col: 6
```

Client-side quartile computation from raw data rows.

#### Funnel

```yaml
- name: Conversion Funnel
  type: chart
  chart: funnel
  sql: SELECT stage, count FROM funnel_data ORDER BY count DESC
  label: stage                  # REQUIRED: category column
  value: count                  # REQUIRED: numeric column
  col: 6
```

#### Sankey

```yaml
- name: Flow Diagram
  type: chart
  chart: sankey
  sql: SELECT source_stage, target_stage, flow_count FROM flows
  source: source_stage          # REQUIRED: source node column
  target: target_stage          # REQUIRED: target node column
  value: flow_count             # REQUIRED: flow weight column
  col: 8
```

#### Heatmap

```yaml
- name: Activity Heatmap
  type: chart
  chart: heatmap
  sql: SELECT day_of_week, hour, event_count FROM activity
  x: hour                       # REQUIRED: X axis column
  y: [day_of_week]              # REQUIRED: Y axis column (array with 1 element)
  value: event_count            # REQUIRED: intensity column
  col: 8
```

Custom SVG rendering with hover tooltips.

#### Calendar

```yaml
- name: Daily Revenue
  type: chart
  chart: calendar
  sql: SELECT date, revenue FROM daily_sales
  x: date                       # REQUIRED: date column (YYYY-MM-DD)
  value: revenue                # REQUIRED: intensity column
  col: 12
```

GitHub-style calendar heatmap, custom SVG.

#### Sparkline

```yaml
- name: Revenue Sparkline
  type: chart
  chart: sparkline
  sql: SELECT month, revenue FROM monthly ORDER BY month
  x: month
  y: [revenue]
  col: 3
```

Compact line chart (60px height), no axes or labels. Great for KPI rows.

#### Waterfall

```yaml
- name: P&L Waterfall
  type: chart
  chart: waterfall
  sql: |
    SELECT category, amount FROM pnl
    ORDER BY CASE category
      WHEN 'Revenue' THEN 1
      WHEN 'COGS' THEN 2
      WHEN 'OpEx' THEN 3
      WHEN 'Net' THEN 4
    END
  x: category
  y: [amount]
  col: 8
```

Positive values shown in one color, negative in another. Bars float to show cumulative effect.

#### XMR (Control Chart)

```yaml
- name: Process Control
  type: chart
  chart: xmr
  sql: SELECT date, value, mean, ucl, lcl FROM process_data
  x: date
  y: [value, mean]              # First = data line, second = center line (dashed)
  yMin: lcl                     # Lower control limit (dashed)
  yMax: ucl                     # Upper control limit (dashed)
  col: 8
```

#### Dumbbell

```yaml
- name: H1 vs H2 Revenue
  type: chart
  chart: dumbbell
  sql: SELECT region, h1_revenue, h2_revenue FROM comparison
  x: region                     # Category column (vertical axis)
  y: [h1_revenue, h2_revenue]   # Two numeric columns (start and end points)
  col: 6
```

Horizontal chart showing range between two values per category.

### Table Widget

Data table with optional column configuration.

```yaml
- name: Recent Orders
  type: table
  file: queries/recent_orders.sql
  columns:                      # Optional: customize column display
    - name: customer_name       # Must match SQL column name
      label: Customer           # Display header
    - name: amount
      label: Amount
      format: currency          # "currency" adds $ prefix, "number" for locale formatting
    - name: created_at
      label: Date               # ISO dates auto-format to readable strings
  col: 12
```

If `columns` is omitted, all result columns are shown with their SQL names as headers.

### Text Widget

Static content with markdown formatting. No query needed.

```yaml
- name: Notes
  type: text
  content: |
    # Section Header

    **Important:** Revenue figures are updated daily.

    Data source: Snowflake `analytics.sales`

    - Bullet point one
    - Bullet point two

    1. Ordered item
    2. Another item

    > This is a blockquote for callouts

    Visit [our docs](https://example.com) for details.

    ---

    *Italic text*, **bold text**, ~~strikethrough~~, and `inline code`.
  col: 12
```

**Supported markdown syntax:**
- Headers: `#` through `######`
- Bold: `**text**` or `__text__`
- Italic: `*text*` or `_text_`
- Bold italic: `***text***`
- Strikethrough: `~~text~~`
- Inline code: `` `code` ``
- Links: `[text](url)`
- Images: `![alt](src)`
- Unordered lists: `- item` or `* item`
- Ordered lists: `1. item`
- Blockquotes: `> text`
- Horizontal rules: `---`, `***`, or `___`

### Divider Widget

A visual horizontal separator line. No query or content needed.

```yaml
- name: separator
  type: divider
  col: 12
```

Use dividers to visually separate sections within a dashboard.

### Image Widget

Displays an image from a URL. No query needed.

```yaml
- name: Company Logo
  type: image
  src: https://example.com/logo.png    # REQUIRED: image URL
  alt: Company logo                     # Optional: alt text
  col: 4
```

---

## Query Templating (Jinja)

SQL queries support Jinja syntax for filter variable substitution. Filter values are available under `filters.<name>`.

**Variable interpolation:**
```sql
WHERE created_at >= '{{ filters.date_range.start }}'
  AND created_at <= '{{ filters.date_range.end }}'
```

**Conditionals:**
```sql
{% if filters.region != 'All' %}
  AND region = '{{ filters.region }}'
{% endif %}
```

**Accessing nested values (date-range):**
```sql
{{ filters.date_range.start }}
{{ filters.date_range.end }}
```

**Accessing simple values (select, text):**
```sql
{{ filters.region }}
{{ filters.search }}
```

---

## CLI Commands

### `dac serve` — Start dev server

```bash
dac serve --dir ./dashboards
dac serve --dir ./dashboards --port 9000 --template bruin-dark
dac serve --dir ./dashboards --template ./themes/corporate.yml
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--port` | `-p` | `8321` | Port (auto-increments if taken) |
| `--dir` | `-d` | `.` | Dashboard directory |
| `--template` | `-t` | `bruin` | Template: `bruin`, `bruin-dark`, or path to `.yml` |
| `--host` | | `localhost` | Bind host |
| `--open` | | `false` | Open browser |

### `dac validate` — Validate YAML structure

```bash
dac validate --dir ./dashboards
```

Checks YAML syntax, required fields, column sums, and query references. Does not execute queries.

### `dac check` — Deep validation (YAML + execute all queries)

```bash
dac check --dir ./dashboards
```

Goes beyond `validate`: parses YAML, resolves all query references, applies default filter values, executes every query, and reports results with row/column counts and timing. Catches SQL errors, missing tables, bad column names.

### `dac query` — Run a SQL query

```bash
# Inline SQL
dac query -c local_duckdb "SELECT * FROM sales LIMIT 5"

# From a .sql file
dac query -c local_duckdb -f queries/my_query.sql

# Run a specific widget's query (resolves named refs, applies default filters)
dac query -d ./dashboards --dashboard "Sales Analytics" --widget "Total Revenue"

# Output formats: table (default), json, csv
dac query -c local_duckdb "SELECT 1" -o json
```

| Flag | Short | Description |
|------|-------|-------------|
| `--connection` | `-c` | Connection name |
| `--file` | `-f` | Path to `.sql` file |
| `--dashboard` | | Dashboard name (with `--widget`) |
| `--widget` | `-w` | Widget name (with `--dashboard`) |
| `--output` | `-o` | Output format: `table`, `json`, `csv` |
| `--dir` | `-d` | Dashboard directory (for `--dashboard`) |

### `dac ls` — List dashboards

```bash
dac ls --dir ./dashboards
```

Shows all discovered dashboards with widget count, filter count, and connection.

### `dac connections` — Test connections

```bash
dac connections --dir ./dashboards
```

Tests each connection in `.bruin.yml` by running `SELECT 1`. Reports connection status.

### Global flags

| Flag | Short | Description |
|------|-------|-------------|
| `--config` | `-c` | Path to `.bruin.yml` (default: auto-discover) |
| `--environment` | `-e` | Target environment name |
| `--debug` | | Enable debug logging |

---

## Custom Themes

Create a `themes/*.yml` file in the dashboard directory for token overrides:

```yaml
# themes/corporate.yml
name: corporate
extends: bruin                        # Inherit from built-in template
tokens:
  background: "#FAFAFA"
  surface: "#FFFFFF"
  border: "#E5E7EB"
  text-primary: "#111827"
  accent: "#0052CC"
  chart-1: "#0052CC"
  chart-2: "#00B8D9"
  chart-3: "#8B5CF6"
  # Missing tokens fall back to the base template
```

**Available tokens:** `background`, `surface`, `surface-hover`, `border`, `text-primary`, `text-secondary`, `text-muted`, `accent`, `accent-hover`, `accent-subtle`, `success`, `warning`, `error`, `chart-1` through `chart-8`.

---

## Complete Example

```yaml
name: Sales Analytics
description: Real-time sales performance
connection: local_duckdb

filters:
  - name: region
    type: select
    default: "All"
    options:
      values: ["All", "North America", "Europe", "APAC"]
  - name: date_range
    type: date-range
    default: this_year

queries:
  total_revenue:
    sql: |
      SELECT SUM(amount) as value FROM sales
      WHERE created_at >= '{{ filters.date_range.start }}'
        AND created_at <= '{{ filters.date_range.end }}'
      {% if filters.region != 'All' %}
        AND region = '{{ filters.region }}'
      {% endif %}

rows:
  # KPI row
  - widgets:
      - name: Total Revenue
        type: metric
        query: total_revenue
        column: value
        prefix: "$"
        format: number
        col: 4
      - name: Total Orders
        type: metric
        col: 4
        sql: SELECT COUNT(*) as total FROM orders
        column: total
        format: number
      - name: Avg Order
        type: metric
        col: 4
        sql: SELECT ROUND(AVG(amount), 2) as avg FROM orders
        column: avg
        prefix: "$"
        format: number

  # Section divider
  - widgets:
      - name: divider
        type: divider
        col: 12

  # Charts
  - widgets:
      - name: Revenue Trend
        type: chart
        chart: area
        file: queries/revenue_by_month.sql
        x: month
        y: [revenue]
        col: 8
      - name: By Region
        type: chart
        chart: pie
        col: 4
        sql: |
          SELECT region, SUM(amount) as total
          FROM sales GROUP BY 1
        label: region
        value: total

  # Explanation text
  - widgets:
      - name: Data Notes
        type: text
        content: |
          **Note:** Revenue figures are refreshed every 5 minutes.
          See [documentation](https://example.com) for methodology.
        col: 12

  # Table
  - widgets:
      - name: Recent Orders
        type: table
        col: 12
        sql: |
          SELECT id, customer_name, amount, status, created_at
          FROM orders ORDER BY created_at DESC LIMIT 20
        columns:
          - name: id
            label: Order ID
          - name: customer_name
            label: Customer
          - name: amount
            label: Amount
            format: currency
          - name: status
            label: Status
          - name: created_at
            label: Date
```

---

## Widget Type Reference

| Type | Required Fields | Query Needed | Description |
|------|----------------|--------------|-------------|
| `metric` | `column` | Yes | Single KPI number card |
| `chart` | `chart` + chart-specific | Yes | Visualization (17 chart types) |
| `table` | — | Yes | Data table with optional column config |
| `text` | `content` | No | Markdown/text content |
| `divider` | — | No | Horizontal separator line |
| `image` | `src` | No | Image from URL |

### Chart Types

| Chart | Required | Optional | Description |
|-------|----------|----------|-------------|
| `line` | `x`, `y` | | Line chart |
| `bar` | `x`, `y` | `stacked` | Bar chart |
| `area` | `x`, `y` | `stacked` | Area chart |
| `pie` | `label`, `value` | | Pie/donut chart |
| `scatter` | `x`, `y` | | Scatter plot |
| `bubble` | `x`, `y`, `size` | | Bubble chart |
| `combo` | `x`, `y`, `lines` | | Mixed bar + line chart |
| `histogram` | `x` | `bins` | Histogram (client-side binning) |
| `boxplot` | `x`, `y` | | Box-and-whisker plot (client-side quartiles) |
| `funnel` | `label`, `value` | | Funnel chart |
| `sankey` | `source`, `target`, `value` | | Sankey/flow diagram |
| `heatmap` | `x`, `y`, `value` | | Grid heatmap |
| `calendar` | `x`, `value` | | Calendar heatmap (GitHub-style) |
| `sparkline` | `x`, `y` | | Compact inline line (60px) |
| `waterfall` | `x`, `y` | | Waterfall chart |
| `xmr` | `x`, `y` | `yMin`, `yMax` | Control chart with limits |
| `dumbbell` | `x`, `y` (2 fields) | | Horizontal range comparison |

---

## Validation Rules

- `name` is required on the dashboard and every widget.
- At least one row is required; each row needs at least one widget.
- `col` must be 1-12; total per row must not exceed 12.
- Every widget that requires data needs a query source (`query`, `sql`, or `file`).
- `metric` widgets require `column`.
- `chart` widgets require `chart` type plus the chart-specific fields listed above.
- `text` widgets require `content`.
- `image` widgets require `src`.
- `divider` widgets have no required fields.
- Filter types must be one of: `select`, `date-range`, `text`.
- Named query references (`query: name`) must exist in the `queries:` map.

Run `dac validate` for structure checks, or `dac check` to also execute all queries and verify they return data.
