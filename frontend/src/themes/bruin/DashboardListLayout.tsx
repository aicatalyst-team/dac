import { Link } from "react-router-dom";
import type { DashboardListLayoutProps } from "../../types/template";

export function BruinDashboardListLayout({ dashboards }: DashboardListLayoutProps) {
  if (!dashboards.length) {
    return (
      <div className="max-w-[1400px] mx-auto px-4 sm:px-6 py-8 sm:py-10">
        <h1 className="text-xl font-semibold tracking-tight text-[var(--dac-text-primary)] mb-6">
          Dashboards
        </h1>
        <div className="border border-dashed border-[var(--dac-border)] rounded px-6 py-10 text-center">
          <p className="text-[13px] text-[var(--dac-text-muted)] mb-1">No dashboards found</p>
          <p className="text-[12px] text-[var(--dac-text-muted)]">
            Create a <code className="font-mono text-[var(--dac-text-secondary)] bg-[var(--dac-surface)] px-1.5 py-0.5 rounded text-[11px]">.yml</code> file to get started.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-[1400px] mx-auto px-4 sm:px-6 py-8 sm:py-10">
      <h1 className="text-xl font-semibold tracking-tight text-[var(--dac-text-primary)] mb-6 animate-in">
        Dashboards
      </h1>
      <div className="space-y-px">
        {dashboards.map((d, i) => (
          <Link
            key={d.name}
            to={`/d/${encodeURIComponent(d.name)}`}
            className="group flex items-center justify-between px-3 sm:px-4 py-3 -mx-3 sm:-mx-4 rounded hover:bg-[var(--dac-surface)] transition-colors duration-100 animate-in"
            style={{ animationDelay: `${i * 40}ms` }}
          >
            <div className="min-w-0">
              <div className="text-[14px] font-medium text-[var(--dac-text-primary)] group-hover:text-[var(--dac-accent)] transition-colors duration-100">
                {d.name}
              </div>
              {d.description && (
                <div className="text-[12px] text-[var(--dac-text-muted)] mt-0.5 truncate">
                  {d.description}
                </div>
              )}
            </div>
            <svg
              width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor"
              strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"
              className="text-[var(--dac-text-muted)] opacity-0 -translate-x-1 group-hover:opacity-100 group-hover:translate-x-0 transition-all duration-150 shrink-0 ml-4 hidden sm:block"
            >
              <path d="M6 4L10 8L6 12" />
            </svg>
          </Link>
        ))}
      </div>
    </div>
  );
}
