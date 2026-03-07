import { useState } from "react";
import type { FilterBarProps } from "../../types/template";
import type { Filter } from "../../types/dashboard";

export function BruinFilterBar({ filters, values, onChange }: FilterBarProps) {
  if (!filters.length) return null;

  return (
    <div className="flex items-end flex-wrap gap-3 mb-5 pb-4 border-b border-[var(--dac-border)]">
      {filters.map((filter) => (
        <FilterControl
          key={filter.name}
          filter={filter}
          value={values[filter.name]}
          onChange={(v) => onChange(filter.name, v)}
        />
      ))}
    </div>
  );
}

const inputClass =
  "h-7 px-2 rounded-sm text-[13px] border border-[var(--dac-border)] bg-[var(--dac-background)] text-[var(--dac-text-primary)] focus:outline-none focus:border-[var(--dac-accent)] transition-colors duration-100";

// --- Date range presets ---

interface DatePreset {
  key: string;
  label: string;
  resolve: () => { start: string; end: string };
}

function fmt(d: Date): string {
  return d.toISOString().slice(0, 10);
}

const ALL_PRESETS: DatePreset[] = [
  {
    key: "today",
    label: "Today",
    resolve: () => {
      const d = fmt(new Date());
      return { start: d, end: d };
    },
  },
  {
    key: "yesterday",
    label: "Yesterday",
    resolve: () => {
      const d = new Date();
      d.setDate(d.getDate() - 1);
      const s = fmt(d);
      return { start: s, end: s };
    },
  },
  {
    key: "last_7_days",
    label: "Last 7 days",
    resolve: () => {
      const end = new Date();
      const start = new Date();
      start.setDate(start.getDate() - 6);
      return { start: fmt(start), end: fmt(end) };
    },
  },
  {
    key: "last_30_days",
    label: "Last 30 days",
    resolve: () => {
      const end = new Date();
      const start = new Date();
      start.setDate(start.getDate() - 29);
      return { start: fmt(start), end: fmt(end) };
    },
  },
  {
    key: "last_90_days",
    label: "Last 90 days",
    resolve: () => {
      const end = new Date();
      const start = new Date();
      start.setDate(start.getDate() - 89);
      return { start: fmt(start), end: fmt(end) };
    },
  },
  {
    key: "this_month",
    label: "This month",
    resolve: () => {
      const now = new Date();
      const start = new Date(now.getFullYear(), now.getMonth(), 1);
      const end = new Date(now.getFullYear(), now.getMonth() + 1, 0);
      return { start: fmt(start), end: fmt(end) };
    },
  },
  {
    key: "last_month",
    label: "Last month",
    resolve: () => {
      const now = new Date();
      const start = new Date(now.getFullYear(), now.getMonth() - 1, 1);
      const end = new Date(now.getFullYear(), now.getMonth(), 0);
      return { start: fmt(start), end: fmt(end) };
    },
  },
  {
    key: "this_quarter",
    label: "This quarter",
    resolve: () => {
      const now = new Date();
      const q = Math.floor(now.getMonth() / 3);
      const start = new Date(now.getFullYear(), q * 3, 1);
      const end = new Date(now.getFullYear(), q * 3 + 3, 0);
      return { start: fmt(start), end: fmt(end) };
    },
  },
  {
    key: "this_year",
    label: "This year",
    resolve: () => {
      const y = new Date().getFullYear();
      return { start: `${y}-01-01`, end: `${y}-12-31` };
    },
  },
  {
    key: "year_to_date",
    label: "Year to date",
    resolve: () => {
      const now = new Date();
      return { start: `${now.getFullYear()}-01-01`, end: fmt(now) };
    },
  },
  {
    key: "all_time",
    label: "All time",
    resolve: () => ({ start: "1970-01-01", end: "2099-12-31" }),
  },
];

const DEFAULT_PRESET_KEYS = [
  "last_7_days", "last_30_days", "last_90_days",
  "this_month", "this_quarter", "this_year",
];

function getPresets(filter: Filter): DatePreset[] {
  const keys = filter.options?.presets;
  if (keys && keys.length > 0) {
    return keys
      .map((k) => ALL_PRESETS.find((p) => p.key === k))
      .filter((p): p is DatePreset => p != null);
  }
  return ALL_PRESETS.filter((p) => DEFAULT_PRESET_KEYS.includes(p.key));
}

/** Resolve a preset key string to {start, end}. Returns null if not a known preset. */
export function resolvePreset(key: string): { start: string; end: string } | null {
  const p = ALL_PRESETS.find((pr) => pr.key === key);
  return p ? p.resolve() : null;
}

/** Detect which preset matches the current value, if any. */
function detectPreset(value: { start: string; end: string }, presets: DatePreset[]): string | null {
  for (const p of presets) {
    const resolved = p.resolve();
    if (resolved.start === value.start && resolved.end === value.end) {
      return p.key;
    }
  }
  return null;
}

// --- Filter controls ---

function FilterControl({
  filter,
  value,
  onChange,
}: {
  filter: Filter;
  value: unknown;
  onChange: (value: unknown) => void;
}) {
  const label = filter.name.replace(/_/g, " ");

  switch (filter.type) {
    case "select":
      return (
        <div className="flex flex-col gap-1">
          <label className="text-[10px] font-medium uppercase tracking-wider text-[var(--dac-text-muted)]">
            {label}
          </label>
          <select
            className={inputClass}
            value={String(value ?? filter.default ?? "")}
            onChange={(e) => onChange(e.target.value)}
          >
            {filter.options?.values?.map((v) => (
              <option key={v} value={v}>{v}</option>
            ))}
          </select>
        </div>
      );

    case "date-range":
      return <DateRangeFilter filter={filter} value={value} onChange={onChange} label={label} />;

    case "text":
      return (
        <div className="flex flex-col gap-1">
          <label className="text-[10px] font-medium uppercase tracking-wider text-[var(--dac-text-muted)]">
            {label}
          </label>
          <input
            type="text"
            className={inputClass}
            value={String(value ?? filter.default ?? "")}
            onChange={(e) => onChange(e.target.value)}
            placeholder={`Filter by ${label}`}
          />
        </div>
      );

    default:
      return null;
  }
}

function DateRangeFilter({
  filter,
  value,
  onChange,
  label,
}: {
  filter: Filter;
  value: unknown;
  onChange: (value: unknown) => void;
  label: string;
}) {
  const presets = getPresets(filter);
  const dateValue = value as { start: string; end: string } | undefined;

  const activePreset = dateValue ? detectPreset(dateValue, presets) : null;
  const [showCustom, setShowCustom] = useState(
    // Show custom inputs if the current value doesn't match any preset
    dateValue != null && activePreset === null,
  );

  const handlePresetChange = (key: string) => {
    if (key === "__custom__") {
      setShowCustom(true);
      return;
    }
    setShowCustom(false);
    const preset = ALL_PRESETS.find((p) => p.key === key);
    if (preset) {
      onChange(preset.resolve());
    }
  };

  const handleDateChange = (field: "start" | "end", val: string) => {
    onChange({ ...dateValue, [field]: val });
  };

  const selectValue = showCustom ? "__custom__" : (activePreset ?? "__custom__");

  return (
    <div className="flex flex-col gap-1">
      <label className="text-[10px] font-medium uppercase tracking-wider text-[var(--dac-text-muted)]">
        {label}
      </label>
      <div className="flex items-center gap-1.5 flex-wrap">
        <select
          className={inputClass}
          value={selectValue}
          onChange={(e) => handlePresetChange(e.target.value)}
        >
          {presets.map((p) => (
            <option key={p.key} value={p.key}>{p.label}</option>
          ))}
          <option value="__custom__">Custom range</option>
        </select>

        {showCustom && (
          <>
            <input
              type="date"
              className={inputClass}
              value={dateValue?.start ?? ""}
              onChange={(e) => handleDateChange("start", e.target.value)}
            />
            <span className="text-[10px] text-[var(--dac-text-muted)]">to</span>
            <input
              type="date"
              className={inputClass}
              value={dateValue?.end ?? ""}
              onChange={(e) => handleDateChange("end", e.target.value)}
            />
          </>
        )}

        {!showCustom && dateValue && (
          <span className="text-[11px] text-[var(--dac-text-secondary)]">
            {dateValue.start} — {dateValue.end}
          </span>
        )}
      </div>
    </div>
  );
}
