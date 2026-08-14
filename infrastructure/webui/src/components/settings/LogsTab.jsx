import { useState, useEffect } from "react";
import {
    FileText,
    Filter,
    Download,
    RefreshCw,
    Loader2,
    Shield,
    User,
    Clock,
    AlertTriangle,
    Check,
    X,
    ChevronLeft,
    ChevronRight
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
        <div className="p-3 rounded-xl bg-violet-500/20 border border-violet-500/30">
            <Icon size={24} className="text-violet-400" />
        </div>
        <div>
            <h2 className="text-xl font-bold text-white tracking-tight">{title}</h2>
            <p className="text-slate-400 text-sm mt-1">{description}</p>
        </div>
    </div>
);

// Log Type Options
const LOG_TYPES = [
    { value: "", label: "Alle Typen" },
    { value: "auth", label: "Authentifizierung" },
    { value: "file", label: "Dateizugriff" },
    { value: "admin", label: "Admin-Aktionen" },
    { value: "security", label: "Sicherheit" },
    { value: "system", label: "System" }
];

export default function LogsTab() {
    const toast = useToast();
    const [loading, setLoading] = useState(true);
    const [logs, setLogs] = useState([]);
    const [page, setPage] = useState(1);
    const [totalPages, setTotalPages] = useState(1);
    const [filters, setFilters] = useState({
        type: "",
        user: "",
        date_from: "",
        date_to: ""
    });

    useEffect(() => {
        loadLogs();
    }, [page, filters]);

    const loadLogs = async () => {
        setLoading(true);
        try {
            const params = new URLSearchParams();
            params.append("page", page.toString());
            params.append("limit", "20");

            if (filters.type) params.append("type", filters.type);
            if (filters.user) params.append("user", filters.user);
            if (filters.date_from) params.append("from", filters.date_from);
            if (filters.date_to) params.append("to", filters.date_to);

            const response = await apiRequest(`/api/v1/admin/audit-logs?${params.toString()}`, {
                method: "GET"
            });

            if (response?.logs) {
                setLogs(response.logs);
                setTotalPages(Math.ceil((response.total || 20) / 20));
            } else if (Array.isArray(response)) {
                setLogs(response);
            }
        } catch (err) {
            toast.error("Logs konnten nicht geladen werden");
            setLogs([]);
        } finally {
            setLoading(false);
        }
    };

    const handleExportCSV = () => {
        if (logs.length === 0) {
            toast.error("Keine Logs zum Exportieren");
            return;
        }

        const headers = ["Datum", "Typ", "Benutzer", "Aktion", "Details"];
        const rows = logs.map(log => [
            new Date(log.created_at).toISOString(),
            log.type || "",
            log.username || log.user_id || "",
            log.action || "",
            log.details || ""
        ]);

        const csv = [headers, ...rows]
            .map(row => row.map(cell => `"${String(cell).replace(/"/g, '""')}"`).join(","))
            .join("\n");

        const blob = new Blob([csv], { type: "text/csv;charset=utf-8;" });
        const url = URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download = `audit-logs-${new Date().toISOString().split("T")[0]}.csv`;
        a.click();
        URL.revokeObjectURL(url);

        toast.success("Export abgeschlossen");
    };

    const formatDate = (dateStr) => {
        if (!dateStr) return "—";
        const date = new Date(dateStr);
        return date.toLocaleDateString("de-DE", {
            day: "2-digit",
            month: "2-digit",
            year: "numeric",
            hour: "2-digit",
            minute: "2-digit",
            second: "2-digit"
        });
    };

    const getTypeIcon = (type) => {
        switch (type) {
            case "auth": return <User size={14} className="text-blue-400" />;
            case "admin": return <Shield size={14} className="text-violet-400" />;
            case "security": return <AlertTriangle size={14} className="text-rose-400" />;
            case "file": return <FileText size={14} className="text-emerald-400" />;
            default: return <Clock size={14} className="text-slate-400" />;
        }
    };

    const getTypeColor = (type) => {
        switch (type) {
            case "auth": return "bg-blue-500/20 text-blue-400";
            case "admin": return "bg-violet-500/20 text-violet-400";
            case "security": return "bg-rose-500/20 text-rose-400";
            case "file": return "bg-emerald-500/20 text-emerald-400";
            default: return "bg-slate-500/20 text-slate-400";
        }
    };

    return (
        <div className="space-y-6">
            {/* Filters */}
            <GlassCard>
                <SectionHeader
                    icon={Filter}
                    title="Filter"
                    description="Logs nach Kriterien filtern"
                />

                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
                    <div>
                        <label className="block text-sm font-medium text-slate-300 mb-2">Typ</label>
                        <select
                            value={filters.type}
                            onChange={(e) => setFilters({ ...filters, type: e.target.value })}
                            className="w-full px-3 py-2.5 bg-slate-800/50 border border-white/10 rounded-xl text-white focus:border-violet-500/50 focus:outline-none"
                        >
                            {LOG_TYPES.map((opt) => (
                                <option key={opt.value} value={opt.value}>{opt.label}</option>
                            ))}
                        </select>
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-slate-300 mb-2">Benutzer</label>
                        <input
                            type="text"
                            value={filters.user}
                            onChange={(e) => setFilters({ ...filters, user: e.target.value })}
                            placeholder="Benutzername..."
                            className="w-full px-3 py-2.5 bg-slate-800/50 border border-white/10 rounded-xl text-white placeholder:text-slate-500 focus:border-violet-500/50 focus:outline-none"
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-slate-300 mb-2">Von</label>
                        <input
                            type="date"
                            value={filters.date_from}
                            onChange={(e) => setFilters({ ...filters, date_from: e.target.value })}
                            className="w-full px-3 py-2.5 bg-slate-800/50 border border-white/10 rounded-xl text-white focus:border-violet-500/50 focus:outline-none"
                        />
                    </div>

                    <div>
                        <label className="block text-sm font-medium text-slate-300 mb-2">Bis</label>
                        <input
                            type="date"
                            value={filters.date_to}
                            onChange={(e) => setFilters({ ...filters, date_to: e.target.value })}
                            className="w-full px-3 py-2.5 bg-slate-800/50 border border-white/10 rounded-xl text-white focus:border-violet-500/50 focus:outline-none"
                        />
                    </div>
                </div>

                <div className="flex gap-3 mt-4">
                    <button
                        onClick={() => { setPage(1); loadLogs(); }}
                        className="flex items-center gap-2 px-4 py-2 bg-violet-500/20 hover:bg-violet-500/30 text-violet-400 rounded-xl border border-violet-500/30 transition-all"
                    >
                        <Filter size={16} />
                        <span>Anwenden</span>
                    </button>

                    <button
                        onClick={() => setFilters({ type: "", user: "", date_from: "", date_to: "" })}
                        className="px-4 py-2 text-slate-400 hover:text-white transition-colors"
                    >
                        Zurücksetzen
                    </button>
                </div>
            </GlassCard>

            {/* Logs Table */}
            <GlassCard>
                <div className="flex items-center justify-between mb-6">
                    <SectionHeader
                        icon={FileText}
                        title="Audit-Logs"
                        description="Systemaktivitäten und Ereignisse"
                    />

                    <button
                        onClick={handleExportCSV}
                        className="flex items-center gap-2 px-4 py-2 bg-slate-700/50 hover:bg-slate-700 text-white rounded-xl transition-all"
                    >
                        <Download size={16} />
                        <span>CSV Export</span>
                    </button>
                </div>

                {loading ? (
                    <div className="flex items-center justify-center py-12">
                        <Loader2 size={32} className="text-violet-400 animate-spin" />
                    </div>
                ) : logs.length === 0 ? (
                    <div className="text-center py-12 text-slate-500">
                        <FileText size={48} className="mx-auto mb-3 opacity-50" />
                        <p>Keine Logs gefunden</p>
                    </div>
                ) : (
                    <>
                        <div className="overflow-x-auto">
                            <table className="w-full">
                                <thead>
                                    <tr className="border-b border-white/10">
                                        <th className="text-left py-3 px-4 text-slate-400 font-medium">Datum</th>
                                        <th className="text-left py-3 px-4 text-slate-400 font-medium">Typ</th>
                                        <th className="text-left py-3 px-4 text-slate-400 font-medium">Benutzer</th>
                                        <th className="text-left py-3 px-4 text-slate-400 font-medium">Aktion</th>
                                        <th className="text-left py-3 px-4 text-slate-400 font-medium">Details</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {logs.map((log, idx) => (
                                        <tr key={log.id || idx} className="border-b border-white/5 hover:bg-white/5">
                                            <td className="py-3 px-4 text-slate-400 text-sm font-mono">
                                                {formatDate(log.created_at)}
                                            </td>
                                            <td className="py-3 px-4">
                                                <span className={`inline-flex items-center gap-1.5 px-2 py-1 rounded-lg text-xs font-medium ${getTypeColor(log.type)}`}>
                                                    {getTypeIcon(log.type)}
                                                    {log.type || "system"}
                                                </span>
                                            </td>
                                            <td className="py-3 px-4 text-white">
                                                {log.username || log.user_id || "—"}
                                            </td>
                                            <td className="py-3 px-4 text-slate-300">
                                                {log.action || "—"}
                                            </td>
                                            <td className="py-3 px-4 text-slate-500 text-sm max-w-xs truncate">
                                                {log.details || log.message || "—"}
                                            </td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                        </div>

                        {/* Pagination */}
                        <div className="flex items-center justify-between mt-4 pt-4 border-t border-white/5">
                            <span className="text-slate-400 text-sm">
                                Seite {page} von {totalPages}
                            </span>
                            <div className="flex gap-2">
                                <button
                                    onClick={() => setPage(p => Math.max(1, p - 1))}
                                    disabled={page <= 1}
                                    className="p-2 bg-slate-800/50 hover:bg-slate-700 rounded-lg text-white disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                                >
                                    <ChevronLeft size={18} />
                                </button>
                                <button
                                    onClick={() => setPage(p => Math.min(totalPages, p + 1))}
                                    disabled={page >= totalPages}
                                    className="p-2 bg-slate-800/50 hover:bg-slate-700 rounded-lg text-white disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                                >
                                    <ChevronRight size={18} />
                                </button>
                            </div>
                        </div>
                    </>
                )}

                <button
                    onClick={loadLogs}
                    className="flex items-center gap-2 px-4 py-2 mt-4 text-slate-400 hover:text-white transition-colors"
                >
                    <RefreshCw size={16} />
                    <span className="text-sm">Aktualisieren</span>
                </button>
            </GlassCard>
        </div>
    );
}
