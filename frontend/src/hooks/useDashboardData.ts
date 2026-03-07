import { useQuery } from "@tanstack/react-query";
import { fetchDashboardData } from "../api/client";
import type { WidgetData } from "../types/dashboard";

export function useDashboardData(
  name: string,
  filters?: Record<string, unknown>,
  enabled = true,
) {
  return useQuery<Record<string, WidgetData>>({
    queryKey: ["dashboard-data", name, filters],
    queryFn: () => fetchDashboardData(name, filters),
    enabled: !!name && enabled,
  });
}
