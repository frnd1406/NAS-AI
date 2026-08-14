import { useState, useEffect, lazy, Suspense } from "react";
import { useSearchParams } from "react-router-dom";
import {
    User,
    Mail,
    Lock,
    Palette,
    Monitor,
    Info,
    Server,
    Cpu,
    HardDrive,
    Shield,
    Save,
    Eye,
    EyeOff,
    Check,
    AlertCircle,
    Loader2,
    ChevronRight,
    Users,
    Activity,
    Database,
    AlertTriangle,
    RefreshCw,
    Power,
    PowerOff,
    Globe,
    FileText,
    Brain,
    Zap,
    Thermometer,
    FolderSync,
    TestTube,
    CircleCheck,
    CircleX,
    KeyRound,
    Unlock,
    FolderOpen
} from "lucide-react";
import { useTheme } from "../components/ThemeToggle";
import { useToast } from '../components/ui/Toast';
import { apiRequest } from "../lib/api";

// Lazy load new settings tabs
const StorageTab = lazy(() => import("../components/settings/StorageTab"));
const BackupTab = lazy(() => import("../components/settings/BackupTab"));
const LogsTab = lazy(() => import("../components/settings/LogsTab"));
const SecurityTab = lazy(() => import("../components/settings/SecurityTab"));
const NetworkTab = lazy(() => import("../components/settings/NetworkTab"));

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
        <div className="p-3 rounded-xl bg-blue-500/20 border border-blue-500/30">
            <Icon size={24} className="text-blue-400" />
        </div>
        <div>
            <h2 className="text-xl font-bold text-white tracking-tight">{title}</h2>
            <p className="text-slate-400 text-sm mt-1">{description}</p>
        </div>
    </div>
);

// Settings Item Row
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

export default function Settings() {
    const { theme, toggleTheme } = useTheme();
    const toast = useToast();
    const [searchParams] = useSearchParams();
    const activeTab = searchParams.get('tab') || 'profile';

    // Profile state
    const [profile, setProfile] = useState({
        username: "",
        email: ""
    });
    const [profileLoading, setProfileLoading] = useState(false);

    // Password change state
    const [passwords, setPasswords] = useState({
        current: "",
        new: "",
        confirm: ""
    });
    const [showPasswords, setShowPasswords] = useState({
        current: false,
        new: false,
        confirm: false
    });
    const [passwordSaving, setPasswordSaving] = useState(false);

    // System info state
    const [systemInfo, setSystemInfo] = useState(null);
    const [systemLoading, setSystemLoading] = useState(false);

    // Load user profile
    useEffect(() => {
        // In future: Fetch from API /api/v1/user/me
        // For now, we assume user is logged in if they access this page
        setProfile({
            username: "Admin User",
            email: "admin@nas.local"
        });
    }, []);

    // Load system info for About section
    useEffect(() => {
        if (activeTab === "about") {
            loadSystemInfo();
        }
    }, [activeTab]);

    const loadSystemInfo = async () => {
        setSystemLoading(true);
        try {
            const data = await apiRequest("/api/v1/system/health", { method: "GET" });
            setSystemInfo(data);
        } catch (err) {
            toast.error("Systeminfo konnte nicht geladen werden");
        } finally {
            setSystemLoading(false);
        }
    };

    const handlePasswordChange = async () => {
        if (passwords.new !== passwords.confirm) {
            toast.error("Passwörter stimmen nicht überein");
            return;
        }
        if (passwords.new.length < 8) {
            toast.error("Passwort muss mindestens 8 Zeichen haben");
            return;
        }

        setPasswordSaving(true);
        try {
            await apiRequest("/api/v1/user/password", {
                method: "POST",
                body: JSON.stringify({
                    current_password: passwords.current,
                    new_password: passwords.new
                })
            });
            toast.success("Passwort erfolgreich geändert");
            setPasswords({ current: "", new: "", confirm: "" });
        } catch (err) {
            toast.error(err.message || "Fehler beim Ändern des Passworts");
        } finally {
            setPasswordSaving(false);
        }
    };

    const tabs = [
        { id: "profile", label: "Profil", icon: User },
        { id: "crypto", label: "Crypto / Files", icon: KeyRound },
        { id: "security", label: "Security", icon: Shield },
        { id: "backup", label: "Backup", icon: FolderSync },
        { id: "network", label: "Netzwerk", icon: Globe },
        { id: "storage", label: "Speicher", icon: HardDrive },
        { id: "logs", label: "Logs", icon: FileText },
        { id: "appearance", label: "Erscheinung", icon: Palette },
        { id: "admin", label: "Admin", icon: Users, adminOnly: true },
        { id: "about", label: "Über", icon: Info }
    ];

    return (
        <div className="space-y-6">
            {/* Header */}
            <div>
                <h1 className="text-3xl font-bold text-white tracking-tight">Einstellungen</h1>
                <p className="text-slate-400 mt-2 text-sm">Konfiguriere dein NAS.AI System</p>
            </div>

            <div className="flex gap-6">
                {/* Main Content - Full width since sidebar is now in Layout */}
                <div className="flex-1">
                    {/* Profile Section */}
                    {activeTab === "profile" && (
                        <div className="space-y-6">
                            <GlassCard>
                                <SectionHeader
                                    icon={User}
                                    title="Profil Informationen"
                                    description="Deine persönlichen Daten und Kontoeinstellungen"
                                />

                                <div className="space-y-4">
                                    <div>
                                        <label className="block text-sm font-medium text-slate-300 mb-2">Benutzername</label>
                                        <div className="relative">
                                            <User size={18} className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-500" />
                                            <input
                                                type="text"
                                                value={profile.username}
                                                onChange={(e) => setProfile({ ...profile, username: e.target.value })}
                                                className="w-full pl-10 pr-4 py-3 bg-slate-800/50 border border-white/10 rounded-xl text-white focus:border-blue-500/50 focus:outline-none transition-all"
                                                placeholder="Dein Benutzername"
                                            />
                                        </div>
                                    </div>

                                    <div>
                                        <label className="block text-sm font-medium text-slate-300 mb-2">E-Mail</label>
                                        <div className="relative">
                                            <Mail size={18} className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-500" />
                                            <input
                                                type="email"
                                                value={profile.email}
                                                onChange={(e) => setProfile({ ...profile, email: e.target.value })}
                                                className="w-full pl-10 pr-4 py-3 bg-slate-800/50 border border-white/10 rounded-xl text-white focus:border-blue-500/50 focus:outline-none transition-all"
                                                placeholder="name@example.com"
                                            />
                                        </div>
                                    </div>

                                    <button className="flex items-center gap-2 px-4 py-2.5 bg-blue-500/20 hover:bg-blue-500/30 text-blue-400 rounded-xl border border-blue-500/30 transition-all mt-4">
                                        <Save size={18} />
                                        <span className="font-medium">Änderungen speichern</span>
                                    </button>
                                </div>
                            </GlassCard>

                            {/* Password Change */}
                            <GlassCard>
                                <SectionHeader
                                    icon={Lock}
                                    title="Passwort ändern"
                                    description="Aktualisiere dein Passwort für mehr Sicherheit"
                                />

                                <div className="space-y-4">
                                    {["current", "new", "confirm"].map((field) => (
                                        <div key={field}>
                                            <label className="block text-sm font-medium text-slate-300 mb-2">
                                                {field === "current" ? "Aktuelles Passwort" : field === "new" ? "Neues Passwort" : "Passwort bestätigen"}
                                            </label>
                                            <div className="relative">
                                                <Lock size={18} className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-500" />
                                                <input
                                                    type={showPasswords[field] ? "text" : "password"}
                                                    value={passwords[field]}
                                                    onChange={(e) => setPasswords({ ...passwords, [field]: e.target.value })}
                                                    className="w-full pl-10 pr-12 py-3 bg-slate-800/50 border border-white/10 rounded-xl text-white focus:border-blue-500/50 focus:outline-none transition-all"
                                                    placeholder="••••••••"
                                                />
                                                <button
                                                    type="button"
                                                    onClick={() => setShowPasswords({ ...showPasswords, [field]: !showPasswords[field] })}
                                                    className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-500 hover:text-white transition-colors"
                                                >
                                                    {showPasswords[field] ? <EyeOff size={18} /> : <Eye size={18} />}
                                                </button>
                                            </div>
                                        </div>
                                    ))}

                                    {passwords.new && passwords.new.length < 8 && (
                                        <div className="flex items-center gap-2 text-amber-400 text-sm">
                                            <AlertCircle size={16} />
                                            <span>Mindestens 8 Zeichen erforderlich</span>
                                        </div>
                                    )}

                                    {passwords.new && passwords.confirm && passwords.new !== passwords.confirm && (
                                        <div className="flex items-center gap-2 text-rose-400 text-sm">
                                            <AlertCircle size={16} />
                                            <span>Passwörter stimmen nicht überein</span>
                                        </div>
                                    )}

                                    <button
                                        onClick={handlePasswordChange}
                                        disabled={passwordSaving || !passwords.current || !passwords.new || passwords.new !== passwords.confirm}
                                        className="flex items-center gap-2 px-4 py-2.5 bg-violet-500/20 hover:bg-violet-500/30 text-violet-400 rounded-xl border border-violet-500/30 transition-all disabled:opacity-50 disabled:cursor-not-allowed mt-4"
                                    >
                                        {passwordSaving ? (
                                            <>
                                                <Loader2 size={18} className="animate-spin" />
                                                <span className="font-medium">Speichern...</span>
                                            </>
                                        ) : (
                                            <>
                                                <Shield size={18} />
                                                <span className="font-medium">Passwort ändern</span>
                                            </>
                                        )}
                                    </button>
                                </div>
                            </GlassCard>
                        </div>
                    )}

                    {/* AI Settings Section — UI hidden; AISettingsTab kept */}
                    {false && activeTab === "ai" && (
                        <AISettingsTab />
                    )}

                    {/* Crypto Settings Section */}
                    {activeTab === "crypto" && (
                        <CryptoSettingsTab />
                    )}

                    {/* Appearance Section */}
                    {activeTab === "appearance" && (
                        <GlassCard>
                            <SectionHeader
                                icon={Palette}
                                title="Erscheinungsbild"
                                description="Passe das Aussehen deiner Oberfläche an"
                            />

                            <div className="space-y-2">
                                <SettingsRow
                                    label="Theme"
                                    description={theme === "dark" ? "Dunkles Theme aktiv" : "Helles Theme aktiv"}
                                >
                                    <button
                                        onClick={toggleTheme}
                                        className={`relative w-14 h-8 rounded-full transition-colors ${theme === "dark" ? "bg-blue-500/30" : "bg-slate-700"
                                            }`}
                                    >
                                        <div className={`absolute top-1 w-6 h-6 rounded-full bg-white shadow-lg transition-all ${theme === "dark" ? "left-7" : "left-1"
                                            }`} />
                                    </button>
                                </SettingsRow>

                                <SettingsRow
                                    label="Sprache"
                                    description="Deutsch"
                                >
                                    <select className="px-3 py-2 bg-slate-800 border border-white/10 rounded-lg text-white focus:outline-none focus:border-blue-500/50">
                                        <option value="de">Deutsch</option>
                                        <option value="en">English</option>
                                    </select>
                                </SettingsRow>

                                <SettingsRow
                                    label="Kompakte Ansicht"
                                    description="Weniger Abstände für mehr Inhalt"
                                >
                                    <button className="relative w-14 h-8 rounded-full bg-slate-700 transition-colors">
                                        <div className="absolute top-1 left-1 w-6 h-6 rounded-full bg-white shadow-lg" />
                                    </button>
                                </SettingsRow>
                            </div>
                        </GlassCard>
                    )}

                    {/* Security Tab */}
                    {activeTab === "security" && (
                        <Suspense fallback={<div className="flex items-center justify-center h-64"><Loader2 size={48} className="text-rose-400 animate-spin" /></div>}>
                            <SecurityTab />
                        </Suspense>
                    )}

                    {/* Backup Tab */}
                    {activeTab === "backup" && (
                        <Suspense fallback={<div className="flex items-center justify-center h-64"><Loader2 size={48} className="text-emerald-400 animate-spin" /></div>}>
                            <BackupTab />
                        </Suspense>
                    )}

                    {/* Network Tab */}
                    {activeTab === "network" && (
                        <Suspense fallback={<div className="flex items-center justify-center h-64"><Loader2 size={48} className="text-cyan-400 animate-spin" /></div>}>
                            <NetworkTab />
                        </Suspense>
                    )}

                    {/* Storage Tab */}
                    {activeTab === "storage" && (
                        <Suspense fallback={<div className="flex items-center justify-center h-64"><Loader2 size={48} className="text-blue-400 animate-spin" /></div>}>
                            <StorageTab />
                        </Suspense>
                    )}

                    {/* Logs Tab */}
                    {activeTab === "logs" && (
                        <Suspense fallback={<div className="flex items-center justify-center h-64"><Loader2 size={48} className="text-violet-400 animate-spin" /></div>}>
                            <LogsTab />
                        </Suspense>
                    )}

                    {/* Admin Section */}
                    {activeTab === "admin" && (
                        <AdminTabContent />
                    )}

                    {/* About Section */}
                    {activeTab === "about" && (
                        <div className="space-y-6">
                            <GlassCard>
                                <SectionHeader
                                    icon={Info}
                                    title="Über NAS.AI"
                                    description="Systeminformationen und Version"
                                />

                                <div className="space-y-4">
                                    <div className="flex items-center gap-4 p-4 bg-gradient-to-br from-blue-500/10 to-violet-500/10 rounded-xl border border-blue-500/20">
                                        <div className="w-16 h-16 rounded-2xl bg-gradient-to-br from-blue-500 to-violet-600 flex items-center justify-center shadow-lg">
                                            <Server size={32} className="text-white" />
                                        </div>
                                        <div>
                                            <h3 className="text-2xl font-bold text-white">NAS.AI</h3>
                                            <p className="text-slate-400">Version 1.0.0</p>
                                        </div>
                                    </div>

                                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 mt-6">
                                        <div className="p-4 bg-slate-800/50 rounded-xl border border-white/5">
                                            <div className="flex items-center gap-2 text-slate-400 text-sm mb-1">
                                                <Monitor size={14} />
                                                <span>Frontend</span>
                                            </div>
                                            <p className="text-white font-medium">React + Vite</p>
                                        </div>

                                        <div className="p-4 bg-slate-800/50 rounded-xl border border-white/5">
                                            <div className="flex items-center gap-2 text-slate-400 text-sm mb-1">
                                                <Server size={14} />
                                                <span>Backend</span>
                                            </div>
                                            <p className="text-white font-medium">Go API</p>
                                        </div>
                                    </div>
                                </div>
                            </GlassCard>

                            <GlassCard>
                                <SectionHeader
                                    icon={Cpu}
                                    title="System Status"
                                    description="Live-Informationen aus dem Backend"
                                />

                                {systemLoading ? (
                                    <div className="flex items-center justify-center py-8">
                                        <Loader2 size={32} className="text-blue-400 animate-spin" />
                                    </div>
                                ) : systemInfo ? (
                                    <div className="space-y-3">
                                        <div className="flex items-center justify-between p-3 bg-slate-800/30 rounded-lg">
                                            <span className="text-slate-400">Status</span>
                                            <span className="flex items-center gap-2 text-emerald-400 font-medium">
                                                <div className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" />
                                                Online
                                            </span>
                                        </div>
                                        <div className="flex items-center justify-between p-3 bg-slate-800/30 rounded-lg">
                                            <span className="text-slate-400">Uptime</span>
                                            <span className="text-white font-medium">
                                                {systemInfo.uptime || "N/A"}
                                            </span>
                                        </div>
                                        <div className="flex items-center justify-between p-3 bg-slate-800/30 rounded-lg">
                                            <span className="text-slate-400">Hostname</span>
                                            <span className="text-white font-medium font-mono">
                                                {systemInfo.hostname || "nas.local"}
                                            </span>
                                        </div>
                                    </div>
                                ) : (
                                    <div className="text-center py-8 text-slate-500">
                                        <p>Keine Systemdaten verfügbar</p>
                                        <button
                                            onClick={loadSystemInfo}
                                            className="mt-2 text-blue-400 hover:underline"
                                        >
                                            Erneut versuchen
                                        </button>
                                    </div>
                                )}
                            </GlassCard>
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
}

// Admin Tab Content Component
function AdminTabContent() {
    const toast = useToast();
    const [loading, setLoading] = useState(true);
    const [settings, setSettings] = useState({
        rate_limit_per_min: 100,
        maintenance_mode: false,
        session_timeout_mins: 60,
        cors_origins: []
    });
    const [systemStatus, setSystemStatus] = useState(null);
    const [users, setUsers] = useState([]);

    useEffect(() => {
        loadAdminData();
    }, []);

    const loadAdminData = async () => {
        setLoading(true);
        try {
            const [settingsData, statusData, usersData] = await Promise.all([
                apiRequest("/api/v1/admin/settings", { method: "GET" }),
                apiRequest("/api/v1/admin/status", { method: "GET" }),
                apiRequest("/api/v1/admin/users", { method: "GET" })
            ]);
            setSettings(settingsData);
            setSystemStatus(statusData);
            setUsers(usersData.users || []);
        } catch (err) {
            toast.error("Admin-Daten konnten nicht geladen werden");
        } finally {
            setLoading(false);
        }
    };

    const toggleMaintenanceMode = async () => {
        try {
            const result = await apiRequest("/api/v1/admin/maintenance", {
                method: "POST",
                body: JSON.stringify({ enabled: !settings.maintenance_mode })
            });
            setSettings({ ...settings, maintenance_mode: result.maintenance_mode });
            toast.success(result.maintenance_mode ? "Wartungsmodus aktiviert" : "Wartungsmodus deaktiviert");
        } catch (err) {
            toast.error("Fehler beim Umschalten");
        }
    };

    const updateUserRole = async (userId, newRole) => {
        try {
            await apiRequest(`/api/v1/admin/users/${userId}/role`, {
                method: "PUT",
                body: JSON.stringify({ role: newRole })
            });
            setUsers(users.map(u => u.id === userId ? { ...u, role: newRole } : u));
            toast.success("Rolle aktualisiert");
        } catch (err) {
            toast.error("Fehler beim Aktualisieren");
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
            {/* System Status */}
            <GlassCard>
                <SectionHeader icon={Activity} title="System Status" description="Live-Systemmetriken" />
                {systemStatus && (
                    <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                        <div className="p-3 bg-slate-800/30 rounded-lg">
                            <p className="text-slate-400 text-sm">Uptime</p>
                            <p className="text-white font-mono">{systemStatus.uptime}</p>
                        </div>
                        <div className="p-3 bg-slate-800/30 rounded-lg">
                            <p className="text-slate-400 text-sm">Go Version</p>
                            <p className="text-white font-mono">{systemStatus.go_version}</p>
                        </div>
                        <div className="p-3 bg-slate-800/30 rounded-lg">
                            <p className="text-slate-400 text-sm">Goroutines</p>
                            <p className="text-white font-mono">{systemStatus.num_goroutines}</p>
                        </div>
                        <div className="p-3 bg-slate-800/30 rounded-lg">
                            <p className="text-slate-400 text-sm">Memory</p>
                            <p className="text-white font-mono">{systemStatus.memory_alloc_mb?.toFixed(2)} MB</p>
                        </div>
                    </div>
                )}
            </GlassCard>

            {/* Maintenance Mode */}
            <GlassCard>
                <SectionHeader icon={AlertTriangle} title="Wartungsmodus" description="System für Wartungsarbeiten sperren" />
                <div className="flex items-center justify-between p-4 bg-slate-800/30 rounded-xl">
                    <div className="flex items-center gap-3">
                        {settings.maintenance_mode ? (
                            <PowerOff size={24} className="text-amber-400" />
                        ) : (
                            <Power size={24} className="text-emerald-400" />
                        )}
                        <div>
                            <p className="text-white font-medium">
                                {settings.maintenance_mode ? "Wartungsmodus AKTIV" : "System online"}
                            </p>
                        </div>
                    </div>
                    <button
                        onClick={toggleMaintenanceMode}
                        className={`px-4 py-2 rounded-xl font-medium transition-all ${settings.maintenance_mode
                            ? "bg-emerald-500/20 text-emerald-400 border border-emerald-500/30"
                            : "bg-amber-500/20 text-amber-400 border border-amber-500/30"
                            }`}
                    >
                        {settings.maintenance_mode ? "Deaktivieren" : "Aktivieren"}
                    </button>
                </div>
            </GlassCard>

            {/* User Management */}
            <GlassCard>
                <SectionHeader icon={Users} title="Benutzerverwaltung" description="Benutzer und Rollen verwalten" />
                <div className="overflow-x-auto">
                    <table className="w-full">
                        <thead>
                            <tr className="border-b border-white/10">
                                <th className="text-left py-3 px-4 text-slate-400 font-medium">Benutzer</th>
                                <th className="text-left py-3 px-4 text-slate-400 font-medium">E-Mail</th>
                                <th className="text-left py-3 px-4 text-slate-400 font-medium">Rolle</th>
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

            {/* DB Pool Status */}
            {systemStatus?.db_pool && (
                <GlassCard>
                    <SectionHeader icon={Database} title="Database Pool" description="Verbindungsstatus" />
                    <div className="grid grid-cols-3 gap-4">
                        <div className="p-3 bg-emerald-500/10 border border-emerald-500/20 rounded-lg">
                            <p className="text-slate-400 text-sm">Connections</p>
                            <p className="text-white font-mono text-xl">{systemStatus.db_pool.open_connections}</p>
                        </div>
                        <div className="p-3 bg-blue-500/10 border border-blue-500/20 rounded-lg">
                            <p className="text-slate-400 text-sm">In Use</p>
                            <p className="text-white font-mono text-xl">{systemStatus.db_pool.in_use}</p>
                        </div>
                        <div className="p-3 bg-slate-500/10 border border-slate-500/20 rounded-lg">
                            <p className="text-slate-400 text-sm">Idle</p>
                            <p className="text-white font-mono text-xl">{systemStatus.db_pool.idle}</p>
                        </div>
                    </div>
                </GlassCard>
            )}
        </div>
    );
}

// AI Settings Tab Component
function AISettingsTab() {
    const toast = useToast();
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [testingConnection, setTestingConnection] = useState(false);
    const [reindexing, setReindexing] = useState(false);

    const [settings, setSettings] = useState({
        llm_model: "llama3.2",
        classifier_model: "llama3.2:1b",
        embedding_model: "mxbai-embed-large",
        temperature: 0.7,
        max_tokens: 500,
        context_documents: 10,
        auto_index: true,
        index_paths: ["/mnt/data"],
        index_interval: 30,
        ollama_url: "http://host.docker.internal:11434"
    });

    const [ollamaStatus, setOllamaStatus] = useState({
        connected: false,
        models: []
    });

    const [indexStats, setIndexStats] = useState({
        total_files: 0,
        indexed_files: 0,
        last_index: null
    });

    useEffect(() => {
        loadAISettings();
    }, []);

    const loadAISettings = async () => {
        setLoading(true);
        try {
            // Get AI settings from backend
            const settingsData = await apiRequest("/api/v1/ai/settings", { method: "GET" });
            if (settingsData) {
                setSettings(prev => ({ ...prev, ...settingsData }));
            }
        } catch (err) {
            console.log("Using default AI settings");
        }

        // Get comprehensive status from AI agent (includes Ollama + index stats)
        await loadAIStatus();

        setLoading(false);
    };

    const loadAIStatus = async () => {
        setTestingConnection(true);
        try {
            const data = await apiRequest("/api/v1/ai/status", { method: "GET" });
            if (data) {
                // Update Ollama status
                if (data.ollama) {
                    setOllamaStatus({
                        connected: data.ollama.connected,
                        models: data.ollama.models || []
                    });
                }
                // Update index stats
                if (data.index) {
                    setIndexStats({
                        total_files: data.index.total_files || 0,
                        indexed_files: data.index.indexed_files || 0,
                        last_index: null
                    });
                }
            }
        } catch (err) {
            console.error("Failed to load AI status:", err);
            setOllamaStatus({ connected: false, models: [] });
        }
        setTestingConnection(false);
    };

    const testOllamaConnection = async () => {
        // Use the backend proxy to test Ollama connection
        await loadAIStatus();
    };

    const loadIndexStats = async () => {
        // Already loaded via loadAIStatus
    };

    const handleSave = async () => {
        setSaving(true);
        try {
            await apiRequest("/api/v1/ai/settings", {
                method: "POST",
                body: JSON.stringify(settings)
            });
            toast.success("AI-Einstellungen gespeichert");
        } catch (err) {
            toast.error("Fehler beim Speichern");
        } finally {
            setSaving(false);
        }
    };

    const handleReindex = async () => {
        setReindexing(true);
        try {
            await apiRequest("/api/v1/ai/reindex", { method: "POST" });
            toast.success("Re-Indexierung gestartet");
            // Reload stats after a delay
            setTimeout(loadIndexStats, 3000);
        } catch (err) {
            toast.error("Fehler beim Starten der Indexierung");
        } finally {
            setReindexing(false);
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
            {/* Model Selection */}
            <GlassCard>
                <SectionHeader
                    icon={Brain}
                    title="Modell-Auswahl"
                    description="Wähle die AI-Modelle für verschiedene Aufgaben"
                />

                <div className="space-y-4">
                    {/* LLM Model */}
                    <div>
                        <label className="block text-sm font-medium text-slate-300 mb-2">
                            Antwort-Modell (RAG)
                        </label>
                        <select
                            value={settings.llm_model}
                            onChange={(e) => setSettings({ ...settings, llm_model: e.target.value })}
                            className="w-full px-4 py-3 bg-slate-800/50 border border-white/10 rounded-xl text-white focus:border-violet-500/50 focus:outline-none"
                        >
                            <option value="llama3.2">Llama 3.2 (Standard)</option>
                            <option value="qwen2.5:3b">Qwen 2.5 3B</option>
                            <option value="mistral">Mistral 7B</option>
                        </select>
                        <p className="text-xs text-slate-500 mt-1">Für AI-Antworten und Zusammenfassungen</p>
                    </div>

                    {/* Classifier Model */}
                    <div>
                        <label className="block text-sm font-medium text-slate-300 mb-2">
                            Klassifikator-Modell
                        </label>
                        <select
                            value={settings.classifier_model}
                            onChange={(e) => setSettings({ ...settings, classifier_model: e.target.value })}
                            className="w-full px-4 py-3 bg-slate-800/50 border border-white/10 rounded-xl text-white focus:border-violet-500/50 focus:outline-none"
                        >
                            <option value="llama3.2:1b">Llama 3.2 1B (Schnell)</option>
                            <option value="heuristic">Nur Heuristik (Kein LLM)</option>
                        </select>
                        <p className="text-xs text-slate-500 mt-1">Für Intent-Erkennung (Suche vs. Frage)</p>
                    </div>

                    {/* Embedding Model */}
                    <div>
                        <label className="block text-sm font-medium text-slate-300 mb-2">
                            Embedding-Modell
                        </label>
                        <select
                            value={settings.embedding_model}
                            onChange={(e) => setSettings({ ...settings, embedding_model: e.target.value })}
                            className="w-full px-4 py-3 bg-slate-800/50 border border-white/10 rounded-xl text-white focus:border-violet-500/50 focus:outline-none"
                        >
                            <option value="mxbai-embed-large">mxbai-embed-large (Standard)</option>
                            <option value="nomic-embed-text">nomic-embed-text</option>
                        </select>
                        <p className="text-xs text-slate-500 mt-1">Für semantische Suche und Vektoren</p>
                    </div>
                </div>
            </GlassCard>

            {/* Response Settings */}
            <GlassCard>
                <SectionHeader
                    icon={Thermometer}
                    title="Antwort-Einstellungen"
                    description="Steuere wie die AI antwortet"
                />

                <div className="space-y-6">
                    {/* Temperature */}
                    <div>
                        <div className="flex justify-between items-center mb-2">
                            <label className="text-sm font-medium text-slate-300">
                                Kreativität (Temperature)
                            </label>
                            <span className="text-violet-400 font-mono">{settings.temperature.toFixed(1)}</span>
                        </div>
                        <input
                            type="range"
                            min="0"
                            max="1"
                            step="0.1"
                            value={settings.temperature}
                            onChange={(e) => setSettings({ ...settings, temperature: parseFloat(e.target.value) })}
                            className="w-full h-2 bg-slate-700 rounded-lg appearance-none cursor-pointer accent-violet-500"
                        />
                        <div className="flex justify-between text-xs text-slate-500 mt-1">
                            <span>Präzise</span>
                            <span>Kreativ</span>
                        </div>
                    </div>

                    {/* Max Tokens */}
                    <div>
                        <div className="flex justify-between items-center mb-2">
                            <label className="text-sm font-medium text-slate-300">
                                Maximale Antwortlänge
                            </label>
                            <span className="text-violet-400 font-mono">{settings.max_tokens}</span>
                        </div>
                        <input
                            type="range"
                            min="100"
                            max="2000"
                            step="100"
                            value={settings.max_tokens}
                            onChange={(e) => setSettings({ ...settings, max_tokens: parseInt(e.target.value) })}
                            className="w-full h-2 bg-slate-700 rounded-lg appearance-none cursor-pointer accent-violet-500"
                        />
                        <div className="flex justify-between text-xs text-slate-500 mt-1">
                            <span>Kurz (100)</span>
                            <span>Lang (2000)</span>
                        </div>
                    </div>

                    {/* Context Documents */}
                    <div>
                        <div className="flex justify-between items-center mb-2">
                            <label className="text-sm font-medium text-slate-300">
                                Kontext-Dokumente (RAG)
                            </label>
                            <span className="text-violet-400 font-mono">{settings.context_documents}</span>
                        </div>
                        <input
                            type="range"
                            min="3"
                            max="20"
                            step="1"
                            value={settings.context_documents}
                            onChange={(e) => setSettings({ ...settings, context_documents: parseInt(e.target.value) })}
                            className="w-full h-2 bg-slate-700 rounded-lg appearance-none cursor-pointer accent-violet-500"
                        />
                        <p className="text-xs text-slate-500 mt-1">Anzahl der Dokumente für RAG-Kontext</p>
                    </div>
                </div>
            </GlassCard>

            {/* Indexing Settings */}
            <GlassCard>
                <SectionHeader
                    icon={FolderSync}
                    title="Indexierung"
                    description="Automatische Datei-Indexierung steuern"
                />

                <div className="space-y-4">
                    {/* Auto Index Toggle */}
                    <SettingsRow
                        label="Auto-Indexierung"
                        description="Neue Dateien automatisch indexieren"
                    >
                        <button
                            onClick={() => setSettings({ ...settings, auto_index: !settings.auto_index })}
                            className={`relative w-14 h-8 rounded-full transition-colors ${settings.auto_index ? "bg-violet-500/30" : "bg-slate-700"
                                }`}
                        >
                            <div className={`absolute top-1 w-6 h-6 rounded-full bg-white shadow-lg transition-all ${settings.auto_index ? "left-7" : "left-1"
                                }`} />
                        </button>
                    </SettingsRow>

                    {/* Index Paths Configuration */}
                    {settings.auto_index && (
                        <div className="mt-4 p-4 bg-slate-800/30 rounded-xl border border-white/5">
                            <label className="block text-sm font-medium text-slate-300 mb-3">
                                Zu indexierende Ordner
                            </label>
                            <div className="space-y-2">
                                {(settings.index_paths || []).map((path, idx) => (
                                    <div key={idx} className="flex gap-2">
                                        <input
                                            type="text"
                                            value={path}
                                            onChange={(e) => {
                                                const newPaths = [...settings.index_paths];
                                                newPaths[idx] = e.target.value;
                                                setSettings({ ...settings, index_paths: newPaths });
                                            }}
                                            className="flex-1 px-3 py-2 bg-slate-900/50 border border-white/10 rounded-lg text-white font-mono text-sm focus:border-violet-500/50 focus:outline-none"
                                            placeholder="/mnt/data"
                                        />
                                        <button
                                            onClick={() => {
                                                const newPaths = settings.index_paths.filter((_, i) => i !== idx);
                                                setSettings({ ...settings, index_paths: newPaths.length > 0 ? newPaths : ["/mnt/data"] });
                                            }}
                                            className="px-3 py-2 bg-rose-500/20 hover:bg-rose-500/30 text-rose-400 rounded-lg border border-rose-500/30 transition-all"
                                            disabled={settings.index_paths.length <= 1}
                                        >
                                            ×
                                        </button>
                                    </div>
                                ))}
                                <button
                                    onClick={() => setSettings({ ...settings, index_paths: [...(settings.index_paths || []), ""] })}
                                    className="flex items-center gap-2 px-3 py-2 bg-violet-500/20 hover:bg-violet-500/30 text-violet-400 rounded-lg border border-violet-500/30 transition-all text-sm"
                                >
                                    + Ordner hinzufügen
                                </button>
                            </div>
                            <p className="text-xs text-slate-500 mt-2">Nur Dateien in diesen Ordnern werden automatisch indexiert</p>
                        </div>
                    )}

                    {/* Index Stats */}
                    <div className="grid grid-cols-3 gap-4 mt-4">
                        <div className="p-3 bg-slate-800/30 rounded-lg">
                            <p className="text-slate-400 text-xs">Dateien gesamt</p>
                            <p className="text-white font-mono text-lg">{indexStats.total_files || 0}</p>
                        </div>
                        <div className="p-3 bg-violet-500/10 border border-violet-500/20 rounded-lg">
                            <p className="text-slate-400 text-xs">Indexiert</p>
                            <p className="text-white font-mono text-lg">{indexStats.indexed_files || 0}</p>
                        </div>
                        <div className="p-3 bg-slate-800/30 rounded-lg">
                            <p className="text-slate-400 text-xs">Letzte Indexierung</p>
                            <p className="text-white font-mono text-sm">{indexStats.last_index || "—"}</p>
                        </div>
                    </div>

                    {/* Reindex Button */}
                    <button
                        onClick={handleReindex}
                        disabled={reindexing}
                        className="flex items-center gap-2 px-4 py-2.5 bg-amber-500/20 hover:bg-amber-500/30 text-amber-400 rounded-xl border border-amber-500/30 transition-all disabled:opacity-50 mt-4"
                    >
                        {reindexing ? (
                            <>
                                <Loader2 size={18} className="animate-spin" />
                                <span className="font-medium">Indexiere...</span>
                            </>
                        ) : (
                            <>
                                <RefreshCw size={18} />
                                <span className="font-medium">Alle Dateien neu indexieren</span>
                            </>
                        )}
                    </button>
                </div>
            </GlassCard>

            {/* Ollama Connection */}
            <GlassCard>
                <SectionHeader
                    icon={Server}
                    title="Ollama Verbindung"
                    description="LLM-Server Konfiguration"
                />

                <div className="space-y-4">
                    {/* Ollama URL */}
                    <div>
                        <label className="block text-sm font-medium text-slate-300 mb-2">
                            Ollama URL
                        </label>
                        <div className="flex gap-2">
                            <input
                                type="text"
                                value={settings.ollama_url}
                                onChange={(e) => setSettings({ ...settings, ollama_url: e.target.value })}
                                className="flex-1 px-4 py-3 bg-slate-800/50 border border-white/10 rounded-xl text-white font-mono focus:border-violet-500/50 focus:outline-none"
                                placeholder="http://localhost:11434"
                            />
                            <button
                                onClick={testOllamaConnection}
                                disabled={testingConnection}
                                className="px-4 py-3 bg-blue-500/20 hover:bg-blue-500/30 text-blue-400 rounded-xl border border-blue-500/30 transition-all disabled:opacity-50"
                            >
                                {testingConnection ? (
                                    <Loader2 size={18} className="animate-spin" />
                                ) : (
                                    <TestTube size={18} />
                                )}
                            </button>
                        </div>
                    </div>

                    {/* Connection Status */}
                    <div className={`flex items-center gap-3 p-4 rounded-xl ${ollamaStatus.connected
                        ? "bg-emerald-500/10 border border-emerald-500/20"
                        : "bg-rose-500/10 border border-rose-500/20"
                        }`}>
                        {ollamaStatus.connected ? (
                            <>
                                <CircleCheck size={24} className="text-emerald-400" />
                                <div>
                                    <p className="text-emerald-400 font-medium">Verbunden</p>
                                    <p className="text-slate-400 text-sm">
                                        {ollamaStatus.models.length} Modelle verfügbar
                                    </p>
                                </div>
                            </>
                        ) : (
                            <>
                                <CircleX size={24} className="text-rose-400" />
                                <div>
                                    <p className="text-rose-400 font-medium">Nicht verbunden</p>
                                    <p className="text-slate-400 text-sm">Prüfe die Ollama URL</p>
                                </div>
                            </>
                        )}
                    </div>

                    {/* Available Models */}
                    {ollamaStatus.connected && ollamaStatus.models.length > 0 && (
                        <div>
                            <p className="text-sm font-medium text-slate-300 mb-2">Verfügbare Modelle</p>
                            <div className="flex flex-wrap gap-2">
                                {ollamaStatus.models.map((model, idx) => (
                                    <span
                                        key={idx}
                                        className="px-3 py-1 bg-slate-800/50 border border-white/10 rounded-lg text-sm text-slate-300 font-mono"
                                    >
                                        {model}
                                    </span>
                                ))}
                            </div>
                        </div>
                    )}
                </div>
            </GlassCard>

            {/* Save Button */}
            <button
                onClick={handleSave}
                disabled={saving}
                className="flex items-center gap-2 px-6 py-3 bg-gradient-to-r from-violet-600 to-blue-600 hover:from-violet-500 hover:to-blue-500 text-white rounded-xl shadow-lg shadow-violet-500/20 transition-all disabled:opacity-50"
            >
                {saving ? (
                    <>
                        <Loader2 size={20} className="animate-spin" />
                        <span className="font-medium">Speichern...</span>
                    </>
                ) : (
                    <>
                        <Save size={20} />
                        <span className="font-medium">Einstellungen speichern</span>
                    </>
                )}
            </button>
        </div>
    );
}

// Crypto Settings Tab Component
function CryptoSettingsTab() {
    const toast = useToast();
    const [loading, setLoading] = useState(true);
    const [vaultStatus, setVaultStatus] = useState(null);
    const [vaultPath, setVaultPath] = useState("");
    const [saving, setSaving] = useState(false);
    const [unlocking, setUnlocking] = useState(false);
    const [masterPassword, setMasterPassword] = useState("");

    const API_BASE = import.meta.env.VITE_API_BASE_URL || window.location.origin;

    useEffect(() => {
        loadVaultStatus();
    }, []);

    const loadVaultStatus = async () => {
        setLoading(true);
        try {
            const res = await fetch(`${API_BASE}/api/v1/vault/status`, {
                credentials: "include",
            });
            if (res.ok) {
                const data = await res.json();
                setVaultStatus(data);
                setVaultPath(data.vaultPath || "");
            }
        } catch (err) {
            console.error("Failed to load vault status:", err);
        } finally {
            setLoading(false);
        }
    };

    const handleSavePath = async () => {
        setSaving(true);
        try {
            const { data, error } = await apiRequest("/api/v1/system/vault/config", {
                method: "PUT",
                body: { vaultPath },
            });
            if (error) throw new Error(error);
            toast.success("Vault-Pfad gespeichert");
            loadVaultStatus();
        } catch (err) {
            toast.error(err.message || "Fehler beim Speichern");
        } finally {
            setSaving(false);
        }
    };

    const handleLock = async () => {
        try {
            const { data, error } = await apiRequest("/api/v1/system/vault/lock", {
                method: "POST",
            });
            if (error) throw new Error(error);
            toast.success("Vault gesperrt");
            loadVaultStatus();
        } catch (err) {
            toast.error(err.message || "Fehler beim Sperren");
        }
    };

    const handleUnlock = async () => {
        if (!masterPassword) {
            toast.error("Master-Passwort erforderlich");
            return;
        }
        setUnlocking(true);
        try {
            const { data, error } = await apiRequest("/api/v1/system/vault/unlock", {
                method: "POST",
                body: { masterPassword },
            });
            if (error) throw new Error(error);
            toast.success("Vault entsperrt");
            setMasterPassword("");
            loadVaultStatus();
        } catch (err) {
            toast.error(err.message || "Falsches Passwort");
        } finally {
            setUnlocking(false);
        }
    };

    if (loading) {
        return (
            <div className="flex items-center justify-center h-64">
                <Loader2 size={48} className="text-amber-400 animate-spin" />
            </div>
        );
    }

    return (
        <div className="space-y-6">
            {/* Vault Status */}
            <GlassCard>
                <SectionHeader
                    icon={KeyRound}
                    title="Vault Status"
                    description="Zero-Knowledge Verschlüsselung für Ihre Daten"
                />

                <div className="flex items-center justify-between p-4 bg-slate-800/30 rounded-xl mb-4">
                    <div className="flex items-center gap-3">
                        {vaultStatus?.locked ? (
                            <Lock size={24} className="text-amber-400" />
                        ) : (
                            <Unlock size={24} className="text-emerald-400" />
                        )}
                        <div>
                            <p className="text-white font-medium">
                                {vaultStatus?.locked ? "Vault GESPERRT" : "Vault ENTSPERRT"}
                            </p>
                            <p className="text-slate-400 text-sm">
                                {vaultStatus?.configured ? "Konfiguriert" : "Nicht eingerichtet"}
                            </p>
                        </div>
                    </div>
                    {vaultStatus?.configured && (
                        <button
                            onClick={vaultStatus?.locked ? null : handleLock}
                            disabled={vaultStatus?.locked}
                            className={`px-4 py-2 rounded-xl font-medium transition-all ${vaultStatus?.locked
                                ? "bg-slate-500/20 text-slate-400 cursor-not-allowed"
                                : "bg-amber-500/20 text-amber-400 border border-amber-500/30 hover:bg-amber-500/30"
                                }`}
                        >
                            {vaultStatus?.locked ? "Gesperrt" : "Sperren"}
                        </button>
                    )}
                </div>

                {/* Unlock Form */}
                {vaultStatus?.locked && vaultStatus?.configured && (
                    <div className="p-4 bg-slate-800/20 rounded-xl border border-white/5">
                        <label className="block text-sm font-medium text-slate-300 mb-2">
                            Master-Passwort zum Entsperren
                        </label>
                        <div className="flex gap-3">
                            <div className="relative flex-1">
                                <KeyRound size={18} className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-500" />
                                <input
                                    type="password"
                                    value={masterPassword}
                                    onChange={(e) => setMasterPassword(e.target.value)}
                                    placeholder="••••••••••••"
                                    className="w-full pl-10 pr-4 py-3 bg-slate-900/50 border border-white/10 rounded-xl text-white placeholder:text-slate-500 focus:border-amber-500/50 focus:outline-none transition-all"
                                />
                            </div>
                            <button
                                onClick={handleUnlock}
                                disabled={unlocking || !masterPassword}
                                className="flex items-center gap-2 px-6 py-3 bg-amber-500/20 hover:bg-amber-500/30 text-amber-400 rounded-xl font-medium transition-all border border-amber-500/30 disabled:opacity-50"
                            >
                                {unlocking ? (
                                    <Loader2 size={18} className="animate-spin" />
                                ) : (
                                    <Unlock size={18} />
                                )}
                                <span>Entsperren</span>
                            </button>
                        </div>
                    </div>
                )}
            </GlassCard>

            {/* Vault Path Configuration */}
            <GlassCard>
                <SectionHeader
                    icon={FolderOpen}
                    title="Vault-Pfad"
                    description="Speicherort für Vault-Dateien (DEK, Salt, Config)"
                />

                <div className="space-y-4">
                    <div>
                        <label className="block text-sm font-medium text-slate-300 mb-2">
                            Aktueller Pfad
                        </label>
                        <div className="relative">
                            <FolderOpen size={18} className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-500" />
                            <input
                                type="text"
                                value={vaultPath}
                                onChange={(e) => setVaultPath(e.target.value)}
                                placeholder="/var/lib/nas/vault"
                                disabled={!vaultStatus?.locked}
                                className="w-full pl-10 pr-4 py-3 bg-slate-800/50 border border-white/10 rounded-xl text-white focus:border-amber-500/50 focus:outline-none transition-all disabled:opacity-50 disabled:cursor-not-allowed"
                            />
                        </div>
                        {!vaultStatus?.locked && (
                            <p className="text-amber-400 text-sm mt-2 flex items-center gap-1">
                                <AlertCircle size={14} />
                                Vault muss gesperrt sein, um Pfad zu ändern
                            </p>
                        )}
                    </div>

                    <button
                        onClick={handleSavePath}
                        disabled={saving || !vaultStatus?.locked}
                        className="flex items-center gap-2 px-4 py-2.5 bg-blue-500/20 hover:bg-blue-500/30 text-blue-400 rounded-xl border border-blue-500/30 transition-all disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                        {saving ? (
                            <Loader2 size={18} className="animate-spin" />
                        ) : (
                            <Save size={18} />
                        )}
                        <span className="font-medium">Pfad speichern</span>
                    </button>
                </div>
            </GlassCard>

            {/* Encryption Info */}
            <GlassCard>
                <SectionHeader
                    icon={Shield}
                    title="Verschlüsselung"
                    description="Details zur verwendeten Verschlüsselung"
                />

                <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
                    <div className="p-4 bg-slate-800/30 rounded-xl border border-white/5">
                        <p className="text-slate-400 text-sm mb-1">Algorithmus</p>
                        <p className="text-white font-mono font-medium">AES-256-GCM</p>
                    </div>
                    <div className="p-4 bg-slate-800/30 rounded-xl border border-white/5">
                        <p className="text-slate-400 text-sm mb-1">Key Derivation</p>
                        <p className="text-white font-mono font-medium">Argon2id</p>
                    </div>
                    <div className="p-4 bg-slate-800/30 rounded-xl border border-white/5">
                        <p className="text-slate-400 text-sm mb-1">Architektur</p>
                        <p className="text-white font-mono font-medium">Zero-Knowledge</p>
                    </div>
                </div>

                <p className="text-slate-500 text-sm mt-4">
                    🔒 Ihr Master-Passwort verlässt niemals das System. Alle Schlüssel werden nur im RAM gehalten.
                </p>
            </GlassCard>
        </div>
    );
}

