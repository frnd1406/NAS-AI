import PropTypes from "prop-types";
import { Cpu, MemoryStick, HardDrive, Server, Shield } from "lucide-react";
import { GlassCard } from "./ui/GlassCard";
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
} from "recharts";

function formatPct(v) {
  return typeof v === "number" && Number.isFinite(v) ? `${Math.round(v)}%` : "—";
}

function formatBytes(bytes) {
  if (bytes == null || !Number.isFinite(bytes)) return null;
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.min(sizes.length - 1, Math.floor(Math.log(Math.max(bytes, 1)) / Math.log(k)));
  return `${(bytes / k ** i).toFixed(2)} ${sizes[i]}`;
}

function barColor(pct) {
  if (typeof pct !== "number") return "from-slate-500 to-slate-400";
  if (pct > 90) return "from-red-500 to-rose-400";
  if (pct > 70) return "from-amber-500 to-yellow-400";
  return "from-cyan-500 to-blue-400";
}

function MetricRow({ icon: Icon, iconClass, label, sub, value, pct, footer }) {
  const width = typeof pct === "number" ? Math.min(100, Math.max(0, pct)) : 0;
  return (
    <div className="p-4 rounded-xl bg-white/5 border border-[var(--border-color)]">
      <div className="flex items-center justify-between mb-3 gap-3">
        <div className="flex items-center gap-3 min-w-0">
          <div className={`p-2 rounded-lg ${iconClass}`}>
            <Icon size={18} />
          </div>
          <div className="min-w-0">
            <p className="text-[var(--text-primary)] text-sm font-medium truncate">{label}</p>
            <p className="text-xs text-[var(--text-secondary)] opacity-70 truncate">{sub}</p>
          </div>
        </div>
        <p className="text-[var(--text-primary)] font-mono text-xl font-bold shrink-0">{value}</p>
      </div>
      <div className="w-full bg-black/20 rounded-full h-2 overflow-hidden">
        <div
          className={`h-full bg-gradient-to-r ${barColor(pct)} transition-all duration-500 rounded-full`}
          style={{ width: `${width}%` }}
        />
      </div>
      {footer && (
        <p className="text-[10px] text-[var(--text-secondary)] mt-2 opacity-80">{footer}</p>
      )}
    </div>
  );
}

MetricRow.propTypes = {
  icon: PropTypes.elementType.isRequired,
  iconClass: PropTypes.string.isRequired,
  label: PropTypes.string.isRequired,
  sub: PropTypes.string,
  value: PropTypes.string.isRequired,
  pct: PropTypes.number,
  footer: PropTypes.string,
};

/**
 * Presentational host metrics card (NAS live or Firewall placeholder).
 */
export default function HostStatusCard({
  title,
  subtitle,
  role,
  live = false,
  placeholder = false,
  placeholderMessage,
  cpuPercent,
  ramPercent,
  diskPercent,
  ramTotal,
  diskTotal,
  diskLabel = "API-Host / Root",
  updatedAt,
  history = [],
  error,
}) {
  const RoleIcon = role === "firewall" ? Shield : Server;

  return (
    <GlassCard className="hover:bg-blue-500/5 transition-colors h-full">
      <div className="flex items-start justify-between mb-5 gap-3">
        <div className="flex items-center gap-3 min-w-0">
          <div className="p-3 rounded-xl bg-blue-500/20 border border-blue-500/30 shrink-0">
            <RoleIcon size={22} className="text-blue-400" />
          </div>
          <div className="min-w-0">
            <p className="text-[var(--text-secondary)] text-xs uppercase tracking-wider">{subtitle}</p>
            <p className="text-lg font-semibold text-[var(--text-primary)] mt-0.5 truncate">{title}</p>
          </div>
        </div>
        {live && !placeholder && (
          <div className="flex items-center gap-2 px-3 py-1.5 rounded-full bg-emerald-500/10 border border-emerald-500/20 shrink-0">
            <div className="w-2 h-2 bg-emerald-400 rounded-full animate-pulse" />
            <span className="text-emerald-400 text-xs font-medium uppercase tracking-wider">Live</span>
          </div>
        )}
        {placeholder && (
          <div className="px-3 py-1.5 rounded-full bg-slate-500/15 border border-slate-500/25 shrink-0">
            <span className="text-slate-400 text-xs font-medium uppercase tracking-wider">Bald</span>
          </div>
        )}
      </div>

      {error && (
        <div className="mb-4 p-3 rounded-xl bg-red-500/10 border border-red-500/20 text-red-400 text-sm">
          {error}
        </div>
      )}

      {placeholder ? (
        <div className="flex-1 flex flex-col items-center justify-center text-center py-10 px-4 rounded-xl border border-dashed border-[var(--border-color)] bg-white/[0.03]">
          <Shield size={28} className="text-slate-500 mb-3" />
          <p className="text-[var(--text-primary)] font-medium text-sm">Monitoring folgt</p>
          <p className="text-[var(--text-secondary)] text-xs mt-2 max-w-xs leading-relaxed">
            {placeholderMessage ||
              "Firewall-Pi (4B) wird später angebunden — Layout ist bereits vorbereitet."}
          </p>
        </div>
      ) : (
        <div className="space-y-3">
          <MetricRow
            icon={Cpu}
            iconClass="bg-cyan-500/20 text-cyan-400"
            label="CPU Auslastung"
            sub="Prozessor"
            value={formatPct(cpuPercent)}
            pct={cpuPercent}
          />
          <MetricRow
            icon={MemoryStick}
            iconClass="bg-violet-500/20 text-violet-400"
            label="Arbeitsspeicher"
            sub="RAM"
            value={formatPct(ramPercent)}
            pct={ramPercent}
            footer={formatBytes(ramTotal) ? `Gesamt: ${formatBytes(ramTotal)}` : undefined}
          />
          <MetricRow
            icon={HardDrive}
            iconClass="bg-amber-500/20 text-amber-400"
            label="Speicher"
            sub={diskLabel}
            value={formatPct(diskPercent)}
            pct={diskPercent}
            footer={
              formatBytes(diskTotal)
                ? `Gesamt: ${formatBytes(diskTotal)} · Container-Root, nicht NVMe`
                : "Container-Root (/), nicht die NVMe"
            }
          />

          {history.length > 1 && (
            <div className="mt-2 pt-3 border-t border-[var(--border-color)]">
              <div className="flex items-center justify-between mb-2 px-1">
                <p className="text-xs text-[var(--text-secondary)] uppercase tracking-wider">
                  CPU Verlauf
                </p>
                {updatedAt && (
                  <p className="text-[10px] text-[var(--text-secondary)] opacity-70">
                    {new Date(updatedAt).toLocaleTimeString()}
                  </p>
                )}
              </div>
              <div className="h-28 -mx-1">
                <ResponsiveContainer width="100%" height="100%">
                  <AreaChart data={history}>
                    <defs>
                      <linearGradient id="hostCpuGrad" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="5%" stopColor="#22d3ee" stopOpacity={0.35} />
                        <stop offset="95%" stopColor="#22d3ee" stopOpacity={0} />
                      </linearGradient>
                    </defs>
                    <XAxis dataKey="time" hide />
                    <YAxis domain={[0, 100]} hide />
                    <Tooltip
                      contentStyle={{
                        backgroundColor: "var(--bg-card)",
                        borderColor: "var(--border-color)",
                        color: "var(--text-primary)",
                        fontSize: 12,
                      }}
                      formatter={(v) => [`${Math.round(v)}%`, "CPU"]}
                    />
                    <Area
                      type="monotone"
                      dataKey="cpu"
                      stroke="#22d3ee"
                      fillOpacity={1}
                      fill="url(#hostCpuGrad)"
                      isAnimationActive={false}
                    />
                  </AreaChart>
                </ResponsiveContainer>
              </div>
            </div>
          )}
        </div>
      )}
    </GlassCard>
  );
}

HostStatusCard.propTypes = {
  title: PropTypes.string.isRequired,
  subtitle: PropTypes.string.isRequired,
  role: PropTypes.oneOf(["nas", "firewall"]),
  live: PropTypes.bool,
  placeholder: PropTypes.bool,
  placeholderMessage: PropTypes.string,
  cpuPercent: PropTypes.number,
  ramPercent: PropTypes.number,
  diskPercent: PropTypes.number,
  ramTotal: PropTypes.number,
  diskTotal: PropTypes.number,
  diskLabel: PropTypes.string,
  updatedAt: PropTypes.oneOfType([PropTypes.string, PropTypes.number]),
  history: PropTypes.arrayOf(
    PropTypes.shape({
      time: PropTypes.string,
      cpu: PropTypes.number,
    })
  ),
  error: PropTypes.string,
};
