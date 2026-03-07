import { useState, useEffect, useRef, useCallback } from "react";
import { streamDashboardData } from "../api/client";
import type { WidgetData } from "../types/dashboard";

interface StreamState {
  /** Accumulated widget results, updated as each arrives. */
  data: Record<string, WidgetData> | undefined;
  /** True while the stream is open and results are still arriving. */
  isLoading: boolean;
  /** Set if the stream fails entirely. */
  error: Error | undefined;
}

export function useDashboardData(
  name: string,
  filters?: Record<string, unknown>,
  enabled = true,
): StreamState {
  const [data, setData] = useState<Record<string, WidgetData> | undefined>(undefined);
  const [isLoading, setIsLoading] = useState(!!name && enabled);
  const [error, setError] = useState<Error | undefined>(undefined);
  const abortRef = useRef<(() => void) | null>(null);

  // Stable serialized key so we can detect filter changes.
  const filterKey = filters ? JSON.stringify(filters) : "";

  const startStream = useCallback(() => {
    // Abort any in-flight stream.
    abortRef.current?.();

    setData(undefined);
    setIsLoading(true);
    setError(undefined);

    const abort = streamDashboardData(
      name,
      filters,
      (id, widgetData) => {
        setData((prev) => ({ ...prev, [id]: widgetData }));
      },
      () => {
        setIsLoading(false);
      },
      (err) => {
        setError(err);
        setIsLoading(false);
      },
    );

    abortRef.current = abort;
  }, [name, filterKey]); // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (!name || !enabled) {
      setIsLoading(false);
      return;
    }

    startStream();

    return () => {
      abortRef.current?.();
      abortRef.current = null;
    };
  }, [name, enabled, startStream]);

  return { data, isLoading, error };
}
