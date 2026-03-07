import { useState, useMemo } from "react";
import { useParams } from "react-router-dom";
import { useDashboard } from "../hooks/useDashboard";
import { useDashboardData } from "../hooks/useDashboardData";
import { useTemplate } from "../themes/TemplateProvider";
import { resolvePreset } from "../themes/bruin/FilterBar";
import { AgentChat } from "./AgentChat";
import type { Filter } from "../types/dashboard";

function buildDefaultFilters(dashboard: { filters?: Filter[] }): Record<string, unknown> {
  const defaults: Record<string, unknown> = {};
  if (dashboard.filters) {
    for (const f of dashboard.filters) {
      if (f.default !== undefined) {
        if (f.type === "date-range" && typeof f.default === "string") {
          const resolved = resolvePreset(f.default);
          defaults[f.name] = resolved ?? f.default;
        } else {
          defaults[f.name] = f.default;
        }
      }
    }
  }
  return defaults;
}

export function DashboardView() {
  const { name } = useParams<{ name: string }>();
  const { data: dashboard, isLoading: dashLoading, error: dashError } = useDashboard(name || "");
  const [agentOpen, setAgentOpen] = useState(false);

  const defaultFilters = useMemo(
    () => dashboard ? buildDefaultFilters(dashboard) : null,
    [dashboard],
  );

  const [filters, setFilters] = useState<Record<string, unknown> | null>(null);
  const activeFilters = filters ?? defaultFilters;

  const {
    DashboardLayout,
    WidgetFrame,
    FilterBar,
    Row,
    WidgetContainer,
  } = useTemplate();

  const { data: widgetData, isLoading: dataLoading } = useDashboardData(
    name || "",
    activeFilters ?? undefined,
    !!activeFilters,
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

  const filterBar = dashboard.filters ? (
    <FilterBar
      filters={dashboard.filters}
      values={activeFilters ?? {}}
      onChange={handleFilterChange}
    />
  ) : null;

  const agentButton = (
    <button
      onClick={() => setAgentOpen(!agentOpen)}
      className={`inline-flex items-center gap-1.5 h-7 px-2 rounded-sm border text-[13px] transition-colors duration-100 ${
        agentOpen
          ? "border-[var(--dac-accent)] text-[var(--dac-accent)]"
          : "border-[var(--dac-border)] bg-[var(--dac-background)] text-[var(--dac-text-secondary)] hover:text-[var(--dac-text-primary)] hover:border-[var(--dac-text-muted)]"
      }`}
      title="Edit with AI"
    >
      {agentOpen ? (
        <svg width="13" height="13" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
          <path d="M4 4L12 12M12 4L4 12" />
        </svg>
      ) : (
        <svg width="13" height="13" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
          <path d="M11.5 1.5L14.5 4.5L5 14H2V11L11.5 1.5Z" />
        </svg>
      )}
      {agentOpen ? "Close" : "Edit"}
    </button>
  );

  return (
    <div className="flex h-screen overflow-hidden">
      <AgentChat
        dashboardName={name || ""}
        isOpen={agentOpen}
        onClose={() => setAgentOpen(false)}
      />
      <div className="flex-1 overflow-y-auto">
        <DashboardLayout dashboard={dashboard} filterBar={filterBar} headerActions={agentButton}>
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
                      <WidgetFrame
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
        </DashboardLayout>
      </div>
    </div>
  );
}
