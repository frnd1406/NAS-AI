import { useCallback, useEffect, useRef, useState } from "react";
import { apiRequest } from "../lib/api";

/**
 * Polls GET /api/v1/system/metrics/live (gopsutil inside API process).
 * Field names: cpu_percent, ram_percent, disk_percent, ram_total, disk_total.
 */
export function useLiveMetrics(intervalMs = 5000) {
  const [metrics, setMetrics] = useState(null);
  const [history, setHistory] = useState([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const mounted = useRef(true);

  const fetchLive = useCallback(async () => {
    try {
      const data = await apiRequest("/api/v1/system/metrics/live");
      if (!mounted.current) return;
      setMetrics(data);
      setError("");
      setHistory((prev) => {
        const point = {
          time: new Date().toLocaleTimeString(),
          cpu: typeof data.cpu_percent === "number" ? data.cpu_percent : 0,
          ram: typeof data.ram_percent === "number" ? data.ram_percent : 0,
        };
        const next = [...prev, point];
        return next.length > 24 ? next.slice(-24) : next;
      });
      setLoading(false);
    } catch (err) {
      if (!mounted.current) return;
      setError(err?.message || "Live-Metriken nicht verfügbar");
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    mounted.current = true;
    fetchLive();
    const id = setInterval(fetchLive, intervalMs);
    return () => {
      mounted.current = false;
      clearInterval(id);
    };
  }, [fetchLive, intervalMs]);

  return { metrics, history, error, loading, refresh: fetchLive };
}
