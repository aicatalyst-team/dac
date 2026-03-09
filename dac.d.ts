/**
 * DAC — Dashboard-as-Code TypeScript Declarations
 *
 * Provides type definitions for .dashboard.tsx files.
 * Reference in your dashboard files:
 *   /// <reference path="dac.d.ts" />
 *
 * Or add to tsconfig.json:
 *   { "include": ["dac.d.ts", "**/*.dashboard.tsx"] }
 */

// ---------------------------------------------------------------------------
// JSX intrinsic elements
// ---------------------------------------------------------------------------

type ChartType =
  | "line"
  | "bar"
  | "area"
  | "pie"
  | "scatter"
  | "bubble"
  | "combo"
  | "histogram"
  | "boxplot"
  | "funnel"
  | "sankey"
  | "heatmap"
  | "calendar"
  | "sparkline"
  | "waterfall"
  | "xmr"
  | "dumbbell";

interface FilterOptionsProps {
  values?: string[];
  query?: string;
  connection?: string;
  presets?: string[];
}

interface SemanticSourceProps {
  table: string;
  dateColumn?: string;
  date_column?: string;
  dateFormat?: string;
  date_format?: string;
  connection?: string;
}

interface SemanticMetricProps {
  aggregate?: "count" | "count_distinct" | "sum" | "avg" | "min" | "max";
  column?: string;
  filter?: Record<string, string>;
  expression?: string;
}

interface SemanticDimensionProps {
  column: string;
  type?: "date";
}

interface TableColumnDef {
  name: string;
  label?: string;
  format?: string;
}

declare namespace JSX {
  interface IntrinsicElements {
    Dashboard: {
      name: string;
      description?: string;
      connection?: string;
      theme?: string;
      refresh?: { interval: string };
      children?: any;
    };

    Row: {
      height?: number | string;
      children?: any;
    };

    Filter: {
      name: string;
      type: "date-range" | "select" | "text";
      multiple?: boolean;
      default?: unknown;
      options?: FilterOptionsProps;
    };

    Query: {
      name: string;
      sql?: string;
      file?: string;
      connection?: string;
    };

    Semantic: {
      source?: SemanticSourceProps;
      metrics?: Record<string, SemanticMetricProps>;
      dimensions?: Record<string, SemanticDimensionProps>;
    };

    Metric: {
      name: string;
      col?: number;
      sql?: string;
      query?: string;
      file?: string;
      connection?: string;
      metric?: string;
      column?: string;
      prefix?: string;
      suffix?: string;
      format?: string;
    };

    Chart: {
      name: string;
      chart: ChartType;
      col?: number;
      sql?: string;
      query?: string;
      file?: string;
      connection?: string;
      x?: string;
      y?: string[];
      label?: string;
      value?: string;
      stacked?: boolean;
      size?: string;
      source?: string;
      target?: string;
      bins?: number;
      lines?: string[];
      yMin?: string;
      yMax?: string;
      dimension?: string;
      metrics?: string[];
      limit?: number;
    };

    Table: {
      name: string;
      col?: number;
      sql?: string;
      query?: string;
      file?: string;
      connection?: string;
      columns?: TableColumnDef[];
    };

    Text: {
      name: string;
      col?: number;
      content: string;
    };

    Divider: {
      name?: string;
      col?: number;
    };

    Image: {
      name: string;
      col?: number;
      src: string;
      alt?: string;
    };
  }

  type Element = any;
}

// ---------------------------------------------------------------------------
// Global functions available in .dashboard.tsx files
// ---------------------------------------------------------------------------

/**
 * Execute a SQL query at dashboard load time.
 * Returns the result synchronously as { columns, rows }.
 *
 * @param connection - The connection name (e.g., "duckdb", "bigquery")
 * @param sql - The SQL query to execute
 * @returns Query result with columns and rows
 *
 * @example
 * const tables = query("duckdb", "SELECT table_name FROM information_schema.tables")
 * // tables = { columns: [{name: "table_name"}], rows: [["users"], ["orders"]] }
 */
declare function query(
  connection: string,
  sql: string
): {
  columns: { name: string; type?: string }[];
  rows: unknown[][];
};

/**
 * Read a SQL file relative to the dashboard file.
 *
 * @param path - Relative path to the .sql file
 * @returns The file contents as a string
 *
 * @example
 * const sql = include("queries/revenue.sql")
 */
declare function include(path: string): string;

/**
 * Import a module (CommonJS require).
 * Supports .tsx, .ts, .jsx, .js, and .json files.
 * Paths are resolved relative to the importing file.
 *
 * @param path - Relative path to the module
 * @returns The module's exports
 *
 * @example
 * const { KPI } = require("./lib/kpi")
 */
declare function require(path: string): any;
