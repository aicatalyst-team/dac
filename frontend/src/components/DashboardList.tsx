import { useDashboardList } from "../hooks/useDashboard";
import { useTemplate } from "../themes/TemplateProvider";

export function DashboardList() {
  const { data: dashboards, isLoading, error } = useDashboardList();
  const { DashboardListLayout } = useTemplate();

  if (isLoading) {
    return (
      <div className="max-w-[1400px] mx-auto px-4 sm:px-6 py-8 sm:py-10">
        <div className="skeleton h-7 w-40 mb-8" />
        <div className="space-y-2">
          <div className="skeleton h-14 w-full" />
          <div className="skeleton h-14 w-full" />
          <div className="skeleton h-14 w-3/4" />
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="max-w-[1400px] mx-auto px-4 sm:px-6 py-8 sm:py-10">
        <div className="text-[13px] font-mono text-[var(--dac-error)]">{error.message}</div>
      </div>
    );
  }

  return <DashboardListLayout dashboards={dashboards ?? []} />;
}
