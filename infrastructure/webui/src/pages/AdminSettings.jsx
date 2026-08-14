import { useState, useEffect } from "react";
import {
    Shield,
    Users,
    Activity,
    Clock,
    Database,
    AlertTriangle,
    RefreshCw,
    Loader2,
    Power,
    PowerOff,
    UserCog,
    FileText,
    Globe,
    Zap,
    Lock
} from "lucide-react";
import { useToast } from "../components/Toast";
import { apiRequest } from "../lib/api";

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
        <div className="p-3 rounded-xl bg-violet-500/20 border border-violet-500/30">
            <Icon size={24} className="text-violet-400" />
        </div>
        <div>
            <h2 className="text-xl font-bold text-white tracking-tight">{title}</h2>
            <p className="text-slate-400 text-sm mt-1">{description}</p>
        </div>
    </div>
);

// Settings Row Component
const SettingsRow = ({ label, description, children }) => (
    <div className="flex items-center justify-between py-4 border-b border-white/5 last:border-0">
        <div className="flex-1">
            <p className="text-white font-medium">{label}</p>
            {description && <p className="text-slate-500 text-sm mt-0.5">{description}</p>}
        </div>
        <div className="ml-4">
            {children}
        </div>
    </div>
);

// Main Admin Settings Component
export default function AdminSettings() {
    const toast = useToast();
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);

    // Admin settings state
    const [settings, setSettings] = useState({
        rate_limit_per_min: 100,
        cors_origins: [],
        ai_service_url: "",
        maintenance_mode: false,
        session_timeout_mins: 60,
        max_login_attempts: 5,
        two_factor_enabled: false,
        ip_whitelist: [],
        audit_log_retention_days: 30
    });

    // System status state
    const [systemStatus, setSystemStatus] = useState(null);

    // Users state
    const [users, setUsers] = useState([]);

    // Audit logs state
    const [auditLogs, setAuditLogs] = useState([]);

    // Load all data on mount
    useEffect(() => {
        loadAdminData();
    }, []);

    const loadAdminData = async () => {
        setLoading(true);
        try {
            const [settingsData, statusData, usersData, logsData] = await Promise.all([
                apiRequest("/api/v1/admin/settings", { method: "GET" }),
                apiRequest("/api/v1/admin/status", { method: "GET" }),
                apiRequest("/api/v1/admin/users", { method: "GET" }),
                apiRequest("/api/v1/admin/audit-logs?limit=20", { method: "GET" })
            ]);

            setSettings(settingsData);
            setSystemStatus(statusData);
            setUsers(usersData.users || []);
            setAuditLogs(logsData.logs || []);
        } catch (err) {
            toast.error("Fehler beim Laden der Admin-Daten");
            console.error("Admin data load error:", err);
        } finally {
            setLoading(false);
        }
    };

    const updateSettings = async (updates) => {
        setSaving(true);
        try {
            await apiRequest("/api/v1/admin/settings", {
                method: "PUT",
                body: JSON.stringify(updates)
            });
            setSettings({ ...settings, ...updates });
            toast.success("Einstellungen gespeichert");
        } catch (err) {
            toast.error(err.message || "Fehler beim Speichern");
        } finally {
            setSaving(false);
        }
    };

    const toggleMaintenanceMode = async () => {
        try {
            const result = await apiRequest("/api/v1/admin/maintenance", {
                method: "POST",
                body: JSON.stringify({
                    enabled: !settings.maintenance_mode,
                    message: "System wird gewartet. Bitte warten Sie."
                })
            });
            setSettings({ ...settings, maintenance_mode: result.maintenance_mode });
            toast.success(result.maintenance_mode ? "Wartungsmodus aktiviert" : "Wartungsmodus deaktiviert");
        } catch (err) {
            toast.error("Fehler beim Umschalten des Wartungsmodus");
        }
    };

    const updateUserRole = async (userId, newRole) => {
        try {
            await apiRequest(`/api/v1/admin/users/${userId}/role`, {
                method: "PUT",
                body: JSON.stringify({ role: newRole })
            });
            setUsers(users.map(u => u.id === userId ? { ...u, role: newRole } : u));
            toast.success("Benutzerrolle aktualisiert");
        } catch (err) {
            toast.error("Fehler beim Aktualisieren der Rolle");
        }
    };

    if (loading) {
        return (
            <div className="flex items-center justify-center h-64">
                <Loader2 size={48} className="text-violet-400 animate-spin" />
            </div>
        );
    }

    return (
        <div className="space-y-6">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-3xl font-bold text-white tracking-tight flex items-center gap-3">
                        <Shield className="text-violet-400" />
                        Admin Einstellungen
                    </h1>
                    <p className="text-slate-400 mt-2 text-sm">Erweiterte Systemkonfiguration (nur für Admins)</p>
                </div>
                <button
                    onClick={loadAdminData}
                    className="flex items-center gap-2 px-4 py-2 bg-slate-800 hover:bg-slate-700 text-white rounded-xl transition-all"
                >
                    <RefreshCw size={18} />
                    Aktualisieren
                </button>
            </div>

            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                {/* System Status */}
                <GlassCard>
                    <SectionHeader
                        icon={Activity}
                        title="System Status"
                        description="Live-Systemmetriken und Datenbankstatus"
                    />

                    {systemStatus && (
                        <div className="space-y-3">
                            <div className="flex items-center justify-between p-3 bg-slate-800/30 rounded-lg">
                                <span className="text-slate-400">Uptime</span>
                                <span className="text-white font-mono">{systemStatus.uptime}</span>
                            </div>
                            <div className="flex items-center justify-between p-3 bg-slate-800/30 rounded-lg">
                                <span className="text-slate-400">Go Version</span>
                                <span className="text-white font-mono">{systemStatus.go_version}</span>
                            </div>
                            <div className="flex items-center justify-between p-3 bg-slate-800/30 rounded-lg">
                                <span className="text-slate-400">Goroutines</span>
                                <span className="text-white font-mono">{systemStatus.num_goroutines}</span>
                            </div>
                            <div className="flex items-center justify-between p-3 bg-slate-800/30 rounded-lg">
                                <span className="text-slate-400">Memory</span>
                                <span className="text-white font-mono">{systemStatus.memory_alloc_mb?.toFixed(2)} MB</span>
                            </div>

                            {/* DB Pool Status */}
                            <div className="mt-4 p-4 bg-emerald-500/10 border border-emerald-500/20 rounded-xl">
                                <div className="flex items-center gap-2 text-emerald-400 mb-2">
                                    <Database size={16} />
                                    <span className="font-medium">Database Pool</span>
                                </div>
                                <div className="grid grid-cols-2 gap-2 text-sm">
                                    <span className="text-slate-400">Connections:</span>
                                    <span className="text-white">{systemStatus.db_pool?.open_connections || 0}</span>
                                    <span className="text-slate-400">In Use:</span>
                                    <span className="text-white">{systemStatus.db_pool?.in_use || 0}</span>
                                    <span className="text-slate-400">Idle:</span>
                                    <span className="text-white">{systemStatus.db_pool?.idle || 0}</span>
                                </div>
                            </div>
                        </div>
                    )}
                </GlassCard>

                {/* Rate Limiting & Security */}
                <GlassCard>
                    <SectionHeader
                        icon={Lock}
                        title="Sicherheit & Rate Limiting"
                        description="Zugriffsbeschränkungen konfigurieren"
                    />

                    <div className="space-y-2">
                        <SettingsRow
                            label="Rate Limit"
                            description={`${settings.rate_limit_per_min} Anfragen/Minute`}
                        >
                            <input
                                type="number"
                                value={settings.rate_limit_per_min}
                                onChange={(e) => {
                                    const value = parseInt(e.target.value);
                                    if (value >= 1) {
                                        setSettings({ ...settings, rate_limit_per_min: value });
                                    }
                                }}
                                onBlur={() => updateSettings({ rate_limit_per_min: settings.rate_limit_per_min })}
                                className="w-24 px-3 py-2 bg-slate-800 border border-white/10 rounded-lg text-white text-center"
                            />
                        </SettingsRow>

                        <SettingsRow
                            label="Session Timeout"
                            description="Minuten bis zur automatischen Abmeldung"
                        >
                            <select
                                value={settings.session_timeout_mins}
                                onChange={(e) => updateSettings({ session_timeout_mins: parseInt(e.target.value) })}
                                className="px-3 py-2 bg-slate-800 border border-white/10 rounded-lg text-white"
                            >
                                <option value="15">15 min</option>
                                <option value="30">30 min</option>
                                <option value="60">1 Stunde</option>
                                <option value="120">2 Stunden</option>
                                <option value="480">8 Stunden</option>
                            </select>
                        </SettingsRow>

                        <SettingsRow
                            label="Max. Loginversuche"
                            description="Bevor Account gesperrt wird"
                        >
                            <input
                                type="number"
                                value={settings.max_login_attempts}
                                onChange={(e) => {
                                    const value = parseInt(e.target.value);
                                    if (value >= 1 && value <= 10) {
                                        updateSettings({ max_login_attempts: value });
                                    }
                                }}
                                className="w-20 px-3 py-2 bg-slate-800 border border-white/10 rounded-lg text-white text-center"
                                min="1"
                                max="10"
                            />
                        </SettingsRow>
                    </div>
                </GlassCard>

                {/* Maintenance Mode */}
                <GlassCard>
                    <SectionHeader
                        icon={AlertTriangle}
                        title="Wartungsmodus"
                        description="System für Wartungsarbeiten sperren"
                    />

                    <div className="flex items-center justify-between p-4 bg-slate-800/30 rounded-xl">
                        <div className="flex items-center gap-3">
                            {settings.maintenance_mode ? (
                                <div className="p-2 bg-amber-500/20 rounded-lg">
                                    <PowerOff size={24} className="text-amber-400" />
                                </div>
                            ) : (
                                <div className="p-2 bg-emerald-500/20 rounded-lg">
                                    <Power size={24} className="text-emerald-400" />
                                </div>
                            )}
                            <div>
                                <p className="text-white font-medium">
                                    {settings.maintenance_mode ? "Wartungsmodus AKTIV" : "System online"}
                                </p>
                                <p className="text-slate-500 text-sm">
                                    {settings.maintenance_mode
                                        ? "Benutzer sehen Wartungsmeldung"
                                        : "Normaler Betrieb"
                                    }
                                </p>
                            </div>
                        </div>
                        <button
                            onClick={toggleMaintenanceMode}
                            className={`px-4 py-2 rounded-xl font-medium transition-all ${settings.maintenance_mode
                                    ? "bg-emerald-500/20 text-emerald-400 hover:bg-emerald-500/30 border border-emerald-500/30"
                                    : "bg-amber-500/20 text-amber-400 hover:bg-amber-500/30 border border-amber-500/30"
                                }`}
                        >
                            {settings.maintenance_mode ? "Deaktivieren" : "Aktivieren"}
                        </button>
                    </div>
                </GlassCard>

                {/* CORS Origins */}
                <GlassCard>
                    <SectionHeader
                        icon={Globe}
                        title="CORS Origins"
                        description="Erlaubte Domains für API-Zugriff"
                    />

                    <div className="space-y-2">
                        {settings.cors_origins?.length > 0 ? (
                            settings.cors_origins.map((origin, index) => (
                                <div key={index} className="flex items-center gap-2 p-3 bg-slate-800/30 rounded-lg">
                                    <Globe size={16} className="text-blue-400" />
                                    <span className="text-white font-mono text-sm">{origin}</span>
                                </div>
                            ))
                        ) : (
                            <div className="text-center py-4 text-slate-500">
                                Keine Origins konfiguriert (alle erlaubt)
                            </div>
                        )}
                    </div>
                </GlassCard>
            </div>

            {/* User Management */}
            <GlassCard>
                <SectionHeader
                    icon={Users}
                    title="Benutzerverwaltung"
                    description="Benutzer verwalten und Rollen zuweisen"
                />

                <div className="overflow-x-auto">
                    <table className="w-full">
                        <thead>
                            <tr className="border-b border-white/10">
                                <th className="text-left py-3 px-4 text-slate-400 font-medium">Benutzer</th>
                                <th className="text-left py-3 px-4 text-slate-400 font-medium">E-Mail</th>
                                <th className="text-left py-3 px-4 text-slate-400 font-medium">Rolle</th>
                                <th className="text-left py-3 px-4 text-slate-400 font-medium">Verifiziert</th>
                                <th className="text-left py-3 px-4 text-slate-400 font-medium">Erstellt</th>
                                <th className="text-right py-3 px-4 text-slate-400 font-medium">Aktion</th>
                            </tr>
                        </thead>
                        <tbody>
                            {users.map((user) => (
                                <tr key={user.id} className="border-b border-white/5 hover:bg-white/5">
                                    <td className="py-3 px-4">
                                        <div className="flex items-center gap-3">
                                            <div className="w-8 h-8 rounded-full bg-gradient-to-br from-violet-500 to-blue-500 flex items-center justify-center text-white font-bold text-sm">
                                                {user.username?.charAt(0).toUpperCase()}
                                            </div>
                                            <span className="text-white font-medium">{user.username}</span>
                                        </div>
                                    </td>
                                    <td className="py-3 px-4 text-slate-400">{user.email}</td>
                                    <td className="py-3 px-4">
                                        <span className={`px-2 py-1 rounded-lg text-xs font-medium ${user.role === "admin"
                                                ? "bg-violet-500/20 text-violet-400"
                                                : "bg-slate-500/20 text-slate-400"
                                            }`}>
                                            {user.role}
                                        </span>
                                    </td>
                                    <td className="py-3 px-4">
                                        {user.email_verified ? (
                                            <span className="text-emerald-400">✓</span>
                                        ) : (
                                            <span className="text-slate-500">–</span>
                                        )}
                                    </td>
                                    <td className="py-3 px-4 text-slate-400 text-sm">
                                        {new Date(user.created_at).toLocaleDateString("de-DE")}
                                    </td>
                                    <td className="py-3 px-4 text-right">
                                        <select
                                            value={user.role}
                                            onChange={(e) => updateUserRole(user.id, e.target.value)}
                                            className="px-2 py-1 bg-slate-800 border border-white/10 rounded-lg text-white text-sm"
                                        >
                                            <option value="user">User</option>
                                            <option value="admin">Admin</option>
                                        </select>
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
            </GlassCard>

            {/* Audit Logs */}
            <GlassCard>
                <SectionHeader
                    icon={FileText}
                    title="Audit Logs"
                    description="Sicherheitsrelevante Ereignisse"
                />

                {auditLogs.length > 0 ? (
                    <div className="space-y-2 max-h-64 overflow-y-auto">
                        {auditLogs.map((log) => (
                            <div key={log.id} className="flex items-center gap-4 p-3 bg-slate-800/30 rounded-lg">
                                <div className="w-10 h-10 rounded-lg bg-slate-700 flex items-center justify-center">
                                    <FileText size={16} className="text-slate-400" />
                                </div>
                                <div className="flex-1">
                                    <p className="text-white text-sm font-medium">{log.action}</p>
                                    <p className="text-slate-500 text-xs">{log.resource} • {log.ip_address}</p>
                                </div>
                                <span className="text-slate-500 text-xs">
                                    {new Date(log.created_at).toLocaleString("de-DE")}
                                </span>
                            </div>
                        ))}
                    </div>
                ) : (
                    <div className="text-center py-8 text-slate-500">
                        <FileText size={32} className="mx-auto mb-2 opacity-50" />
                        <p>Keine Audit-Logs vorhanden</p>
                        <p className="text-xs mt-1">Audit-Logging ist noch nicht konfiguriert</p>
                    </div>
                )}
            </GlassCard>
        </div>
    );
}
