import { useSystemHealth } from "../hooks/useSystemHealth";
import { useLiveMetrics } from "../hooks/useLiveMetrics";
import { useVault } from "../context/VaultContext";
import {
  ShieldCheck,
  ShieldAlert,
  Loader2,
  Clock,
  Archive,
  Lock,
  AlertTriangle,
} from "lucide-react";
import { DashboardSkeleton } from "../components/ui/Skeleton";
import HostStatusCard from "../components/HostStatusCard";
import { GlassCard } from "../components/ui/GlassCard";

export default function Dashboard() {
  const { data, isLoading } = useSystemHealth();
  const { metrics, history, error: liveError } = useLiveMetrics(5000);
  const { vaultConfig, isLoading: vaultLoading } = useVault();
  const settings = data?.settings;
  const lastBackup = data?.lastBackup;
  const snapshotCount = data?.snapshotCount || 0;

  const getNextBackupTime = () => {
    if (!settings?.backup_schedule) return "Nicht geplant";

    const now = new Date();
    const [hours, minutes] = settings.backup_schedule.split(":");
    const nextRun = new Date();
    nextRun.setHours(parseInt(hours, 10), parseInt(minutes, 10), 0, 0);

    if (nextRun < now) {
      nextRun.setDate(nextRun.getDate() + 1);
    }

    const isToday = nextRun.toDateString() === now.toDateString();
    const timeStr = settings.backup_schedule;

    return isToday ? `Heute, ${timeStr} Uhr` : `Morgen, ${timeStr} Uhr`;
  };

  const formatLastBackupTime = () => {
    if (!lastBackup) return "Kein Backup vorhanden";

    const date = new Date(lastBackup.modTime || lastBackup.created_at);
    const now = new Date();
    const diffMs = now - date;
    const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
    const diffDays = Math.floor(diffHours / 24);

    if (diffDays > 0) {
      return `vor ${diffDays} Tag${diffDays > 1 ? "en" : ""}`;
    }
    if (diffHours > 0) {
      return `vor ${diffHours} Stunde${diffHours > 1 ? "n" : ""}`;
    }
    return "Kürzlich";
  };

  const isBackupActive = settings?.auto_backup_enabled ?? true;

  if (isLoading && !data) {
    return <DashboardSkeleton />;
  }

  return (
    <div className="space-y-6">
      <div className="sr-only">
        <h1>Dashboard</h1>
      </div>

      {!vaultLoading && !vaultConfig && (
        <div className="relative overflow-hidden rounded-2xl border border-amber-500/40 bg-amber-500/15 backdrop-blur-xl shadow-lg">
          <div className="absolute top-0 left-0 w-full h-[1px] bg-gradient-to-r from-transparent via-amber-400/40 to-transparent" />
          <div className="p-4 flex items-center gap-4">
            <div className="p-3 rounded-xl bg-amber-500/20 border border-amber-500/40 shrink-0">
              <Lock size={24} className="text-amber-700 dark:text-amber-400" />
            </div>
            <div className="flex-1">
              <div className="flex items-center gap-2 mb-1">
                <AlertTriangle size={16} className="text-amber-700 dark:text-amber-400" />
                <p className="text-amber-900 dark:text-amber-200 font-semibold text-sm uppercase tracking-wider">
                  Vault nicht eingerichtet
                </p>
              </div>
              <p className="text-amber-800/90 dark:text-amber-200/80 text-sm">
                Für verschlüsselte Dateispeicherung gehe zu{" "}
                <a
                  href="/files"
                  className="text-amber-900 dark:text-amber-300 hover:underline font-medium"
                >
                  Dateien → vault
                </a>{" "}
                und richte die Ende-zu-Ende-Verschlüsselung ein.
              </p>
            </div>
          </div>
        </div>
      )}

      <div>
        <p className="text-xs uppercase tracking-wider text-[var(--text-secondary)] mb-3">
          Homelab Hosts
        </p>
        <div className="grid grid-cols-1 xl:grid-cols-2 gap-6">
          <HostStatusCard
            role="nas"
            title="NAS · Raspberry Pi 5"
            subtitle="nasserver · 192.168.1.81"
            live
            cpuPercent={metrics?.cpu_percent}
            ramPercent={metrics?.ram_percent}
            diskPercent={metrics?.disk_percent}
            ramTotal={metrics?.ram_total}
            diskTotal={metrics?.disk_total}
            diskLabel="API-Host / Root"
            updatedAt={metrics?.timestamp}
            history={history}
            error={liveError}
          />
          <HostStatusCard
            role="firewall"
            title="Firewall · Raspberry Pi 4"
            subtitle="firewall · 192.168.1.82"
            placeholder
            placeholderMessage="CPU/RAM/Disk für den Firewall-Pi folgen in einem späteren Schritt (eigene Datenquelle, kein API-Scraping)."
          />
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <GlassCard
          className={`lg:col-span-1 ${isBackupActive ? "hover:bg-emerald-500/5" : "hover:bg-slate-500/5"} transition-colors`}
        >
          <div className="flex items-start justify-between mb-4">
            <div
              className={`p-3 rounded-xl ${isBackupActive ? "bg-emerald-500/20 border-emerald-500/30" : "bg-black/10 border-[var(--border-color)]"} border`}
            >
              {isBackupActive ? (
                <ShieldCheck size={24} className="text-emerald-400" />
              ) : (
                <ShieldAlert size={24} className="text-[var(--text-secondary)]" />
              )}
            </div>
            {isLoading && (
              <Loader2 size={16} className="animate-spin text-[var(--text-secondary)]" />
            )}
          </div>

          <div className="flex-1 space-y-4">
            <div>
              <p className="text-[var(--text-secondary)] text-xs uppercase tracking-wider">
                Datensicherheit
              </p>
              <p
                className={`text-2xl font-bold mt-2 ${isBackupActive ? "text-emerald-400" : "text-[var(--text-secondary)]"}`}
              >
                {isBackupActive ? "Auto-Backup" : "Manuell"}
              </p>
            </div>

            <div className="p-3 rounded-lg bg-white/5 border border-[var(--border-color)]">
              <div className="flex items-center gap-2 mb-2">
                <Clock
                  size={14}
                  className={isBackupActive ? "text-emerald-400" : "text-[var(--text-secondary)]"}
                />
                <span className="text-[var(--text-secondary)] text-xs font-medium">
                  Nächster Lauf
                </span>
              </div>
              <p
                className={`text-sm font-semibold ${isBackupActive ? "text-emerald-400" : "text-[var(--text-secondary)]"}`}
              >
                {getNextBackupTime()}
              </p>
            </div>

            <div className="p-3 rounded-lg bg-white/5 border border-[var(--border-color)]">
              <div className="flex items-center gap-2 mb-2">
                <Archive size={14} className="text-blue-400" />
                <span className="text-[var(--text-secondary)] text-xs font-medium">
                  Letzter Snapshot
                </span>
              </div>
              <p className="text-sm font-semibold text-blue-400">{formatLastBackupTime()}</p>
            </div>

            <div className="p-3 rounded-lg bg-blue-500/10 border border-blue-500/20">
              <div className="flex items-center justify-between">
                <span className="text-[var(--text-secondary)] text-xs font-medium">
                  Gespeicherte Snapshots
                </span>
                <span className="text-blue-400 font-bold text-lg">{snapshotCount}</span>
              </div>
              {settings?.backup_retention && (
                <p className="text-[var(--text-secondary)] text-xs mt-1 opacity-70">
                  {settings.backup_retention} Tage Aufbewahrung
                </p>
              )}
            </div>
          </div>
        </GlassCard>
      </div>
    </div>
  );
}
