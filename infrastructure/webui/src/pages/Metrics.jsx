import { useEffect, useMemo, useState } from "react";
import { authHeaders } from "../utils/auth";
import { AlertTriangle, CheckCircle, AlertCircle, Activity, Loader2, Cpu, HardDrive, Gauge } from "lucide-react";

const API_BASE =
  import.meta.env.VITE_API_BASE_URL ||
  window.location.origin;
const POLL_MS = 5000;

// Glass Card Component
const GlassCard = ({ children, className = "" }) => (
  <div className={`relative overflow-hidden rounded-2xl border border-white/10 bg-slate-900/40 backdrop-blur-xl shadow-2xl ${className}`}>
    {/* Internal "Shimmer" Reflection */}
    <div className="absolute top-0 left-0 w-full h-[1px] bg-gradient-to-r from-transparent via-white/20 to-transparent opacity-50"></div>
    <div className="p-6 h-full flex flex-col">
      {children}
    </div>
  </div>
);

export default function Metrics() {
  const [alerts, setAlerts] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [metrics, setMetrics] = useState([]);
  const [metricsError, setMetricsError] = useState("");

  const banner = useMemo(() => {
    const hasCritical = alerts.some((a) => a.severity === "CRITICAL");
    const hasWarning = alerts.some((a) => a.severity === "WARNING");

    if (hasCritical) {
      return {
        icon: AlertCircle,
        color: "rose",
        borderClass: "border-rose-500/30",
        bgClass: "bg-rose-500/10",
        textClass: "text-rose-400",
        text: "Critical Alerts Active",
      };
    }
    if (hasWarning) {
      return {
        icon: AlertTriangle,
        color: "amber",
        borderClass: "border-amber-500/30",
        bgClass: "bg-amber-500/10",
        textClass: "text-amber-400",
        text: "Warnings Active",
      };
    }
    return {
      icon: CheckCircle,
      color: "emerald",
      borderClass: "border-emerald-500/30",
      bgClass: "bg-emerald-500/10",
      textClass: "text-emerald-400",
      text: "System Healthy",
    };
  }, [alerts]);

  const fetchAlerts = async () => {
    setLoading(true);
    setError("");
    try {
      const res = await fetch(`${API_BASE}/api/v1/system/alerts`, {
        credentials: "include",
        headers: authHeaders(),
      });
      if (!res.ok) {
        throw new Error(`HTTP ${res.status}`);
      }
      const data = await res.json();
      setAlerts(data.items || []);
    } catch (err) {
      setError(err.message || "Failed to load alerts");
    } finally {
      setLoading(false);
    }
  };

  const fetchMetrics = async () => {
    try {
      const res = await fetch(`${API_BASE}/api/v1/system/metrics?limit=5`, {
        credentials: "include",
        headers: authHeaders(),
      });
      if (!res.ok) {
        throw new Error(`HTTP ${res.status}`);
      }
      const data = await res.json();
      setMetrics(data.items || []);
      setMetricsError("");
    } catch (err) {
      setMetricsError(err.message || "Failed to load metrics");
    }
  };

  const fetchAll = async () => {
    await Promise.all([fetchAlerts(), fetchMetrics()]);
  };

  useEffect(() => {
    fetchAll();
    const id = setInterval(fetchAll, POLL_MS);
    return () => clearInterval(id);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const getSeverityStyle = (severity) => {
    switch (severity) {
      case "CRITICAL":
        return {
          bgClass: "bg-rose-500/10",
          textClass: "text-rose-400",
          borderClass: "border-rose-500/20",
          icon: AlertCircle,
        };
      case "WARNING":
        return {
          bgClass: "bg-amber-500/10",
          textClass: "text-amber-400",
          borderClass: "border-amber-500/20",
          icon: AlertTriangle,
        };
      default:
        return {
          bgClass: "bg-blue-500/10",
          textClass: "text-blue-400",
          borderClass: "border-blue-500/20",
          icon: Activity,
        };
    }
  };

  const BannerIcon = banner.icon;
  const latest = metrics[0];
  const statBlocks = latest
    ? [
        { label: "CPU Usage", value: latest.cpu_usage, icon: Cpu, color: "text-blue-400", bar: "bg-blue-500" },
        { label: "RAM Usage", value: latest.ram_usage, icon: Gauge, color: "text-emerald-400", bar: "bg-emerald-500" },
        { label: "Disk Usage", value: latest.disk_usage, icon: HardDrive, color: "text-amber-400", bar: "bg-amber-500" },
      ]
    : [];

  return (
    <div className="space-y-6">

      {/* Status Banner */}
      <GlassCard className={`${banner.bgClass} border ${banner.borderClass}`}>
        <div className="flex items-center gap-4">
          <div className={`p-3 rounded-xl ${banner.bgClass}`}>
            <BannerIcon className={banner.textClass} size={28} strokeWidth={2} />
          </div>
          <div className="flex-1">
            <h2 className={`text-xl font-bold ${banner.textClass} tracking-tight`}>
              {banner.text}
            </h2>
            <p className="text-slate-400 text-sm mt-1">
              {alerts.length === 0
                ? "All systems operational"
                : `${alerts.length} alert${alerts.length > 1 ? 's' : ''} detected`}
            </p>
          </div>
          {loading && (
            <div className="flex items-center gap-2 text-slate-400">
              <Loader2 size={16} className="animate-spin" />
              <span className="text-xs">Refreshing...</span>
            </div>
          )}
        </div>

        {error && (
          <div className="mt-4 p-3 rounded-lg bg-rose-500/10 border border-rose-500/20">
            <p className="text-rose-400 text-sm font-medium">Error: {error}</p>
          </div>
        )}
      </GlassCard>

      {/* Live Metrics */}
      <GlassCard>
        <div className="flex items-center justify-between mb-6">
          <div>
            <h3 className="text-white font-semibold text-lg tracking-tight flex items-center gap-2">
              <Gauge size={20} className="text-blue-400" />
              Live System Metrics
            </h3>
            <p className="text-slate-400 text-xs mt-1">CPU / RAM / Disk (latest sample)</p>
          </div>
          {metricsError && (
            <span className="text-rose-400 text-xs font-medium">Error: {metricsError}</span>
          )}
        </div>

        {latest ? (
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
            {statBlocks.map((stat) => {
              const Icon = stat.icon;
              const pct = Math.min(100, Math.max(0, Math.round(stat.value)));
              return (
                <div key={stat.label} className="p-4 rounded-xl bg-white/5 border border-white/10 shadow-lg">
                  <div className="flex items-center justify-between mb-3">
                    <div className="flex items-center gap-2">
                      <Icon size={18} className={stat.color} />
                      <span className="text-sm font-semibold text-white">{stat.label}</span>
                    </div>
                    <span className="text-sm font-semibold text-white">{pct}%</span>
                  </div>
                  <div className="h-2 rounded-full bg-white/5 overflow-hidden">
                    <div
                      className={`${stat.bar} h-full transition-all duration-500`}
                      style={{ width: `${pct}%` }}
                    />
                  </div>
                  <p className="text-[11px] text-slate-500 mt-2">Sampled at {new Date(latest.created_at || latest.createdAt).toLocaleTimeString()}</p>
                </div>
              );
            })}
          </div>
        ) : (
          <div className="flex flex-col items-center justify-center py-8 text-slate-400 text-sm">
            {metricsError ? metricsError : "No metrics received yet. Waiting for agents..."}
          </div>
        )}
      </GlassCard>

      {/* Active Alerts */}
      <GlassCard>
        <div className="flex items-center justify-between mb-6">
          <div>
            <h3 className="text-white font-semibold text-lg tracking-tight flex items-center gap-2">
              <Activity size={20} className="text-blue-400" />
              Active Alerts
            </h3>
            <p className="text-slate-400 text-xs mt-1">Real-time system notifications</p>
          </div>
          <div className={`px-3 py-1.5 rounded-full text-xs font-medium border ${banner.borderClass} ${banner.bgClass} ${banner.textClass}`}>
            {alerts.length} Total
          </div>
        </div>

        {loading && alerts.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-12">
            <Loader2 size={32} className="text-blue-400 animate-spin mb-3" />
            <p className="text-slate-400 text-sm">Loading alerts...</p>
          </div>
        ) : (
          <div className="space-y-3">
            {alerts.map((alert) => {
              const style = getSeverityStyle(alert.severity);
              const AlertIcon = style.icon;

              return (
                <div
                  key={alert.id}
                  className={`p-4 rounded-xl border ${style.borderClass} ${style.bgClass} transition-all hover:scale-[1.01]`}
                >
                  <div className="flex items-start gap-3">
                    <div className={`p-2 rounded-lg ${style.bgClass}`}>
                      <AlertIcon className={style.textClass} size={20} />
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 mb-1">
                        <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium border ${style.borderClass} ${style.textClass}`}>
                          {alert.severity}
                        </span>
                        <span className="text-xs text-slate-500">
                          {new Date(alert.created_at || alert.createdAt).toLocaleString()}
                        </span>
                      </div>
                      <p className="text-white text-sm font-medium leading-relaxed">
                        {alert.message}
                      </p>
                    </div>
                  </div>
                </div>
              );
            })}

            {alerts.length === 0 && (
              <div className="flex flex-col items-center justify-center py-16">
                <div className="p-4 rounded-full bg-emerald-500/10 mb-4">
                  <CheckCircle className="text-emerald-400" size={48} strokeWidth={1.5} />
                </div>
                <h4 className="text-white font-semibold text-lg mb-2">All Clear</h4>
                <p className="text-slate-400 text-sm text-center max-w-md">
                  No active alerts at this time. Your system is running smoothly.
                </p>
              </div>
            )}
          </div>
        )}
      </GlassCard>

      {/* Alert Statistics */}
      {alerts.length > 0 && (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <GlassCard className="bg-rose-500/5 hover:bg-rose-500/10 transition-colors">
            <div className="flex items-center gap-3">
              <div className="p-3 rounded-xl bg-rose-500/20">
                <AlertCircle className="text-rose-400" size={24} />
              </div>
              <div>
                <p className="text-slate-400 text-xs uppercase tracking-wider">Critical</p>
                <p className="text-2xl font-bold text-white mt-1">
                  {alerts.filter(a => a.severity === 'CRITICAL').length}
                </p>
              </div>
            </div>
          </GlassCard>

          <GlassCard className="bg-amber-500/5 hover:bg-amber-500/10 transition-colors">
            <div className="flex items-center gap-3">
              <div className="p-3 rounded-xl bg-amber-500/20">
                <AlertTriangle className="text-amber-400" size={24} />
              </div>
              <div>
                <p className="text-slate-400 text-xs uppercase tracking-wider">Warning</p>
                <p className="text-2xl font-bold text-white mt-1">
                  {alerts.filter(a => a.severity === 'WARNING').length}
                </p>
              </div>
            </div>
          </GlassCard>

          <GlassCard className="bg-blue-500/5 hover:bg-blue-500/10 transition-colors">
            <div className="flex items-center gap-3">
              <div className="p-3 rounded-xl bg-blue-500/20">
                <Activity className="text-blue-400" size={24} />
              </div>
              <div>
                <p className="text-slate-400 text-xs uppercase tracking-wider">Info</p>
                <p className="text-2xl font-bold text-white mt-1">
                  {alerts.filter(a => a.severity !== 'CRITICAL' && a.severity !== 'WARNING').length}
                </p>
              </div>
            </div>
          </GlassCard>
        </div>
      )}
    </div>
  );
}
