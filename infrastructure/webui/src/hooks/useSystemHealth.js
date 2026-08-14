import { useQuery } from "@tanstack/react-query";
import { authHeaders } from "../utils/auth";

const API_BASE = import.meta.env.VITE_API_BASE_URL || window.location.origin;

async function fetchJSON(url, options = {}) {
  const res = await fetch(url, {
    credentials: "include",
    ...options,
  });
  if (!res.ok) {
    throw new Error(`HTTP ${res.status}`);
  }
  return res.json();
}

export function useSystemHealth() {
  return useQuery({
    queryKey: ["system-health"],
    queryFn: async () => {
      const headers = authHeaders();
      const controller = new AbortController();
      const signal = controller.signal;

      // Settings + Backups
      let settings = null;
      let lastBackup = null;
      let snapshotCount = 0;

      try {
        const settingsData = await fetchJSON(`${API_BASE}/api/v1/system/settings`, {
          headers,
          signal,
        });
        settings = settingsData;
      } catch (err) {
        // keep settings null on error
      }

      try {
        const backupsData = await fetchJSON(`${API_BASE}/api/v1/backups`, {
          headers,
          signal,
        });
        const backups = backupsData.items || [];
        snapshotCount = backups.length;
        if (backups.length > 0) {
          const sorted = backups.sort(
            (a, b) =>
              new Date(b.modTime || b.created_at) - new Date(a.modTime || a.created_at)
          );
          lastBackup = sorted[0];
        }
      } catch (err) {
        // keep backups null on error
      }

      // Metrics
      let latestMetric = null;
      let metricsError = "";
      try {
        const metricData = await fetchJSON(`${API_BASE}/api/v1/system/metrics?limit=1`, {
          signal,
        });
        const items = metricData.items || [];
        latestMetric = items[0] || null;
      } catch (err) {
        metricsError = err.message || "Metrics nicht verfügbar";
        latestMetric = null;
      }

      return {
        settings,
        lastBackup,
        snapshotCount,
        latestMetric,
        metricsError,
      };
    },
    refetchInterval: 30000, // 30s background refresh
  });
}
