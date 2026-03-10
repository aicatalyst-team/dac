import { useEffect, useState, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { useDashboardList } from "../hooks/useDashboard";
import { useTemplate } from "../themes/TemplateProvider";
import { fetchConfig } from "../api/client";
import { AgentChat } from "./AgentChat";

const isStaticMode = !!(window as any).__DAC_STATIC__;

export function DashboardList() {
  const { data: dashboards, isLoading, error } = useDashboardList();
  const { data: config } = useQuery({
    queryKey: ["config"],
    queryFn: fetchConfig,
    staleTime: Infinity,
  });
  const { DashboardListLayout } = useTemplate();
  const navigate = useNavigate();
  const [createOpen, setCreateOpen] = useState(false);
  const [agentWidth, setAgentWidth] = useState(420);
  const [isResizing, setIsResizing] = useState(false);
  const handleResize = useCallback((delta: number) => {
    setAgentWidth((w) => Math.max(320, Math.min(600, w + delta)));
  }, []);
  const onResizeStart = useCallback(() => setIsResizing(true), []);
  const onResizeEnd = useCallback(() => setIsResizing(false), []);

  const handleDashboardCreated = useCallback((name: string) => {
    // Transfer the agent chat session from "__create__" to the new dashboard name
    // so the conversation continues on the dashboard page.
    const KEY_PREFIX = "dac-agent-";
    try {
      const data = localStorage.getItem(KEY_PREFIX + "__create__");
      if (data) {
        localStorage.setItem(KEY_PREFIX + name, data);
        localStorage.removeItem(KEY_PREFIX + "__create__");
      }
    } catch { /* ignore */ }
    setCreateOpen(false);
    navigate(`/d/${encodeURIComponent(name)}`, { state: { agentOpen: true } });
  }, [navigate]);

  // Auto-redirect to the dashboard if there's only one.
  useEffect(() => {
    if (dashboards && dashboards.length === 1 && !createOpen) {
      navigate(`/d/${encodeURIComponent(dashboards[0].name)}`, { replace: true });
    }
  }, [dashboards, navigate, createOpen]);

  if (isLoading) {
    return (
      <div className="max-w-[860px] mx-auto px-4 sm:px-6 pt-16 sm:pt-24 pb-8">
        <div className="skeleton h-7 w-40 mb-8" />
        <div className="skeleton h-8 w-full mb-4 rounded" />
        <div className="space-y-2">
          <div className="skeleton h-12 w-full" />
          <div className="skeleton h-12 w-full" />
          <div className="skeleton h-12 w-3/4" />
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="max-w-[860px] mx-auto px-4 sm:px-6 pt-16 sm:pt-24 pb-8">
        <div className="text-[13px] font-mono text-[var(--dac-error)]">{error.message}</div>
      </div>
    );
  }

  // If there's only one dashboard, we'll redirect — don't flash the list.
  if (dashboards && dashboards.length === 1 && !createOpen) {
    return null;
  }

  if (isStaticMode) {
    return (
      <DashboardListLayout
        dashboards={dashboards ?? []}
        adminEnabled={config?.admin_enabled}
      />
    );
  }

  return (
    <div
      className={`h-screen overflow-hidden ${isResizing ? "select-none" : ""}`}
      style={{
        display: "grid",
        gridTemplateColumns: createOpen ? `${agentWidth}px 1fr` : "1fr",
      }}
    >
      {createOpen && (
        <AgentChat
          dashboardName="__create__"
          isOpen={true}
          onClose={() => setCreateOpen(false)}
          onResize={handleResize}
          onResizeStart={onResizeStart}
          onResizeEnd={onResizeEnd}
          onDashboardCreated={handleDashboardCreated}
        />
      )}
      <div className="overflow-y-auto">
        <DashboardListLayout
          dashboards={dashboards ?? []}
          adminEnabled={config?.admin_enabled}
          onCreateClick={() => setCreateOpen(true)}
        />
      </div>
    </div>
  );
}
