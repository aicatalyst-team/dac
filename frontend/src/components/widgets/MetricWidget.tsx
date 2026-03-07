import type { Widget, WidgetData } from "../../types/dashboard";

interface Props {
  widget: Widget;
  data?: WidgetData;
}

function formatValue(value: unknown, format?: string): string {
  if (value === null || value === undefined) return "—";
  const num = Number(value);
  if (isNaN(num)) return String(value);

  switch (format) {
    case "number":
      return num.toLocaleString();
    case "currency":
      return num.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 });
    case "percent":
      return num.toLocaleString(undefined, { minimumFractionDigits: 1, maximumFractionDigits: 1 });
    case "compact":
      return Intl.NumberFormat(undefined, { notation: "compact" }).format(num);
    default:
      return num.toLocaleString();
  }
}

export function MetricWidget({ widget, data }: Props) {
  if (!data?.rows?.length || !data.columns?.length) {
    return (
      <div className="h-12">
        <div className="skeleton h-8 w-28" />
      </div>
    );
  }

  const colIdx = data.columns.findIndex((c) => c.name === widget.column);
  const rawValue = colIdx >= 0 ? data.rows[0][colIdx] : data.rows[0][0];
  const formatted = formatValue(rawValue, widget.format);

  return (
    <div className="tabular-nums">
      <div className="flex items-baseline gap-1">
        {widget.prefix && (
          <span className="text-base font-normal text-[var(--dac-text-muted)]">{widget.prefix}</span>
        )}
        <span
          className="font-semibold tracking-tight text-[var(--dac-text-primary)]"
          style={{ fontSize: "clamp(1.35rem, 2.5vw, 2rem)", lineHeight: 1.1 }}
        >
          {formatted}
        </span>
        {widget.suffix && (
          <span className="text-sm font-normal text-[var(--dac-text-muted)]">{widget.suffix}</span>
        )}
      </div>
    </div>
  );
}
