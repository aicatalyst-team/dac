import type { DashboardSummary, Dashboard, BatchDataResponse, WidgetData } from "../types/dashboard";
import type { Theme } from "../types/theme";

const BASE = "/api/v1";

async function fetchJSON<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, init);
  if (!res.ok) {
    const body = await res.text();
    throw new Error(`API error ${res.status}: ${body}`);
  }
  return res.json();
}

export interface ServerConfig {
  template: string;
  tokens?: Record<string, string>;
}

export async function fetchConfig(): Promise<ServerConfig> {
  return fetchJSON<ServerConfig>(`${BASE}/config`);
}

export async function listDashboards(): Promise<DashboardSummary[]> {
  const data = await fetchJSON<{ dashboards: DashboardSummary[] }>(`${BASE}/dashboards`);
  return data.dashboards;
}

export async function getDashboard(name: string): Promise<Dashboard> {
  return fetchJSON<Dashboard>(`${BASE}/dashboards/${encodeURIComponent(name)}`);
}

export async function fetchDashboardData(
  name: string,
  filters?: Record<string, unknown>,
): Promise<Record<string, WidgetData>> {
  const data = await fetchJSON<BatchDataResponse>(
    `${BASE}/dashboards/${encodeURIComponent(name)}/data`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ filters: filters ?? {} }),
    },
  );
  return data.widgets;
}

export async function listThemes(): Promise<string[]> {
  const data = await fetchJSON<{ themes: string[] }>(`${BASE}/themes`);
  return data.themes;
}

export async function getTheme(name: string): Promise<Theme> {
  return fetchJSON<Theme>(`${BASE}/themes/${encodeURIComponent(name)}`);
}
