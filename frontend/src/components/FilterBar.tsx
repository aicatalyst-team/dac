import type { Filter } from "../types/dashboard";

interface Props {
  filters: Filter[];
  values: Record<string, unknown>;
  onChange: (name: string, value: unknown) => void;
}

export function FilterBar({ filters, values, onChange }: Props) {
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
      return (
        <div className="flex flex-col gap-1">
          <label className="text-[10px] font-medium uppercase tracking-wider text-[var(--dac-text-muted)]">
            {label}
          </label>
          <div className="flex items-center gap-1.5 flex-wrap">
            <input
              type="date"
              className={inputClass}
              value={String((value as Record<string, string>)?.start ?? "")}
              onChange={(e) =>
                onChange({ ...(value as Record<string, string>), start: e.target.value })
              }
            />
            <span className="text-[10px] text-[var(--dac-text-muted)]">to</span>
            <input
              type="date"
              className={inputClass}
              value={String((value as Record<string, string>)?.end ?? "")}
              onChange={(e) =>
                onChange({ ...(value as Record<string, string>), end: e.target.value })
              }
            />
          </div>
        </div>
      );

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
