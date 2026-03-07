import type { Widget as WidgetType, WidgetData } from "../types/dashboard";
import { MetricWidget } from "./widgets/MetricWidget";
import { ChartWidget } from "./widgets/ChartWidget";
import { TableWidget } from "./widgets/TableWidget";
import { TextWidget } from "./widgets/TextWidget";

interface WidgetProps {
  widget: WidgetType;
  data?: WidgetData;
  isLoading: boolean;
}

const containerClass: Record<string, string> = {
  metric: "py-3 px-4 h-full border-l border-[var(--dac-border)]",
  chart: "py-3 px-4 h-full border border-[var(--dac-border)] rounded",
  table: "py-3 h-full border border-[var(--dac-border)] rounded overflow-hidden",
  text: "py-3 h-full",
};

export function Widget({ widget, data, isLoading }: WidgetProps) {
  const isTable = widget.type === "table";

  return (
    <div className={containerClass[widget.type] ?? containerClass.text}>
      <div className={`text-[11px] font-medium uppercase tracking-wider text-[var(--dac-text-muted)] mb-1.5 ${isTable ? "px-4" : ""}`}>
        {widget.name}
      </div>

      {data?.error && (
        <div className={`text-xs text-[var(--dac-error)] font-mono mt-1 ${isTable ? "px-4" : ""}`}>{data.error}</div>
      )}

      {isLoading && !data && <LoadingSkeleton type={widget.type} />}

      {!isLoading && !data?.error && (
        <WidgetContent widget={widget} data={data} />
      )}
    </div>
  );
}

function LoadingSkeleton({ type }: { type: string }) {
  if (type === "metric") {
    return <div className="skeleton h-8 w-24 mt-1" />;
  }
  if (type === "chart") {
    return <div className="skeleton h-[240px] w-full mt-2 rounded" />;
  }
  if (type === "table") {
    return (
      <div className="mt-2 space-y-1.5 px-4">
        <div className="skeleton h-6 w-full" />
        <div className="skeleton h-5 w-full" />
        <div className="skeleton h-5 w-full" />
        <div className="skeleton h-5 w-3/4" />
      </div>
    );
  }
  return <div className="skeleton h-8 w-full" />;
}

function WidgetContent({ widget, data }: { widget: WidgetType; data?: WidgetData }) {
  switch (widget.type) {
    case "metric":
      return <MetricWidget widget={widget} data={data} />;
    case "chart":
      return <ChartWidget widget={widget} data={data} />;
    case "table":
      return <TableWidget widget={widget} data={data} />;
    case "text":
      return <TextWidget widget={widget} />;
    default:
      return <div className="text-[var(--dac-text-muted)] text-xs">Unknown widget type</div>;
  }
}
