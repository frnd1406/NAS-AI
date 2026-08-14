import { useState, useEffect } from "react";
import {
    Globe,
    AlertTriangle,
    Save,
    Loader2,
    RefreshCw,
    Gauge,
    Clock,
    Shield,
    Link2,
    Activity,
    Network
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
        <div className="p-3 rounded-xl bg-cyan-500/20 border border-cyan-500/30">
            <Icon size={24} className="text-cyan-400" />
        </div>
        <div>
            <h2 className="text-xl font-bold text-white tracking-tight">{title}</h2>
            <p className="text-slate-400 text-sm mt-1">{description}</p>
        </div>
    </div>
);

// Session Timeout Options
const SESSION_TIMEOUTS = [
    { value: 15, label: "15 Minuten" },
    { value: 30, label: "30 Minuten" },
    { value: 60, label: "1 Stunde" },
    { value: 120, label: "2 Stunden" },
    { value: 480, label: "8 Stunden" },
    { value: 1440, label: "24 Stunden" }
];

const formatBytes = (bytes) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
};

export default function NetworkTab() {
    const toast = useToast();
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [hasChanges, setHasChanges] = useState(false);
    const [metrics, setMetrics] = useState([]);

    const [settings, setSettings] = useState({
        rate_limit_per_min: 100,
        session_timeout_mins: 60,
        cors_origins: ""
    });

    const [originalSettings, setOriginalSettings] = useState(null);

    useEffect(() => {
        loadData();
    }, []);

    useEffect(() => {
        if (originalSettings) {
            const changed =
                settings.rate_limit_per_min !== originalSettings.rate_limit_per_min ||
                settings.session_timeout_mins !== originalSettings.session_timeout_mins ||
                settings.cors_origins !== originalSettings.cors_origins;
            setHasChanges(changed);
        }
    }, [settings, originalSettings]);

    const loadData = async () => {
        setLoading(true);
        try {
            const [settingsData, metricsData] = await Promise.all([
                apiRequest("/api/v1/network/settings", { method: "GET" }),
                apiRequest("/api/v1/system/hardware/network", { method: "GET" }).catch(() => [])
            ]);

            const loadedSettings = {
                rate_limit_per_min: settingsData?.rate_limit_per_min || 100,
                session_timeout_mins: settingsData?.session_timeout_mins || 60,
                cors_origins: Array.isArray(settingsData?.cors_origins)
                    ? settingsData.cors_origins.join("\n")
                    : (settingsData?.cors_origins || "")
            };

            setSettings(loadedSettings);
            setOriginalSettings(loadedSettings);
            setMetrics(metricsData || []);
        } catch (err) {
            toast.error("Daten konnten nicht geladen werden");
            console.error(err);
        } finally {
            setLoading(false);
        }
    };

    const handleSave = async () => {
        setSaving(true);
        try {
            // Parse CORS origins from textarea (one per line)
            const corsArray = settings.cors_origins
                .split("\n")
                .map(s => s.trim())
                .filter(s => s.length > 0);

            await apiRequest("/api/v1/network/settings", {
                method: "PUT",
                body: JSON.stringify({
                    rate_limit_per_min: parseInt(settings.rate_limit_per_min),
                    session_timeout_mins: parseInt(settings.session_timeout_mins),
                    cors_origins: corsArray
                })
            });

            toast.success("Einstellungen gespeichert");

            // Update original to track changes
            setOriginalSettings({ ...settings });
            setHasChanges(false);
        } catch (err) {
            toast.error("Fehler beim Speichern");
        } finally {
            setSaving(false);
        }
    };

    if (loading) {
        return (
            <div className="flex items-center justify-center h-64">
                <Loader2 size={48} className="text-cyan-400 animate-spin" />
            </div>
        );
    }

    return (
        <div className="space-y-6">
            {/* Live Metrics */}
            <GlassCard>
                <SectionHeader
                    icon={Activity}
                    title="Netzwerk Status"
                    description="Aktuelle Schnittstellen und Traffic"
                />

                <div className="overflow-x-auto">
                    <table className="w-full text-left border-collapse">
                        <thead>
                            <tr className="text-slate-400 border-b border-white/10 text-xs uppercase tracking-wider">
                                <th className="p-3">Interface</th>
                                <th className="p-3">IP Adresse</th>
                                <th className="p-3">MAC</th>
                                <th className="p-3 text-right">Download</th>
                                <th className="p-3 text-right">Upload</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-white/5">
                            {metrics.map((iface) => (
                                <tr key={iface.name} className="hover:bg-white/5 transition-colors">
                                    <td className="p-3 flex items-center gap-2 font-medium text-cyan-400">
                                        <Network size={16} />
                                        {iface.name}
                                    </td>
                                    <td className="p-3 text-slate-300 text-sm font-mono">
                                        {iface.ips?.join(", ") || "-"}
                                    </td>
                                    <td className="p-3 text-slate-400 text-xs font-mono">
                                        {iface.mac}
                                    </td>
                                    <td className="p-3 text-right text-slate-300 font-mono text-sm">
                                        {formatBytes(iface.bytes_recv)}
                                    </td>
                                    <td className="p-3 text-right text-slate-300 font-mono text-sm">
                                        {formatBytes(iface.bytes_sent)}
                                    </td>
                                </tr>
                            ))}
                            {metrics.length === 0 && (
                                <tr>
                                    <td colSpan="5" className="p-4 text-center text-slate-500 text-sm italic">
                                        Keine Netzwerkschnittstellen gefunden
                                    </td>
                                </tr>
                            )}
                        </tbody>
                    </table>
                </div>
            </GlassCard>

            {/* Restart Warning Banner (shown when changes exist) */}
            {hasChanges && (
                <div className="p-4 bg-amber-500/10 border border-amber-500/30 rounded-xl flex items-start gap-3 animate-pulse">
                    <AlertTriangle size={20} className="text-amber-400 mt-0.5" />
                    <div>
                        <p className="text-amber-400 font-medium">Neustart erforderlich</p>
                        <p className="text-slate-400 text-sm mt-1">
                            Änderungen an Netzwerkeinstellungen werden erst nach einem Server-Neustart aktiv.
                        </p>
                    </div>
                </div>
            )}

            {/* Rate Limiting */}
            <GlassCard>
                <SectionHeader
                    icon={Gauge}
                    title="Rate Limiting"
                    description="Schutz vor Brute-Force und DDoS-Angriffen"
                />

                <div className="space-y-4">
                    <div>
                        <label className="block text-sm font-medium text-slate-300 mb-2">
                            Anfragen pro Minute
                        </label>
                        <div className="flex items-center gap-4">
                            <input
                                type="range"
                                min="10"
                                max="500"
                                step="10"
                                value={settings.rate_limit_per_min}
                                onChange={(e) => setSettings({ ...settings, rate_limit_per_min: parseInt(e.target.value) })}
                                className="flex-1 h-2 bg-slate-700 rounded-lg appearance-none cursor-pointer accent-cyan-500"
                            />
                            <div className="w-24 px-3 py-2 bg-slate-800/50 border border-white/10 rounded-xl text-white text-center font-mono">
                                {settings.rate_limit_per_min}
                            </div>
                        </div>
                        <div className="flex justify-between text-xs text-slate-500 mt-1">
                            <span>Strikt (10)</span>
                            <span>Normal (100)</span>
                            <span>Locker (500)</span>
                        </div>
                    </div>
                </div>
            </GlassCard>

            {/* Session Management */}
            <GlassCard>
                <SectionHeader
                    icon={Clock}
                    title="Session-Verwaltung"
                    description="Automatische Abmeldung nach Inaktivität"
                />

                <div>
                    <label className="block text-sm font-medium text-slate-300 mb-2">
                        Session-Timeout
                    </label>
                    <select
                        value={settings.session_timeout_mins}
                        onChange={(e) => setSettings({ ...settings, session_timeout_mins: parseInt(e.target.value) })}
                        className="w-full px-4 py-3 bg-slate-800/50 border border-white/10 rounded-xl text-white focus:border-cyan-500/50 focus:outline-none transition-all"
                    >
                        {SESSION_TIMEOUTS.map((opt) => (
                            <option key={opt.value} value={opt.value}>
                                {opt.label}
                            </option>
                        ))}
                    </select>
                    <p className="text-slate-500 text-sm mt-2">
                        Benutzer werden nach dieser Zeit der Inaktivität automatisch abgemeldet.
                    </p>
                </div>
            </GlassCard>

            {/* CORS Origins */}
            <GlassCard>
                <SectionHeader
                    icon={Link2}
                    title="CORS Origins"
                    description="Erlaubte Ursprünge für API-Anfragen"
                />

                <div className="space-y-4">
                    <div>
                        <label className="block text-sm font-medium text-slate-300 mb-2">
                            Erlaubte Domains (eine pro Zeile)
                        </label>
                        <textarea
                            value={settings.cors_origins}
                            onChange={(e) => setSettings({ ...settings, cors_origins: e.target.value })}
                            placeholder="https://nas.local&#10;https://app.example.com"
                            rows={4}
                            className="w-full px-4 py-3 bg-slate-800/50 border border-white/10 rounded-xl text-white placeholder:text-slate-500 focus:border-cyan-500/50 focus:outline-none transition-all font-mono text-sm resize-none"
                        />
                        <p className="text-slate-500 text-sm mt-2">
                            Leer lassen für Standard-Konfiguration. Wildcard (*) erlaubt alle Origins.
                        </p>
                    </div>
                </div>
            </GlassCard>

            {/* Actions */}
            <div className="flex items-center gap-4">
                <button
                    onClick={handleSave}
                    disabled={saving || !hasChanges}
                    className="flex items-center gap-2 px-6 py-3 bg-cyan-500/20 hover:bg-cyan-500/30 text-cyan-400 rounded-xl border border-cyan-500/30 transition-all disabled:opacity-50 disabled:cursor-not-allowed font-medium"
                >
                    {saving ? (
                        <Loader2 size={18} className="animate-spin" />
                    ) : (
                        <Save size={18} />
                    )}
                    <span>Speichern</span>
                </button>

                <button
                    onClick={loadData}
                    className="flex items-center gap-2 px-4 py-3 text-slate-400 hover:text-white transition-colors"
                >
                    <RefreshCw size={16} />
                    <span>Zurücksetzen</span>
                </button>

                {hasChanges && (
                    <div className="flex-1 text-right">
                        <span className="text-amber-400 text-sm">
                            ⚠️ Ungespeicherte Änderungen
                        </span>
                    </div>
                )}
            </div>

            {/* CLI Restart Info */}
            <GlassCard className="bg-slate-900/60">
                <div className="flex items-start gap-3">
                    <Shield size={20} className="text-slate-400 mt-0.5" />
                    <div>
                        <p className="text-slate-300 font-medium">Server-Neustart</p>
                        <p className="text-slate-500 text-sm mt-1">
                            Um Änderungen zu aktivieren, starte den Server neu:
                        </p>
                        <code className="block mt-2 px-3 py-2 bg-slate-800/50 rounded-lg text-cyan-400 text-sm font-mono">
                            docker-compose restart api
                        </code>
                    </div>
                </div>
            </GlassCard>
        </div>
    );
}
