import {
  LineChart, Line, BarChart, Bar, AreaChart, Area, PieChart, Pie, Cell,
  XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend,
} from "recharts";
import type { Widget, WidgetData } from "../../types/dashboard";
import { useTheme } from "../../themes/ThemeProvider";

interface Props {
  widget: Widget;
  data?: WidgetData;
}

function toChartData(data: WidgetData): Record<string, unknown>[] {
  return data.rows.map((row) => {
    const obj: Record<string, unknown> = {};
    data.columns.forEach((col, i) => {
      obj[col.name] = row[i];
    });
    return obj;
  });
}

function formatAxisTick(value: unknown): string {
  const s = String(value);
  const isoMatch = s.match(/^(\d{4})-(\d{2})-(\d{2})T/);
  if (isoMatch) {
    const [, , m, d] = isoMatch;
    const months = ["Jan","Feb","Mar","Apr","May","Jun","Jul","Aug","Sep","Oct","Nov","Dec"];
    const monthIdx = parseInt(m, 10) - 1;
    const day = parseInt(d, 10);
    if (day === 1) return months[monthIdx];
    return `${months[monthIdx]} ${day}`;
  }
  const monthMatch = s.match(/^(\d{4})-(\d{2})$/);
  if (monthMatch) {
    const months = ["Jan","Feb","Mar","Apr","May","Jun","Jul","Aug","Sep","Oct","Nov","Dec"];
    return months[parseInt(monthMatch[2], 10) - 1] ?? s;
  }
  const dateMatch = s.match(/^(\d{4})-(\d{2})-(\d{2})$/);
  if (dateMatch) {
    const months = ["Jan","Feb","Mar","Apr","May","Jun","Jul","Aug","Sep","Oct","Nov","Dec"];
    return `${months[parseInt(dateMatch[2], 10) - 1]} ${parseInt(dateMatch[3], 10)}`;
  }
  return s;
}

function formatTooltipValue(value: unknown): string {
  const num = Number(value);
  if (!isNaN(num)) return num.toLocaleString();
  return String(value);
}

function formatYTick(value: unknown): string {
  const num = Number(value);
  if (isNaN(num)) return String(value);
  if (Math.abs(num) >= 1_000_000) return `${(num / 1_000_000).toFixed(1)}M`;
  if (Math.abs(num) >= 1_000) return `${(num / 1_000).toFixed(0)}k`;
  return num.toLocaleString();
}

const CHART_COLORS = [
  "chart-1", "chart-2", "chart-3", "chart-4",
  "chart-5", "chart-6", "chart-7", "chart-8",
];

const AXIS_STYLE = { fontSize: 11, fontFamily: '"Geist", system-ui' };

function CustomTooltip({ active, payload, label }: any) {
  if (!active || !payload?.length) return null;
  return (
    <div className="dac-tooltip">
      <div className="dac-tooltip-label">{formatAxisTick(label)}</div>
      {payload.map((p: any, i: number) => (
        <div key={i} className="dac-tooltip-row">
          <span className="dac-tooltip-dot" style={{ background: p.color }} />
          <span className="dac-tooltip-name">{p.name ?? p.dataKey}</span>
          <span className="dac-tooltip-value">{formatTooltipValue(p.value)}</span>
        </div>
      ))}
    </div>
  );
}

export function ChartWidget({ widget, data }: Props) {
  const theme = useTheme();

  if (!data?.rows?.length) {
    return <div className="text-[var(--dac-text-muted)] text-xs py-6 text-center">No data</div>;
  }

  const chartData = toChartData(data);
  const colors = CHART_COLORS.map((key) => theme.tokens[key] || "#888");
  const gridColor = theme.tokens["border"];
  const axisColor = theme.tokens["text-muted"];

  const commonAxisProps = {
    tick: { ...AXIS_STYLE, fill: axisColor },
    axisLine: false,
    tickLine: false,
  };

  switch (widget.chart) {
    case "line":
      return (
        <ResponsiveContainer width="100%" height={240}>
          <LineChart data={chartData} margin={{ top: 4, right: 8, bottom: 4, left: -4 }}>
            <CartesianGrid vertical={false} stroke={gridColor} strokeOpacity={0.5} strokeDasharray="3 3" />
            <XAxis dataKey={widget.x} {...commonAxisProps} dy={6} tickFormatter={formatAxisTick} />
            <YAxis {...commonAxisProps} dx={-4} tickFormatter={formatYTick} />
            <Tooltip content={<CustomTooltip />} />
            {widget.y?.map((field, i) => (
              <Line
                key={field}
                type="monotone"
                dataKey={field}
                stroke={colors[i % colors.length]}
                strokeWidth={1.5}
                dot={false}
                activeDot={{ r: 3, strokeWidth: 0 }}
              />
            ))}
          </LineChart>
        </ResponsiveContainer>
      );

    case "bar": {
      const yFields = widget.y ?? [];
      const isStacked = widget.stacked && yFields.length > 1;
      return (
        <ResponsiveContainer width="100%" height={240}>
          <BarChart data={chartData} margin={{ top: 4, right: 8, bottom: 4, left: -4 }}>
            <CartesianGrid vertical={false} stroke={gridColor} strokeOpacity={0.5} strokeDasharray="3 3" />
            <XAxis dataKey={widget.x} {...commonAxisProps} dy={6} tickFormatter={formatAxisTick} />
            <YAxis {...commonAxisProps} dx={-4} tickFormatter={formatYTick} />
            <Tooltip content={<CustomTooltip />} cursor={{ fill: gridColor, fillOpacity: 0.2 }} />
            {isStacked && <Legend wrapperStyle={{ fontSize: 11, fontFamily: '"Geist", system-ui' }} iconSize={7} />}
            {yFields.map((field, i) => (
              <Bar
                key={field}
                dataKey={field}
                fill={colors[i % colors.length]}
                stackId={isStacked ? "stack" : undefined}
                radius={isStacked && i < yFields.length - 1 ? undefined : [2, 2, 0, 0]}
              />
            ))}
          </BarChart>
        </ResponsiveContainer>
      );
    }

    case "area":
      return (
        <ResponsiveContainer width="100%" height={240}>
          <AreaChart data={chartData} margin={{ top: 4, right: 8, bottom: 4, left: -4 }}>
            <CartesianGrid vertical={false} stroke={gridColor} strokeOpacity={0.5} strokeDasharray="3 3" />
            <XAxis dataKey={widget.x} {...commonAxisProps} dy={6} tickFormatter={formatAxisTick} />
            <YAxis {...commonAxisProps} dx={-4} tickFormatter={formatYTick} />
            <Tooltip content={<CustomTooltip />} />
            {widget.y?.map((field, i) => (
              <Area
                key={field}
                type="monotone"
                dataKey={field}
                stroke={colors[i % colors.length]}
                fill={colors[i % colors.length]}
                fillOpacity={0.06}
                strokeWidth={1.5}
              />
            ))}
          </AreaChart>
        </ResponsiveContainer>
      );

    case "pie": {
      const legendStyle = { fontSize: 11, fontFamily: '"Geist", system-ui' };
      return (
        <ResponsiveContainer width="100%" height={240}>
          <PieChart>
            <Pie
              data={chartData}
              dataKey={widget.value || "value"}
              nameKey={widget.label || "label"}
              cx="50%"
              cy="45%"
              outerRadius={75}
              innerRadius={40}
              strokeWidth={0}
              label={({ name, percent }: { name?: string; percent?: number }) =>
                `${name ?? ""} ${((percent ?? 0) * 100).toFixed(0)}%`
              }
              labelLine={false}
              style={{ fontSize: 11, fontFamily: '"Geist", system-ui' }}
            >
              {chartData.map((_, i) => (
                <Cell key={i} fill={colors[i % colors.length]} />
              ))}
            </Pie>
            <Tooltip content={<CustomTooltip />} />
            <Legend wrapperStyle={legendStyle} iconSize={7} />
          </PieChart>
        </ResponsiveContainer>
      );
    }

    default:
      return <div className="text-[var(--dac-text-muted)] text-xs">Unknown chart: {widget.chart}</div>;
  }
}
