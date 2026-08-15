import { useQuery } from "@tanstack/react-query";

const API_BASE = import.meta.env.VITE_API_BASE_URL || window.location.origin;

async function fetchUPSStatus() {
  const res = await fetch(`${API_BASE}/api/v1/system/hardware/ups`, {
    credentials: "include",
  });
  if (!res.ok) {
    throw new Error(`HTTP ${res.status}`);
  }
  return res.json();
}

// useUPSStatus polls the public UPS endpoint. The backend never 500s here: an
// unreachable UPS returns { available: false }, so consumers just check `state`.
export function useUPSStatus() {
  return useQuery({
    queryKey: ["ups-status"],
    queryFn: fetchUPSStatus,
    refetchInterval: 10000, // 10s — fast enough to catch a power cut promptly
    staleTime: 5000,
    retry: 1,
  });
}
