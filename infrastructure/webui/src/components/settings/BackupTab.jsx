import { useState, useEffect } from "react";
import {
    FolderSync,
    Clock,
    Play,
    Check,
    X,
    Loader2,
    Calendar,
    RefreshCw,
    Download
} from "lucide-react";
import { useToast } from "../ui/Toast";
import { apiRequest } from "../../lib/api";

// Glass Card Component
const GlassCard = ({ children, className = "" }) => (
    <div className={`relative overflow-hidden rounded-2xl border border-white/10 bg-slate-900/40 backdrop-blur-xl shadow-2xl ${className}`}>
        <div className="absolute top-0 left-0 w-full h-[1px] bg-gradient-to-r from-transparent via-white/20 to-transparent opacity-50"></div>
        <div className="p-6 h-full flex flex-col">
            {children}
        </div>
    </div>
);

// Section Header Component
const SectionHeader = ({ icon: Icon, title, description }) => (
    <div className="flex items-start gap-4 mb-6">
        <div className="p-3 rounded-xl bg-emerald-500/20 border border-emerald-500/30">
            <Icon size={24} className="text-emerald-400" />
        </div>
        <div>
            <h2 className="text-xl font-bold text-white tracking-tight">{title}</h2>
            <p className="text-slate-400 text-sm mt-1">{description}</p>
        </div>
    </div>
);

// Backup Schedule Options
const SCHEDULE_OPTIONS = [
    { value: "0 3 * * *", label: "Täglich um 03:00" },
    { value: "0 3 * * 0", label: "Wöchentlich (Sonntag 03:00)" },
    { value: "0 3 1 * *", label: "Monatlich (1. des Monats)" },
    { value: "manual", label: "Nur manuell" }
];

export default function BackupTab() {
    const toast = useToast();
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [backingUp, setBackingUp] = useState(false);

    const [schedule, setSchedule] = useState("0 3 * * *");
    const [backups, setBackups] = useState([]);
    const [lastBackup, setLastBackup] = useState(null);

    useEffect(() => {
        loadBackupData();
    }, []);

    const loadBackupData = async () => {
        setLoading(true);
        try {
            // Load settings for schedule from setup.json
            const settings = await apiRequest("/api/v1/backup/settings", { method: "GET" });
            if (settings?.backup_schedule) {
                setSchedule(settings.backup_schedule);
            }

            // Load backup history
            try {
                const backupList = await apiRequest("/api/v1/backups", { method: "GET" });
                if (Array.isArray(backupList)) {
                    setBackups(backupList.slice(0, 10)); // Last 10 backups
                    if (backupList.length > 0) {
                        setLastBackup(backupList[0]);
                    }
                }
            } catch (e) {
                console.log("Backup history not available");
            }
        } catch (err) {
            console.error("Failed to load backup data:", err);
        } finally {
            setLoading(false);
        }
    };

    const handleSaveSchedule = async () => {
        setSaving(true);
        try {
            await apiRequest("/api/v1/backup/settings", {
                method: "PUT",
                body: JSON.stringify({
                    backup_schedule: schedule,
                    backup_destination: "/mnt/backups",
                    backup_retention_days: 30
                })
            });
            toast.success("Backup-Zeitplan gespeichert");
        } catch (err) {
            toast.error("Fehler beim Speichern");
        } finally {
            setSaving(false);
        }
    };

    const handleStartBackup = async () => {
        setBackingUp(true);
        try {
            await apiRequest("/api/v1/backups", { method: "POST" });
            toast.success("Backup gestartet");

            // Poll for completion
            setTimeout(loadBackupData, 5000);
        } catch (err) {
            toast.error("Fehler beim Starten des Backups");
        } finally {
            setBackingUp(false);
        }
    };

    const formatDate = (dateStr) => {
        if (!dateStr) return "—";
        const date = new Date(dateStr);
        return date.toLocaleDateString("de-DE", {
            day: "2-digit",
            month: "2-digit",
            year: "numeric",
            hour: "2-digit",
            minute: "2-digit"
        });
    };

    const formatSize = (bytes) => {
        if (!bytes) return "—";
        if (bytes >= 1024 * 1024 * 1024) {
            return (bytes / (1024 * 1024 * 1024)).toFixed(2) + " GB";
        } else if (bytes >= 1024 * 1024) {
            return (bytes / (1024 * 1024)).toFixed(2) + " MB";
        }
        return bytes + " B";
    };

    const getTimeSince = (dateStr) => {
        if (!dateStr) return "Nie";
        const date = new Date(dateStr);
        const now = new Date();
        const diffMs = now - date;
        const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
        const diffDays = Math.floor(diffHours / 24);

        if (diffDays > 0) return `Vor ${diffDays} Tag${diffDays > 1 ? "en" : ""}`;
        if (diffHours > 0) return `Vor ${diffHours} Stunde${diffHours > 1 ? "n" : ""}`;
        return "Gerade eben";
    };

    if (loading) {
        return (
            <div className="flex items-center justify-center h-64">
                <Loader2 size={48} className="text-emerald-400 animate-spin" />
            </div>
        );
    }

    return (
        <div className="space-y-6">
            {/* Backup Status */}
            <GlassCard>
                <SectionHeader
                    icon={FolderSync}
                    title="Backup Status"
                    description="Übersicht über den aktuellen Backup-Zustand"
                />

                <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 p-4 bg-slate-800/30 rounded-xl">
                    <div className="flex items-center gap-4">
                        <div className={`p-3 rounded-xl ${lastBackup?.status === "success"
                            ? "bg-emerald-500/20"
                            : lastBackup?.status === "failed"
                                ? "bg-rose-500/20"
                                : "bg-slate-500/20"}`}>
                            {lastBackup?.status === "success" ? (
                                <Check size={24} className="text-emerald-400" />
                            ) : lastBackup?.status === "failed" ? (
                                <X size={24} className="text-rose-400" />
                            ) : (
                                <Clock size={24} className="text-slate-400" />
                            )}
                        </div>
                        <div>
                            <p className="text-white font-medium">
                                Letztes Backup: {getTimeSince(lastBackup?.created_at)}
                            </p>
                            <p className="text-slate-400 text-sm">
                                {lastBackup?.status === "success" ? "✅ Erfolgreich"
                                    : lastBackup?.status === "failed" ? "❌ Fehlgeschlagen"
                                        : "Kein Backup vorhanden"}
                            </p>
                        </div>
                    </div>

                    <button
                        onClick={handleStartBackup}
                        disabled={backingUp}
                        className="flex items-center gap-2 px-4 py-2.5 bg-emerald-500/20 hover:bg-emerald-500/30 text-emerald-400 rounded-xl border border-emerald-500/30 transition-all disabled:opacity-50"
                    >
                        {backingUp ? (
                            <Loader2 size={18} className="animate-spin" />
                        ) : (
                            <Play size={18} />
                        )}
                        <span className="font-medium">Jetzt sichern</span>
                    </button>
                </div>
            </GlassCard>

            {/* Schedule */}
            <GlassCard>
                <SectionHeader
                    icon={Calendar}
                    title="Zeitplan"
                    description="Automatisches Backup konfigurieren"
                />

                <div className="space-y-4">
                    <div>
                        <label className="block text-sm font-medium text-slate-300 mb-2">
                            Backup-Intervall
                        </label>
                        <select
                            value={schedule}
                            onChange={(e) => setSchedule(e.target.value)}
                            className="w-full px-4 py-3 bg-slate-800/50 border border-white/10 rounded-xl text-white focus:border-emerald-500/50 focus:outline-none transition-all"
                        >
                            {SCHEDULE_OPTIONS.map((opt) => (
                                <option key={opt.value} value={opt.value}>
                                    {opt.label}
                                </option>
                            ))}
                        </select>
                    </div>

                    <button
                        onClick={handleSaveSchedule}
                        disabled={saving}
                        className="flex items-center gap-2 px-4 py-2.5 bg-blue-500/20 hover:bg-blue-500/30 text-blue-400 rounded-xl border border-blue-500/30 transition-all disabled:opacity-50"
                    >
                        {saving ? (
                            <Loader2 size={18} className="animate-spin" />
                        ) : (
                            <Check size={18} />
                        )}
                        <span className="font-medium">Speichern</span>
                    </button>
                </div>
            </GlassCard>

            {/* Backup History */}
            <GlassCard>
                <SectionHeader
                    icon={Clock}
                    title="Backup-Verlauf"
                    description="Letzte Sicherungen"
                />

                {backups.length === 0 ? (
                    <div className="text-center py-8 text-slate-500">
                        <FolderSync size={48} className="mx-auto mb-3 opacity-50" />
                        <p>Noch keine Backups vorhanden</p>
                    </div>
                ) : (
                    <div className="overflow-x-auto">
                        <table className="w-full">
                            <thead>
                                <tr className="border-b border-white/10">
                                    <th className="text-left py-3 px-4 text-slate-400 font-medium">Datum</th>
                                    <th className="text-left py-3 px-4 text-slate-400 font-medium">Größe</th>
                                    <th className="text-left py-3 px-4 text-slate-400 font-medium">Status</th>
                                    <th className="text-right py-3 px-4 text-slate-400 font-medium">Aktion</th>
                                </tr>
                            </thead>
                            <tbody>
                                {backups.map((backup, idx) => (
                                    <tr key={backup.id || idx} className="border-b border-white/5 hover:bg-white/5">
                                        <td className="py-3 px-4 text-white">
                                            {formatDate(backup.created_at)}
                                        </td>
                                        <td className="py-3 px-4 text-slate-400 font-mono">
                                            {formatSize(backup.size)}
                                        </td>
                                        <td className="py-3 px-4">
                                            <span className={`px-2 py-1 rounded-lg text-xs font-medium ${backup.status === "success"
                                                ? "bg-emerald-500/20 text-emerald-400"
                                                : backup.status === "failed"
                                                    ? "bg-rose-500/20 text-rose-400"
                                                    : "bg-amber-500/20 text-amber-400"
                                                }`}>
                                                {backup.status === "success" ? "Erfolg"
                                                    : backup.status === "failed" ? "Fehler"
                                                        : "Läuft..."}
                                            </span>
                                        </td>
                                        <td className="py-3 px-4 text-right">
                                            {backup.status === "success" && (
                                                <button className="p-2 text-slate-400 hover:text-white transition-colors">
                                                    <Download size={16} />
                                                </button>
                                            )}
                                        </td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
                )}

                <button
                    onClick={loadBackupData}
                    className="flex items-center gap-2 px-4 py-2 mt-4 text-slate-400 hover:text-white transition-colors"
                >
                    <RefreshCw size={16} />
                    <span className="text-sm">Aktualisieren</span>
                </button>
            </GlassCard>
        </div>
    );
}
