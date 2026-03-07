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

// --- Admin API ---

const ADMIN_PASSWORD_KEY = "dac_admin_password";

export function getAdminPassword(): string | null {
  return sessionStorage.getItem(ADMIN_PASSWORD_KEY);
}

export function setAdminPassword(password: string): void {
  sessionStorage.setItem(ADMIN_PASSWORD_KEY, password);
}

export function clearAdminPassword(): void {
  sessionStorage.removeItem(ADMIN_PASSWORD_KEY);
}

function adminHeaders(): Record<string, string> {
  const password = getAdminPassword();
  if (!password) {
    throw new Error("Not authenticated");
  }
  return { Authorization: `Bearer ${password}` };
}

export async function adminLogin(password: string): Promise<void> {
  const res = await fetch(`${BASE}/admin/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ password }),
  });
  if (!res.ok) {
    throw new Error("Invalid password");
  }
  setAdminPassword(password);
}

export interface AdminConnections {
  connections: Record<string, Array<{ name: string; [key: string]: unknown }>>;
}

export async function listConnections(): Promise<AdminConnections> {
  return fetchJSON<AdminConnections>(`${BASE}/admin/connections`, {
    headers: adminHeaders(),
  });
}

export async function createConnection(
  type: string,
  name: string,
  fields: Record<string, unknown>,
): Promise<void> {
  await fetchJSON(`${BASE}/admin/connections`, {
    method: "POST",
    headers: { "Content-Type": "application/json", ...adminHeaders() },
    body: JSON.stringify({ type, name, fields }),
  });
}

export async function updateConnection(
  type: string,
  name: string,
  fields: Record<string, unknown>,
): Promise<void> {
  await fetchJSON(
    `${BASE}/admin/connections/${encodeURIComponent(type)}/${encodeURIComponent(name)}`,
    {
      method: "PUT",
      headers: { "Content-Type": "application/json", ...adminHeaders() },
      body: JSON.stringify({ fields }),
    },
  );
}

export async function deleteConnection(type: string, name: string): Promise<void> {
  await fetchJSON(
    `${BASE}/admin/connections/${encodeURIComponent(type)}/${encodeURIComponent(name)}`,
    {
      method: "DELETE",
      headers: adminHeaders(),
    },
  );
}

export async function testConnection(
  type: string,
  name: string,
): Promise<{ ok: boolean }> {
  return fetchJSON<{ ok: boolean }>(
    `${BASE}/admin/connections/${encodeURIComponent(type)}/${encodeURIComponent(name)}/test`,
    {
      method: "POST",
      headers: adminHeaders(),
    },
  );
}
