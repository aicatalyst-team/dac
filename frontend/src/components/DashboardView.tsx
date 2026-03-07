import { useState, useEffect } from "react";
import { useParams, Link } from "react-router-dom";
import { useDashboard } from "../hooks/useDashboard";
import { useDashboardData } from "../hooks/useDashboardData";
import { Row, WidgetContainer } from "./Row";
import { Widget } from "./Widget";
import { FilterBar } from "./FilterBar";

function buildDefaultFilters(dashboard: { filters?: { name: string; default?: unknown }[] }): Record<string, unknown> {
  const defaults: Record<string, unknown> = {};
  if (dashboard.filters) {
    for (const f of dashboard.filters) {
      if (f.default !== undefined) {
        defaults[f.name] = f.default;
      }
    }
  }
  return defaults;
}

export function DashboardView() {
  const { name } = useParams<{ name: string }>();
  const { data: dashboard, isLoading: dashLoading, error: dashError } = useDashboard(name || "");
  const [filters, setFilters] = useState<Record<string, unknown> | null>(null);
  const [defaultsApplied, setDefaultsApplied] = useState(false);

  useEffect(() => {
    if (dashboard && !defaultsApplied) {
      setFilters(buildDefaultFilters(dashboard));
      setDefaultsApplied(true);
    }
  }, [dashboard, defaultsApplied]);

  const { data: widgetData, isLoading: dataLoading } = useDashboardData(
    name || "",
    filters ?? undefined,
    !!filters,
  );

  if (dashLoading) {
    return (
      <div className="max-w-[1400px] mx-auto px-4 sm:px-6 py-6 sm:py-8">
        <div className="skeleton h-3 w-16 mb-6" />
        <div className="skeleton h-6 w-48 mb-1.5" />
        <div className="skeleton h-3 w-72" />
      </div>
    );
  }

  if (dashError || !dashboard) {
    return (
      <div className="max-w-[1400px] mx-auto px-4 sm:px-6 py-6 sm:py-8">
        <div className="text-[13px] font-mono text-[var(--dac-error)]">
          {dashError?.message || "Dashboard not found"}
        </div>
      </div>
    );
  }

  const handleFilterChange = (filterName: string, value: unknown) => {
    setFilters((prev) => ({ ...prev, [filterName]: value }));
  };

  return (
    <div className="max-w-[1400px] mx-auto px-4 sm:px-6 py-6 sm:py-8">
      <header className="mb-6 animate-in">
        <Link
          to="/"
          className="inline-flex items-center gap-1 text-[12px] text-[var(--dac-text-muted)] hover:text-[var(--dac-text-secondary)] transition-colors duration-100 mb-3"
        >
          <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
            <path d="M10 12L6 8L10 4" />
          </svg>
          Dashboards
        </Link>
        <h1 className="text-xl font-semibold tracking-tight text-[var(--dac-text-primary)]">
          {dashboard.name}
        </h1>
        {dashboard.description && (
          <p className="text-[13px] text-[var(--dac-text-secondary)] mt-1 max-w-lg">
            {dashboard.description}
          </p>
        )}
      </header>

      {dashboard.filters && (
        <div className="animate-in" style={{ animationDelay: "30ms" }}>
          <FilterBar
            filters={dashboard.filters}
            values={filters ?? {}}
            onChange={handleFilterChange}
          />
        </div>
      )}

      <div className="flex flex-col gap-4">
        {dashboard.rows.map((row, rowIdx) => (
          <div
            key={rowIdx}
            className="animate-in"
            style={{ animationDelay: `${50 + rowIdx * 30}ms` }}
          >
            <Row>
              {row.widgets.map((widget, widgetIdx) => {
                const id = `r${rowIdx}-w${widgetIdx}`;
                const col = widget.col || Math.floor(12 / row.widgets.length);
                return (
                  <WidgetContainer key={id} col={col}>
                    <Widget
                      widget={widget}
                      data={widgetData?.[id]}
                      isLoading={dataLoading}
                    />
                  </WidgetContainer>
                );
              })}
            </Row>
          </div>
        ))}
      </div>
    </div>
  );
}
