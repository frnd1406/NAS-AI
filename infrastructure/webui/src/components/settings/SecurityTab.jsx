import { useState, useEffect } from "react";
import {
    Shield,
    AlertTriangle,
    Check,
    Loader2,
    Lock,
    Eye,
    Clock,
    Target,
    Zap
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
const SectionHeader = ({ icon: Icon, title, description, iconColor = "rose" }) => (
    <div className="flex items-start gap-4 mb-6">
        <div className={`p-3 rounded-xl bg-${iconColor}-500/20 border border-${iconColor}-500/30`}>
            <Icon size={24} className={`text-${iconColor}-400`} />
        </div>
        <div>
            <h2 className="text-xl font-bold text-white tracking-tight">{title}</h2>
            <p className="text-slate-400 text-sm mt-1">{description}</p>
        </div>
    </div>
);

// Monitor Mode Options (Obfuscated - no "honeyfile" terminology)
const MONITOR_MODES = [
    { value: "audit_strict", label: "Audit Strict (PANIC)", description: "Löst sofortigen Vault-Lock bei Zugriff aus" }
];

export default function SecurityTab() {
    const toast = useToast();
    const [loading, setLoading] = useState(false);
    const [creating, setCreating] = useState(false);
    const [alertsLoading, setAlertsLoading] = useState(true);

    // Form state
    const [formData, setFormData] = useState({
        resource_path: "",
        monitor_mode: "audit_strict",
        retention: "persistent"  // Dummy field for obfuscation
    });

    // Intrusion history (NOT active checkpoints - Stealth Mode!)
    const [alerts, setAlerts] = useState([]);

    useEffect(() => {
        loadSecurityAlerts();
    }, []);

    const loadSecurityAlerts = async () => {
        setAlertsLoading(true);
        try {
            // Get security alerts/events - NOT the checkpoint list
            const response = await apiRequest("/api/v1/admin/audit-logs?type=security&limit=10", {
                method: "GET"
            });

            if (response?.logs) {
                setAlerts(response.logs);
            } else if (Array.isArray(response)) {
                // Filter for security-related events
                setAlerts(response.filter(log =>
                    log.type === "security" ||
                    log.action?.includes("INTEGRITY") ||
                    log.action?.includes("LOCKED")
                ).slice(0, 10));
            }
        } catch (err) {
            // Silently fail - security logs might not be available
            console.log("Security alerts not available");
            setAlerts([]);
        } finally {
            setAlertsLoading(false);
        }
    };

    const handleSubmit = async (e) => {
        e.preventDefault();

        if (!formData.resource_path.trim()) {
            toast.error("Pfad ist erforderlich");
            return;
        }

        // Validate path starts with /
        if (!formData.resource_path.startsWith("/")) {
            toast.error("Pfad muss mit / beginnen");
            return;
        }

        setCreating(true);
        try {
            // POST to stealth endpoint - NO RESPONSE DATA IS STORED
            await apiRequest("/api/v1/sys/integrity/checkpoints", {
                method: "POST",
                body: JSON.stringify({
                    resource_path: formData.resource_path,
                    monitor_mode: formData.monitor_mode,
                    retention: formData.retention
                })
            });

            // Success - show generic message (no details about what was created)
            toast.success("✅ Integritäts-Monitor aktiv");

            // Reset form
            setFormData({
                resource_path: "",
                monitor_mode: "audit_strict",
                retention: "persistent"
            });

        } catch (err) {
            toast.error("Registrierung fehlgeschlagen");
        } finally {
            setCreating(false);
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

    // Encryption Policies State
    const [policies, setPolicies] = useState({
        encrypt_docs: true,
        encrypt_images: true,
        encrypt_videos: false,
        max_size_gb: 2
    });
    const [policiesSaving, setPoliciesSaving] = useState(false);

    // Load policies (mock)
    useEffect(() => {
        // In a real app, this would be an API call
        // const load = async () => { ... }
        // For now, we use defaults
        console.log("Loaded encryption policies (mock)");
    }, []);

    const handlePolicyChange = (key, value) => {
        setPolicies(prev => ({ ...prev, [key]: value }));
    };

    const savePolicies = async () => {
        setPoliciesSaving(true);
        try {
            // Mock API call
            await new Promise(resolve => setTimeout(resolve, 800));
            console.log("Saved policies:", policies);

            // In reality: await apiRequest("/api/v1/admin/encryption-policies", { method: "POST", body: JSON.stringify(policies) });

            toast.success("Encryption Policies gespeichert");
        } catch (err) {
            toast.error("Fehler beim Speichern");
        } finally {
            setPoliciesSaving(false);
        }
    };

    return (
        <div className="space-y-6">
            {/* Smart Encryption Policies */}
            <GlassCard>
                <SectionHeader
                    icon={Shield}
                    title="Smart Encryption Policies"
                    description="Globale Verschlüsselungs-Regeln für alle Uploads"
                    iconColor="blue"
                />

                <div className="space-y-6">
                    <div className="space-y-4">
                        <div className="flex items-center justify-between p-4 bg-slate-800/30 rounded-xl border border-white/5">
                            <div>
                                <p className="text-white font-medium">Dokumente verschlüsseln</p>
                                <p className="text-slate-400 text-sm">PDF, DOCX, TXT, etc.</p>
                            </div>
                            <button
                                onClick={() => handlePolicyChange('encrypt_docs', !policies.encrypt_docs)}
                                className={`relative w-12 h-7 rounded-full transition-colors ${policies.encrypt_docs ? "bg-blue-500" : "bg-slate-700"}`}
                            >
                                <div className={`absolute top-1 left-1 w-5 h-5 rounded-full bg-white shadow-lg transition-transform ${policies.encrypt_docs ? "translate-x-5" : "translate-x-0"}`} />
                            </button>
                        </div>

                        <div className="flex items-center justify-between p-4 bg-slate-800/30 rounded-xl border border-white/5">
                            <div>
                                <p className="text-white font-medium">Bilder verschlüsseln</p>
                                <p className="text-slate-400 text-sm">JPG, PNG, WEBP (Vorschauen bleiben generierbar)</p>
                            </div>
                            <button
                                onClick={() => handlePolicyChange('encrypt_images', !policies.encrypt_images)}
                                className={`relative w-12 h-7 rounded-full transition-colors ${policies.encrypt_images ? "bg-blue-500" : "bg-slate-700"}`}
                            >
                                <div className={`absolute top-1 left-1 w-5 h-5 rounded-full bg-white shadow-lg transition-transform ${policies.encrypt_images ? "translate-x-5" : "translate-x-0"}`} />
                            </button>
                        </div>

                        <div className="flex items-center justify-between p-4 bg-slate-800/30 rounded-xl border border-white/5">
                            <div>
                                <div className="flex items-center gap-2">
                                    <p className="text-white font-medium">Videos verschlüsseln</p>
                                    {!policies.encrypt_videos && <span className="px-2 py-0.5 rounded text-[10px] uppercase font-bold bg-emerald-500/20 text-emerald-400">Performance</span>}
                                </div>
                                <p className="text-slate-400 text-sm">MP4, MOV, MKV (Streaming limitiert bei Encryption)</p>
                            </div>
                            <button
                                onClick={() => handlePolicyChange('encrypt_videos', !policies.encrypt_videos)}
                                className={`relative w-12 h-7 rounded-full transition-colors ${policies.encrypt_videos ? "bg-blue-500" : "bg-slate-700"}`}
                            >
                                <div className={`absolute top-1 left-1 w-5 h-5 rounded-full bg-white shadow-lg transition-transform ${policies.encrypt_videos ? "translate-x-5" : "translate-x-0"}`} />
                            </button>
                        </div>
                    </div>

                    <div className="p-4 bg-slate-800/30 rounded-xl border border-white/5">
                        <div className="flex justify-between mb-2">
                            <label className="text-white font-medium">Max File Size Limit</label>
                            <span className="text-blue-400 font-mono">{policies.max_size_gb} GB</span>
                        </div>
                        <input
                            type="range"
                            min="0.5"
                            max="10"
                            step="0.5"
                            value={policies.max_size_gb}
                            onChange={(e) => handlePolicyChange('max_size_gb', parseFloat(e.target.value))}
                            className="w-full h-2 bg-slate-700 rounded-lg appearance-none cursor-pointer accent-blue-500"
                        />
                        <p className="text-slate-500 text-xs mt-2">
                            Dateien größer als {policies.max_size_gb}GB werden automatisch unverschlüsselt (Standard) gespeichert, um Performance-Probleme zu vermeiden.
                        </p>
                    </div>

                    <div className="flex justify-end">
                        <button
                            onClick={savePolicies}
                            disabled={policiesSaving}
                            className="flex items-center gap-2 px-5 py-2.5 bg-blue-500 hover:bg-blue-600 text-white rounded-xl shadow-lg shadow-blue-500/20 transition-all active:scale-95 disabled:opacity-50 disabled:cursor-not-allowed"
                        >
                            {policiesSaving ? <Loader2 size={18} className="animate-spin" /> : <Check size={18} />}
                            <span>Policies speichern</span>
                        </button>
                    </div>
                </div>
            </GlassCard>
            {/* Info Banner */}
            <div className="p-4 bg-amber-500/10 border border-amber-500/30 rounded-xl flex items-start gap-3">
                <AlertTriangle size={20} className="text-amber-400 mt-0.5" />
                <div>
                    <p className="text-amber-400 font-medium">Stealth Mode aktiv</p>
                    <p className="text-slate-400 text-sm mt-1">
                        Registrierte Pfade werden aus Sicherheitsgründen nicht angezeigt.
                        Zugriff auf überwachte Ressourcen löst sofortigen Vault-Lock aus.
                    </p>
                </div>
            </div>

            {/* Integrity Checkpoint Form (Blind Write) */}
            <GlassCard>
                <SectionHeader
                    icon={Target}
                    title="Integritäts-Checkpoint"
                    description="Ressource zur Überwachung registrieren"
                    iconColor="rose"
                />

                <form onSubmit={handleSubmit} className="space-y-4">
                    <div>
                        <label className="block text-sm font-medium text-slate-300 mb-2">
                            Ressourcen-Pfad
                        </label>
                        <div className="relative">
                            <Lock size={18} className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-500" />
                            <input
                                type="text"
                                value={formData.resource_path}
                                onChange={(e) => setFormData({ ...formData, resource_path: e.target.value })}
                                placeholder="/mnt/data/passwords.txt"
                                className="w-full pl-10 pr-4 py-3 bg-slate-800/50 border border-white/10 rounded-xl text-white placeholder:text-slate-500 focus:border-rose-500/50 focus:outline-none transition-all font-mono"
                            />
                        </div>
                        <p className="text-slate-500 text-sm mt-1">
                            Absoluter Pfad zur Datei oder zum Ordner
                        </p>
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-slate-300 mb-2">
                            Monitor-Modus
                        </label>
                        <select
                            value={formData.monitor_mode}
                            onChange={(e) => setFormData({ ...formData, monitor_mode: e.target.value })}
                            className="w-full px-4 py-3 bg-slate-800/50 border border-white/10 rounded-xl text-white focus:border-rose-500/50 focus:outline-none transition-all"
                        >
                            {MONITOR_MODES.map((mode) => (
                                <option key={mode.value} value={mode.value}>
                                    {mode.label}
                                </option>
                            ))}
                        </select>
                        <p className="text-rose-400/70 text-sm mt-1 flex items-center gap-1">
                            <Zap size={12} />
                            {MONITOR_MODES.find(m => m.value === formData.monitor_mode)?.description}
                        </p>
                    </div>

                    <button
                        type="submit"
                        disabled={creating || !formData.resource_path.trim()}
                        className="flex items-center gap-2 px-6 py-3 bg-rose-500/20 hover:bg-rose-500/30 text-rose-400 rounded-xl border border-rose-500/30 transition-all disabled:opacity-50 disabled:cursor-not-allowed font-medium"
                    >
                        {creating ? (
                            <Loader2 size={18} className="animate-spin" />
                        ) : (
                            <Shield size={18} />
                        )}
                        <span>Scharfschalten</span>
                    </button>
                </form>
            </GlassCard>

            {/* Security Info */}
            <GlassCard>
                <SectionHeader
                    icon={Shield}
                    title="Sicherheitsmechanismen"
                    description="Aktive Schutzfunktionen"
                    iconColor="emerald"
                />

                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                    <div className="p-4 bg-emerald-500/10 border border-emerald-500/20 rounded-xl">
                        <div className="flex items-center gap-2 mb-2">
                            <Check size={16} className="text-emerald-400" />
                            <span className="text-emerald-400 font-medium">ZIP-Sicherheit</span>
                        </div>
                        <ul className="text-slate-400 text-sm space-y-1">
                            <li>• Magic Bytes Validierung</li>
                            <li>• Anti-Zip-Bomb (100:1 Ratio)</li>
                            <li>• Path Traversal Schutz</li>
                            <li>• 2GB Größenlimit</li>
                        </ul>
                    </div>

                    <div className="p-4 bg-emerald-500/10 border border-emerald-500/20 rounded-xl">
                        <div className="flex items-center gap-2 mb-2">
                            <Check size={16} className="text-emerald-400" />
                            <span className="text-emerald-400 font-medium">Vault-Verschlüsselung</span>
                        </div>
                        <ul className="text-slate-400 text-sm space-y-1">
                            <li>• AES-256-GCM</li>
                            <li>• Argon2id Key Derivation</li>
                            <li>• Zero-Knowledge Architektur</li>
                            <li>• Automatischer Lock bei Intrusion</li>
                        </ul>
                    </div>
                </div>
            </GlassCard>

            {/* Intrusion History (Past Events, NOT active traps) */}
            <GlassCard>
                <SectionHeader
                    icon={AlertTriangle}
                    title="Sicherheitsereignisse"
                    description="Vergangene Alarme und Reaktionen"
                    iconColor="amber"
                />

                {alertsLoading ? (
                    <div className="flex items-center justify-center py-8">
                        <Loader2 size={32} className="text-amber-400 animate-spin" />
                    </div>
                ) : alerts.length === 0 ? (
                    <div className="text-center py-8 text-slate-500">
                        <Shield size={48} className="mx-auto mb-3 opacity-50" />
                        <p>Keine Sicherheitsereignisse</p>
                        <p className="text-sm mt-1">Das ist gut! Keine Intrusion-Versuche erkannt.</p>
                    </div>
                ) : (
                    <div className="overflow-x-auto">
                        <table className="w-full">
                            <thead>
                                <tr className="border-b border-white/10">
                                    <th className="text-left py-3 px-4 text-slate-400 font-medium">Datum</th>
                                    <th className="text-left py-3 px-4 text-slate-400 font-medium">Ereignis</th>
                                    <th className="text-left py-3 px-4 text-slate-400 font-medium">Aktion</th>
                                </tr>
                            </thead>
                            <tbody>
                                {alerts.map((alert, idx) => (
                                    <tr key={alert.id || idx} className="border-b border-white/5 hover:bg-white/5">
                                        <td className="py-3 px-4 text-slate-400 text-sm font-mono">
                                            {formatDate(alert.created_at)}
                                        </td>
                                        <td className="py-3 px-4">
                                            <span className="text-rose-400">
                                                {alert.action || alert.message || "Security Event"}
                                            </span>
                                        </td>
                                        <td className="py-3 px-4">
                                            <span className="px-2 py-1 bg-rose-500/20 text-rose-400 rounded-lg text-xs font-medium">
                                                🔒 System Locked
                                            </span>
                                        </td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
                )}
            </GlassCard>
        </div>
    );
}
