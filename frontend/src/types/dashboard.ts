export interface DashboardSummary {
  name: string;
  description?: string;
}

export interface Dashboard {
  name: string;
  description?: string;
  connection?: string;
  theme?: string;
  refresh?: { interval: string };
  filters?: Filter[];
  queries?: Record<string, Query>;
  rows: Row[];
}

export interface Filter {
  name: string;
  type: "date-range" | "select" | "text";
  multiple?: boolean;
  default?: unknown;
  options?: {
    values?: string[];
    query?: string;
    connection?: string;
  };
}

export interface Query {
  sql?: string;
  file?: string;
  connection?: string;
}

export interface Row {
  height?: number | string;
  widgets: Widget[];
}

export interface Widget {
  name: string;
  type: "metric" | "chart" | "table" | "text";
  col?: number;

  // Query source
  query?: string;
  sql?: string;
  file?: string;
  connection?: string;

  // Metric
  column?: string;
  prefix?: string;
  suffix?: string;
  format?: string;

  // Chart
  chart?: "line" | "bar" | "area" | "pie";
  x?: string;
  y?: string[];
  label?: string;
  value?: string;
  stacked?: boolean;

  // Table
  columns?: TableColumn[];

  // Text
  content?: string;
}

export interface TableColumn {
  name: string;
  label?: string;
  format?: string;
}

export interface WidgetData {
  columns: { name: string; type?: string }[];
  rows: unknown[][];
  error?: string;
}

export interface BatchDataResponse {
  widgets: Record<string, WidgetData>;
}
