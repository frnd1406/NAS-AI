import { useState, useEffect } from "react";
import {
    HardDrive,
    Trash2,
    Folder,
    File,
    AlertTriangle,
    Loader2,
    RefreshCw,
    Plus,
    X,
    Database,
    Save,
    ChevronRight,
    Info,
    Server
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
        <div className="p-3 rounded-xl bg-blue-500/20 border border-blue-500/30">
            <Icon size={24} className="text-blue-400" />
        </div>
        <div>
            <h2 className="text-xl font-bold text-white tracking-tight">{title}</h2>
            <p className="text-slate-400 text-sm mt-1">{description}</p>
        </div>
    </div>
);

// Helper function for formatting size
const formatSize = (bytes) => {
    if (bytes >= 1024 * 1024 * 1024 * 1024) {
        return (bytes / (1024 * 1024 * 1024 * 1024)).toFixed(2) + " TB";
    } else if (bytes >= 1024 * 1024 * 1024) {
        return (bytes / (1024 * 1024 * 1024)).toFixed(2) + " GB";
    } else if (bytes >= 1024 * 1024) {
        return (bytes / (1024 * 1024)).toFixed(2) + " MB";
    }
    return bytes + " B";
};

// Progress Bar for Disk Usage (now clickable)
const DiskUsageBar = ({ disk, onClick, isSelected }) => {
    const { used, total, mount_point, filesystem, free, device, drive_type } = disk;
    const percentage = total > 0 ? (used / total) * 100 : 0;

    // Color based on usage
    let barColor = "bg-emerald-500";
    let bgColor = "bg-emerald-500/20";
    let borderColor = "border-white/5";
    if (percentage > 90) {
        barColor = "bg-rose-500";
        bgColor = "bg-rose-500/20";
    } else if (percentage > 70) {
        barColor = "bg-amber-500";
        bgColor = "bg-amber-500/20";
    }
    if (isSelected) {
        borderColor = "border-blue-500/50";
    }

    // Clean up mount point display (remove /host prefix for cleaner display)
    const displayMount = mount_point?.replace('/host', '') || '/';

    return (
        <div
            onClick={onClick}
            className={`space-y-3 p-4 bg-slate-800/30 rounded-xl border ${borderColor} cursor-pointer hover:bg-slate-800/50 hover:border-blue-500/30 transition-all group`}
        >
            <div className="flex justify-between items-start">
                <div className="flex items-center gap-3">
                    <div className={`p-2 rounded-lg ${isSelected ? 'bg-blue-500/20' : 'bg-slate-700/50'} transition-colors`}>
                        <HardDrive size={18} className={isSelected ? 'text-blue-400' : 'text-slate-400'} />
                    </div>
                    <div>
                        <div className="flex items-center gap-2">
                            <span className="text-white font-medium">{device || displayMount}</span>
                            <span className={`text-xs px-2 py-0.5 rounded-full ${drive_type === 'NVMe SSD' ? 'bg-blue-500/20 text-blue-400' :
                                drive_type === 'Removable' ? 'bg-amber-500/20 text-amber-400' :
                                    'bg-slate-700 text-slate-300'
                                }`}>
                                {drive_type || filesystem}
                            </span>
                        </div>
                        <span className="text-xs text-slate-500">{displayMount || mount_point}</span>
                    </div>
                </div>
                <div className="flex items-center gap-2">
                    <span className="text-white font-mono text-sm">
                        {formatSize(used)} / {formatSize(total)}
                    </span>
                    <ChevronRight size={16} className="text-slate-500 group-hover:text-blue-400 transition-colors" />
                </div>
            </div>

            <div className={`h-3 rounded-full ${bgColor} overflow-hidden`}>
                <div
                    className={`h-full ${barColor} transition-all duration-500 rounded-full`}
                    style={{ width: `${Math.min(percentage, 100)}%` }}
                />
            </div>

            <div className="flex justify-between text-xs text-slate-500">
                <span>{formatSize(free)} frei</span>
                <span>{percentage.toFixed(1)}% belegt</span>
            </div>
        </div>
    );
};

// Disk Detail Modal
const DiskDetailModal = ({ disk, onClose }) => {
    if (!disk) return null;

    const percentage = disk.total > 0 ? (disk.used / disk.total) * 100 : 0;

    let statusColor = "text-emerald-400";
    let statusBg = "bg-emerald-500/20";
    let statusText = "Gesund";
    if (percentage > 90) {
        statusColor = "text-rose-400";
        statusBg = "bg-rose-500/20";
        statusText = "Kritisch";
    } else if (percentage > 70) {
        statusColor = "text-amber-400";
        statusBg = "bg-amber-500/20";
        statusText = "Warnung";
    }

    return (
        <div className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50" onClick={onClose}>
            <div
                className="bg-slate-900/95 border border-white/10 rounded-2xl p-6 max-w-lg w-full mx-4 shadow-2xl"
                onClick={(e) => e.stopPropagation()}
            >
                {/* Header */}
                <div className="flex items-start justify-between mb-6">
                    <div className="flex items-center gap-4">
                        <div className={`p-3 rounded-xl ${disk.drive_type === 'NVMe SSD' ? 'bg-blue-500/20 border border-blue-500/30' :
                                disk.drive_type === 'Removable' ? 'bg-amber-500/20 border border-amber-500/30' :
                                    'bg-slate-700/50 border border-white/10'
                            }`}>
                            <HardDrive size={28} className={
                                disk.drive_type === 'NVMe SSD' ? 'text-blue-400' :
                                    disk.drive_type === 'Removable' ? 'text-amber-400' :
                                        'text-slate-400'
                            } />
                        </div>
                        <div>
                            <h2 className="text-xl font-bold text-white">{disk.device || disk.mount_point}</h2>
                            <div className="flex items-center gap-2 mt-1">
                                <span className={`text-xs px-2 py-0.5 rounded-full ${disk.drive_type === 'NVMe SSD' ? 'bg-blue-500/20 text-blue-400' :
                                        disk.drive_type === 'Removable' ? 'bg-amber-500/20 text-amber-400' :
                                            'bg-slate-700 text-slate-300'
                                    }`}>
                                    {disk.drive_type || 'Volume'}
                                </span>
                                <span className="text-slate-500 text-sm">{disk.filesystem}</span>
                            </div>
                        </div>
                    </div>
                    <button
                        onClick={onClose}
                        className="p-2 text-slate-400 hover:text-white hover:bg-white/10 rounded-lg transition-colors"
                    >
                        <X size={20} />
                    </button>
                </div>

                {/* Status Badge */}
                <div className={`inline-flex items-center gap-2 px-3 py-1.5 ${statusBg} rounded-full mb-6`}>
                    <div className={`w-2 h-2 rounded-full ${statusColor.replace('text-', 'bg-')} animate-pulse`} />
                    <span className={`text-sm font-medium ${statusColor}`}>{statusText}</span>
                </div>

                {/* Usage Bar */}
                <div className="mb-6">
                    <div className="flex justify-between text-sm mb-2">
                        <span className="text-slate-400">Speichernutzung</span>
                        <span className="text-white font-mono">{percentage.toFixed(1)}%</span>
                    </div>
                    <div className="h-4 rounded-full bg-slate-800 overflow-hidden">
                        <div
                            className={`h-full ${percentage > 90 ? 'bg-rose-500' : percentage > 70 ? 'bg-amber-500' : 'bg-emerald-500'} transition-all rounded-full`}
                            style={{ width: `${percentage}%` }}
                        />
                    </div>
                </div>

                {/* Stats Grid */}
                <div className="grid grid-cols-3 gap-4 mb-6">
                    <div className="p-4 bg-slate-800/50 rounded-xl border border-white/5 text-center">
                        <Server size={20} className="text-blue-400 mx-auto mb-2" />
                        <p className="text-white font-mono text-lg">{formatSize(disk.total)}</p>
                        <p className="text-slate-500 text-xs">Gesamt</p>
                    </div>
                    <div className="p-4 bg-slate-800/50 rounded-xl border border-white/5 text-center">
                        <HardDrive size={20} className="text-amber-400 mx-auto mb-2" />
                        <p className="text-white font-mono text-lg">{formatSize(disk.used)}</p>
                        <p className="text-slate-500 text-xs">Belegt</p>
                    </div>
                    <div className="p-4 bg-slate-800/50 rounded-xl border border-white/5 text-center">
                        <Database size={20} className="text-emerald-400 mx-auto mb-2" />
                        <p className="text-white font-mono text-lg">{formatSize(disk.free)}</p>
                        <p className="text-slate-500 text-xs">Frei</p>
                    </div>
                </div>

                {/* Details List */}
                <div className="space-y-3">
                    <div className="flex justify-between p-3 bg-slate-800/30 rounded-lg">
                        <span className="text-slate-400">Mount-Punkt</span>
                        <span className="text-white font-mono text-sm">{disk.mount_point}</span>
                    </div>
                    <div className="flex justify-between p-3 bg-slate-800/30 rounded-lg">
                        <span className="text-slate-400">Dateisystem</span>
                        <span className="text-white font-mono text-sm">{disk.filesystem}</span>
                    </div>
                    <div className="flex justify-between p-3 bg-slate-800/30 rounded-lg">
                        <span className="text-slate-400">Auslastung</span>
                        <span className={`font-mono text-sm ${statusColor}`}>{percentage.toFixed(2)}%</span>
                    </div>
                </div>

                {/* Info Note */}
                <div className="mt-6 p-3 bg-blue-500/10 border border-blue-500/20 rounded-lg flex items-start gap-3">
                    <Info size={18} className="text-blue-400 mt-0.5 flex-shrink-0" />
                    <p className="text-slate-400 text-sm">
                        Um diesen Speicher für AI-Indexierung zu verwenden, füge den Pfad unter "AI Index Locations" hinzu.
                    </p>
                </div>
            </div>
        </div>
    );
};


export default function StorageTab() {
    const toast = useToast();
    const [loading, setLoading] = useState(true);
    const [emptyingTrash, setEmptyingTrash] = useState(false);
    const [savingPaths, setSavingPaths] = useState(false);

    // Data States
    const [disks, setDisks] = useState([]);
    const [selectedDisk, setSelectedDisk] = useState(null);
    const [trashData, setTrashData] = useState({ size: 0, count: 0 });
    const [settings, setSettings] = useState({
        warningThreshold: 80,
        criticalThreshold: 90,
        autoCleanup: false,
        cleanupAgeDays: 30,
        aiIndexPaths: []
    });

    // New Path Input
    const [newPath, setNewPath] = useState("");

    useEffect(() => {
        loadData();
    }, []);

    const loadData = async () => {
        setLoading(true);
        try {
            const [hwData, settingsData, trashRes] = await Promise.all([
                apiRequest("/api/v1/system/hardware/storage", { method: "GET" }).catch(() => []),
                apiRequest("/api/v1/storage/settings", { method: "GET" }),
                apiRequest("/api/v1/storage/trash", { method: "GET" }).catch(() => ({ items: [] }))
            ]);

            setDisks(hwData || []);
            setSettings({
                ...settingsData,
                aiIndexPaths: settingsData?.aiIndexPaths || []
            });

            const trashItems = trashRes?.items || [];
            setTrashData({
                count: trashItems.length,
                size: trashItems.reduce((sum, item) => sum + (item.size || 0), 0)
            });

        } catch (err) {
            console.error(err);
            toast.error("Speicherdaten konnten nicht geladen werden");
        } finally {
            setLoading(false);
        }
    };

    const handleEmptyTrash = async () => {
        if (!confirm("Papierkorb endgültig leeren? Diese Aktion kann nicht rückgängig gemacht werden.")) {
            return;
        }

        setEmptyingTrash(true);
        try {
            await apiRequest("/api/v1/storage/trash/empty", { method: "POST" });
            toast.success("Papierkorb geleert");
            setTrashData({ size: 0, count: 0 });
        } catch (err) {
            toast.error("Fehler beim Leeren des Papierkorbs");
        } finally {
            setEmptyingTrash(false);
        }
    };

    const handleAddPath = () => {
        if (!newPath.trim()) return;
        if (settings.aiIndexPaths.includes(newPath.trim())) {
            toast.error("Pfad existiert bereits");
            return;
        }

        const updatedPaths = [...settings.aiIndexPaths, newPath.trim()];
        savePaths(updatedPaths);
        setNewPath("");
    };

    const handleRemovePath = (pathToRemove) => {
        const updatedPaths = settings.aiIndexPaths.filter(p => p !== pathToRemove);
        savePaths(updatedPaths);
    };

    const savePaths = async (paths) => {
        setSavingPaths(true);
        try {
            // We only update the AI Index Paths here, keeping other settings as is
            await apiRequest("/api/v1/storage/settings", {
                method: "PUT",
                body: JSON.stringify({
                    ...settings,
                    aiIndexPaths: paths
                })
            });
            setSettings(prev => ({ ...prev, aiIndexPaths: paths }));
            toast.success("Pfade aktualisiert");
        } catch (err) {
            toast.error("Fehler beim Speichern der Pfade");
        } finally {
            setSavingPaths(false);
        }
    };

    const formatSize = (bytes) => {
        if (bytes >= 1024 * 1024 * 1024) {
            return (bytes / (1024 * 1024 * 1024)).toFixed(2) + " GB";
        } else if (bytes >= 1024 * 1024) {
            return (bytes / (1024 * 1024)).toFixed(2) + " MB";
        }
        return bytes + " B";
    };

    if (loading) {
        return (
            <div className="flex items-center justify-center h-64">
                <Loader2 size={48} className="text-blue-400 animate-spin" />
            </div>
        );
    }

    return (
        <div className="space-y-6">
            {/* Physical Storage */}
            <GlassCard>
                <SectionHeader
                    icon={HardDrive}
                    title="Physische Speicher"
                    description="Status der verbundenen Laufwerke"
                />

                <div className="space-y-3">
                    {disks.map((disk, idx) => (
                        <DiskUsageBar
                            key={idx}
                            disk={disk}
                            onClick={() => setSelectedDisk(disk)}
                            isSelected={selectedDisk?.mount_point === disk.mount_point}
                        />
                    ))}
                    {disks.length === 0 && (
                        <p className="text-slate-500 italic text-center py-4">Keine Laufwerke gefunden</p>
                    )}
                </div>

                {/* Disk Detail Modal */}
                {selectedDisk && (
                    <DiskDetailModal
                        disk={selectedDisk}
                        onClose={() => setSelectedDisk(null)}
                    />
                )}

                <button
                    onClick={loadData}
                    className="flex items-center gap-2 px-4 py-2 mt-4 text-slate-400 hover:text-white transition-colors"
                >
                    <RefreshCw size={16} />
                    <span className="text-sm">Aktualisieren</span>
                </button>
            </GlassCard>

            {/* AI Index Locations */}
            <GlassCard>
                <SectionHeader
                    icon={Database}
                    title="AI Index Locations"
                    description="Verzeichnisse, die automatisch vom AI Agent überwacht werden"
                />

                <div className="space-y-4">
                    <div className="flex gap-2">
                        <input
                            type="text"
                            value={newPath}
                            onChange={(e) => setNewPath(e.target.value)}
                            placeholder="/mnt/data/documents"
                            className="flex-1 px-4 py-2 bg-slate-800/50 border border-white/10 rounded-xl text-white placeholder:text-slate-500 focus:border-blue-500/50 focus:outline-none"
                            onKeyDown={(e) => e.key === 'Enter' && handleAddPath()}
                        />
                        <button
                            onClick={handleAddPath}
                            disabled={savingPaths || !newPath.trim()}
                            className="px-4 py-2 bg-blue-500/20 hover:bg-blue-500/30 text-blue-400 rounded-xl border border-blue-500/30 transition-all flex items-center gap-2 disabled:opacity-50"
                        >
                            {savingPaths ? <Loader2 size={18} className="animate-spin" /> : <Plus size={18} />}
                            Hinzufügen
                        </button>
                    </div>

                    <div className="space-y-2">
                        {settings.aiIndexPaths.map((path) => (
                            <div key={path} className="flex items-center justify-between p-3 bg-slate-800/30 rounded-xl border border-white/5 group">
                                <div className="flex items-center gap-3">
                                    <Folder size={18} className="text-slate-400" />
                                    <span className="text-slate-200 font-mono text-sm">{path}</span>
                                </div>
                                <button
                                    onClick={() => handleRemovePath(path)}
                                    disabled={savingPaths}
                                    className="p-2 text-slate-500 hover:text-rose-400 hover:bg-rose-500/10 rounded-lg transition-all opacity-0 group-hover:opacity-100"
                                >
                                    <X size={16} />
                                </button>
                            </div>
                        ))}
                        {settings.aiIndexPaths.length === 0 && (
                            <p className="text-slate-500 italic text-sm p-2 text-center">Keine Pfade konfiguriert</p>
                        )}
                    </div>
                </div>
            </GlassCard>

            {/* Trash Bin */}
            <GlassCard>
                <SectionHeader
                    icon={Trash2}
                    title="Papierkorb"
                    description="Gelöschte Dateien, die wiederhergestellt werden können"
                />

                <div className="flex items-center justify-between p-4 bg-slate-800/30 rounded-xl">
                    <div>
                        <p className="text-white font-medium">
                            {trashData.count} Elemente
                        </p>
                        <p className="text-slate-400 text-sm">
                            Größe: {formatSize(trashData.size)}
                        </p>
                    </div>

                    <button
                        onClick={handleEmptyTrash}
                        disabled={emptyingTrash || trashData.count === 0}
                        className="flex items-center gap-2 px-4 py-2 bg-rose-500/20 hover:bg-rose-500/30 text-rose-400 rounded-xl border border-rose-500/30 transition-all disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                        {emptyingTrash ? (
                            <Loader2 size={18} className="animate-spin" />
                        ) : (
                            <Trash2 size={18} />
                        )}
                        <span className="font-medium">Leeren</span>
                    </button>
                </div>
            </GlassCard>
        </div>
    );
}
