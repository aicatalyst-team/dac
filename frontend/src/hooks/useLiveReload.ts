import { useEffect, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";

/**
 * Listens for SSE `full_reload` events, invalidates React Query caches, and
 * returns a monotonically-increasing counter that bumps on every reload.
 * Use it as a React `key` on components that need a full remount (e.g.
 * DashboardView) so all local state resets to match the new definition.
 */
export function useLiveReload(): number {
  const queryClient = useQueryClient();
  const [reloadKey, setReloadKey] = useState(0);

  useEffect(() => {
    // No server to connect to in static mode.
    if ((window as any).__DAC_STATIC__) return;

    const es = new EventSource("/api/v1/events");

    es.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        if (data.type === "full_reload") {
          queryClient.invalidateQueries();
          setReloadKey((k) => k + 1);
        }
      } catch {
        // ignore parse errors
      }
    };

    es.onerror = () => {
      // EventSource will automatically reconnect
    };

    return () => es.close();
  }, [queryClient]);

  return reloadKey;
}
