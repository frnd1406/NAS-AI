import { useEffect, useState } from "react";
import { apiRequest } from "../lib/api";
import {
  Archive,
  Trash2,
  RefreshCw,
  Plus,
  HardDrive,
  AlertTriangle,
  Loader2,
  Settings,
  Clock,
  Save,
  FolderOpen,
  Calendar,
  CheckCircle,
  X,
  Cloud,
  CloudOff,
  Key,
  LogOut,
} from "lucide-react";

// Glass Card Component
const GlassCard = ({ children, className = "" }) => (
  <div className={`relative overflow-hidden rounded-2xl border border-white/10 bg-slate-900/40 backdrop-blur-xl shadow-2xl ${className}`}>
    {/* Internal "Shimmer" Reflection */}
    <div className="absolute top-0 left-0 w-full h-[1px] bg-gradient-to-r from-transparent via-white/20 to-transparent opacity-50"></div>
    <div className="h-full flex flex-col">
      {children}
    </div>
  </div>
);

export default function Backup() {
  const [backups, setBackups] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [processingId, setProcessingId] = useState(null);

  // Settings State - RENAMED to showSettingsModal
  const [showSettingsModal, setShowSettingsModal] = useState(false);
  const [settingsLoading, setSettingsLoading] = useState(false);
  const [settingsSaving, setSettingsSaving] = useState(false);
  const [successMessage, setSuccessMessage] = useState("");
  const [backupSchedule, setBackupSchedule] = useState("03:00");
  const [retentionDays, setRetentionDays] = useState(7);
  const [backupPath, setBackupPath] = useState("/mnt/backups");
  const [autoBackupEnabled, setAutoBackupEnabled] = useState(true);

  // Smart Config State
  const [backupFrequency, setBackupFrequency] = useState("daily"); // daily, weekly, cron
  const [pathValidationStatus, setPathValidationStatus] = useState(null); // null, 'valid', 'warning', 'error'
  const [pathValidationMessage, setPathValidationMessage] = useState("");
  const [pathValidating, setPathValidating] = useState(false);

  // iCloud Connect State
  const [icloudState, setIcloudState] = useState("disconnected"); // disconnected, connecting, pending_2fa, connected, syncing
  const [icloudAccount, setIcloudAccount] = useState("");
  const [icloudAppleId, setIcloudAppleId] = useState("");
  const [icloudPassword, setIcloudPassword] = useState("");
  const [icloud2faCode, setIcloud2faCode] = useState("");
  const [icloudError, setIcloudError] = useState("");
  const [icloudLoading, setIcloudLoading] = useState(false);

  const loadBackups = async () => {
    setLoading(true);
    setError("");
    try {
      const data = await apiRequest("/api/v1/backups", { method: "GET" });
      setBackups(data.items || []);
    } catch (err) {
      setError(err.message || "Fehler beim Laden");
    } finally {
      setLoading(false);
    }
  };

  const loadSettings = async () => {
    setSettingsLoading(true);
    try {
      const data = await apiRequest("/api/v1/system/settings", { method: "GET" });

      // API gibt { backup: { schedule, retention, path } } zurück
      if (data.backup) {
        // Konvertiere Cron-Format "0 3 * * *" zu "HH:MM" Format und erkenne Frequenz
        if (data.backup.schedule) {
          const cronParts = data.backup.schedule.split(' ');
          if (cronParts.length >= 5) {
            const minutes = cronParts[0].padStart(2, '0');
            const hours = cronParts[1].padStart(2, '0');
            const dayOfWeek = cronParts[4];

            setBackupSchedule(`${hours}:${minutes}`);

            // Erkenne Frequenz aus Cron-String
            if (dayOfWeek === '0' || dayOfWeek === '7') {
              setBackupFrequency('weekly');
            } else if (dayOfWeek === '*') {
              setBackupFrequency('daily');
            } else {
              setBackupFrequency('cron');
            }
          }
        }

        if (data.backup.retention !== undefined) setRetentionDays(data.backup.retention);
        if (data.backup.path) {
          setBackupPath(data.backup.path);
          validatePath(data.backup.path); // Initial validation
        }
      }
    } catch (err) {
      console.error("Fehler beim Laden der Einstellungen:", err);
      // Fehlermeldung nicht anzeigen, da Einstellungen optional sind
    } finally {
      setSettingsLoading(false);
    }
  };

  // Path validation with debounce
  const validatePath = async (path) => {
    if (!path || path.trim() === '') {
      setPathValidationStatus('error');
      setPathValidationMessage('Pfad darf nicht leer sein');
      return;
    }

    setPathValidating(true);
    setPathValidationStatus(null);
    setPathValidationMessage('');

    try {
      const data = await apiRequest("/api/v1/system/validate-path", {
        method: "POST",
        body: JSON.stringify({ path }),
      });

      if (data.valid) {
        setPathValidationStatus('valid');
        setPathValidationMessage(data.message || 'Pfad existiert und ist beschreibbar');
        return;
      }

      if (!data.exists) {
        setPathValidationStatus('warning');
        setPathValidationMessage(data.message || 'Pfad existiert nicht (wird bei Bedarf erstellt)');
        return;
      }

      if (!data.writable) {
        setPathValidationStatus('error');
        setPathValidationMessage(data.message || 'Pfad existiert, ist aber nicht beschreibbar');
        return;
      }

      setPathValidationStatus('error');
      setPathValidationMessage(data.message || 'Pfad-Validierung fehlgeschlagen');
    } catch (err) {
      setPathValidationStatus('error');
      setPathValidationMessage(err.message || 'Verbindungsfehler bei Pfad-Validierung');
    } finally {
      setPathValidating(false);
    }
  };

  // Debounced path validation
  useEffect(() => {
    const timer = setTimeout(() => {
      if (backupPath && showSettingsModal) {
        validatePath(backupPath);
      }
    }, 500);

    return () => clearTimeout(timer);
  }, [backupPath, showSettingsModal]);

  useEffect(() => {
    loadBackups();
    loadSettings();
    checkIcloudStatus();
  }, []);

  // ============================================================
  // iCloud Bridge Functions
  // ============================================================

  const ICLOUD_BRIDGE_URL = "/icloud-api";

  const checkIcloudStatus = async () => {
    try {
      const res = await fetch(`${ICLOUD_BRIDGE_URL}/status`);
      const data = await res.json();
      setIcloudState(data.status || "disconnected");
      setIcloudAccount(data.account || "");
    } catch (err) {
      // Bridge nicht erreichbar - wahrscheinlich nicht gestartet
      setIcloudState("disconnected");
    }
  };

  const handleIcloudLogin = async () => {
    if (!icloudAppleId || !icloudPassword) {
      setIcloudError("Apple-ID und Passwort erforderlich");
      return;
    }

    setIcloudLoading(true);
    setIcloudError("");

    try {
      const res = await fetch(`${ICLOUD_BRIDGE_URL}/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ appleId: icloudAppleId, password: icloudPassword }),
      });

      const data = await res.json();

      if (data.status === "connected") {
        setIcloudState("connected");
        setIcloudAccount(data.account);
        setIcloudPassword(""); // Passwort nicht speichern
      } else if (data.status === "pending_2fa") {
        setIcloudState("pending_2fa");
        setIcloudAccount(data.account);
        setIcloudPassword(""); // Passwort nicht speichern
      } else if (data.error) {
        setIcloudError(data.error);
      }
    } catch (err) {
      setIcloudError("Verbindung zur iCloud-Bridge fehlgeschlagen");
    } finally {
      setIcloudLoading(false);
    }
  };

  const handleIcloud2fa = async () => {
    if (!icloud2faCode || icloud2faCode.length !== 6) {
      setIcloudError("6-stelliger Code erforderlich");
      return;
    }

    setIcloudLoading(true);
    setIcloudError("");

    try {
      const res = await fetch(`${ICLOUD_BRIDGE_URL}/2fa`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ code: icloud2faCode }),
      });

      const data = await res.json();

      if (data.status === "connected") {
        setIcloudState("connected");
        setIcloud2faCode("");
      } else if (data.error) {
        setIcloudError(data.error);
      }
    } catch (err) {
      setIcloudError("2FA-Verifizierung fehlgeschlagen");
    } finally {
      setIcloudLoading(false);
    }
  };

  const handleIcloudSync = async () => {
    setIcloudLoading(true);
    setIcloudError("");

    try {
      const res = await fetch(`${ICLOUD_BRIDGE_URL}/sync`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ type: "photos" }),
      });

      const data = await res.json();

      if (data.status === "syncing") {
        setIcloudState("syncing");
        // Nach kurzer Zeit Status prüfen
        setTimeout(checkIcloudStatus, 5000);
      }
    } catch (err) {
      setIcloudError("Sync konnte nicht gestartet werden");
    } finally {
      setIcloudLoading(false);
    }
  };

  const handleIcloudDisconnect = async () => {
    if (!window.confirm("iCloud-Verbindung wirklich trennen?")) return;

    try {
      await fetch(`${ICLOUD_BRIDGE_URL}/disconnect`, { method: "POST" });
      setIcloudState("disconnected");
      setIcloudAccount("");
      setIcloudAppleId("");
    } catch (err) {
      setIcloudError("Trennen fehlgeschlagen");
    }
  };

  const saveSettings = async () => {
    setSettingsSaving(true);
    setError("");
    setSuccessMessage("");

    try {
      // Validate path before saving
      if (pathValidationStatus === 'error') {
        throw new Error('Bitte beheben Sie zuerst die Fehler beim Backup-Pfad');
      }

      // Convert HH:MM to cron format based on frequency
      const [hours, minutes] = backupSchedule.split(':');
      let cronSchedule;

      switch (backupFrequency) {
        case 'daily':
          cronSchedule = `${parseInt(minutes)} ${parseInt(hours)} * * *`;
          break;
        case 'weekly':
          // Sunday = 0 or 7 in cron
          cronSchedule = `${parseInt(minutes)} ${parseInt(hours)} * * 0`;
          break;
        case 'cron':
          // For manual cron, user should provide full cron string (future feature)
          cronSchedule = `${parseInt(minutes)} ${parseInt(hours)} * * *`;
          break;
        default:
          cronSchedule = `${parseInt(minutes)} ${parseInt(hours)} * * *`;
      }

      await apiRequest("/api/v1/system/settings/backup", {
        method: "PUT",
        body: JSON.stringify({
          schedule: cronSchedule,        // API erwartet "schedule" im Cron-Format
          retention: retentionDays,      // API erwartet "retention"
          path: backupPath,              // API erwartet "path"
        }),
      });

      const frequencyText = backupFrequency === 'daily' ? 'täglich' : 'sonntags';
      setSuccessMessage(`✓ Einstellungen gespeichert! Nächstes Backup ${frequencyText} um ${backupSchedule} Uhr`);
      setTimeout(() => setSuccessMessage(""), 5000);
      setShowSettingsModal(false); // Close modal on success
    } catch (err) {
      setError(err.message || "Speichern fehlgeschlagen");
    } finally {
      setSettingsSaving(false);
    }
  };

  const createBackup = async () => {
    setBusy(true);
    setError("");
    try {
      await apiRequest("/api/v1/backups", { method: "POST" });
      await loadBackups();
    } catch (err) {
      setError(err.message || "Backup fehlgeschlagen");
    } finally {
      setBusy(false);
    }
  };

  const restoreBackup = async (id) => {
    if (!window.confirm("⚠️ ACHTUNG: Dies überschreibt alle aktuellen Daten im NAS! Wollen Sie wirklich fortfahren?")) return;

    setProcessingId(id);
    setBusy(true);
    setError("");
    try {
      await apiRequest(`/api/v1/backups/${encodeURIComponent(id)}/restore`, { method: "POST" });
      await loadBackups();
      alert("✓ System erfolgreich wiederhergestellt!");
    } catch (err) {
      setError(err.message || "Restore fehlgeschlagen");
    } finally {
      setBusy(false);
      setProcessingId(null);
    }
  };

  const deleteBackup = async (id) => {
    if (!window.confirm("Soll dieses Backup wirklich gelöscht werden?")) return;

    setBusy(true);
    setError("");
    try {
      await apiRequest(`/api/v1/backups/${encodeURIComponent(id)}`, { method: "DELETE" });
      await loadBackups();
    } catch (err) {
      setError(err.message || "Löschen fehlgeschlagen");
    } finally {
      setBusy(false);
    }
  };

  const formatSize = (bytes) => {
    if (!bytes || bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  return (
    <div className="space-y-6">

      {/* Header Section with Auto-Backup Status */}
      <div className="flex flex-col md:flex-row md:items-start justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold text-white tracking-tight flex items-center gap-3">
            <Archive className="text-blue-400" size={32} />
            Backup & Recovery
          </h1>
          <p className="text-slate-400 mt-2 text-sm">
            Verwalten Sie System-Snapshots und stellen Sie Daten wieder her.
          </p>

          {/* Auto-Backup Status Badge */}
          {autoBackupEnabled && (
            <div className="mt-3 inline-flex items-center gap-2 px-3 py-1.5 rounded-lg bg-emerald-500/10 border border-emerald-500/20">
              <CheckCircle size={14} className="text-emerald-400" />
              <span className="text-xs font-medium text-emerald-400">
                Auto-Backup: AKTIV (Täglich um {backupSchedule} Uhr)
              </span>
            </div>
          )}
        </div>

        <div className="flex items-center gap-3 self-start">
          <button
            onClick={() => setShowSettingsModal(true)}
            className="flex items-center justify-center gap-2 px-4 py-3 rounded-xl bg-white/5 hover:bg-white/10 text-slate-300 border border-white/10 transition-all shadow-[0_0_15px_rgba(255,255,255,0.05)]"
            aria-label="Backup-Konfiguration"
          >
            <Settings size={18} />
            <span className="hidden sm:inline font-medium">Konfiguration</span>
          </button>
          <button
            onClick={createBackup}
            disabled={busy || loading}
            className="flex items-center justify-center gap-2 px-6 py-3 bg-blue-500/20 hover:bg-blue-500/30 text-blue-400 rounded-xl font-medium transition-all shadow-[0_0_20px_rgba(59,130,246,0.3)] hover:shadow-[0_0_30px_rgba(59,130,246,0.5)] disabled:opacity-50 disabled:cursor-not-allowed border border-blue-500/30"
          >
            {busy ? <Loader2 size={20} className="animate-spin" /> : <Plus size={20} />}
            <span>Neues Backup erstellen</span>
          </button>
        </div>
      </div>

      {/* Success Message */}
      {successMessage && (
        <div className="rounded-xl border border-emerald-500/30 bg-emerald-500/10 p-4 animate-in fade-in duration-300">
          <p className="text-emerald-400 text-sm font-medium">{successMessage}</p>
        </div>
      )}

      {/* Error Display */}
      {error && (
        <div className="rounded-xl border border-rose-500/30 bg-rose-500/10 p-4">
          <p className="text-rose-400 text-sm font-medium">{error}</p>
        </div>
      )}

      {/* Settings Modal */}
      {showSettingsModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4 animate-in fade-in duration-200">
          {/* Modal Container */}
          <div className="w-full max-w-2xl animate-in zoom-in-95 duration-200">
            <GlassCard className="max-h-[90vh] overflow-y-auto">
              <div className="p-6">
                {/* Modal Header */}
                <div className="flex items-start justify-between mb-6">
                  <div className="flex items-center gap-3">
                    <div className="p-3 rounded-xl bg-violet-500/20 border border-violet-500/30">
                      <Settings size={24} className="text-violet-400" />
                    </div>
                    <div>
                      <h2 className="text-2xl font-bold text-white tracking-tight">
                        Backup-Einstellungen
                      </h2>
                      <p className="text-slate-400 text-sm mt-1">
                        Konfigurieren Sie Ihre Backup-Strategie
                      </p>
                    </div>
                  </div>

                  {/* Close Button */}
                  <button
                    onClick={() => setShowSettingsModal(false)}
                    className="p-2 rounded-lg bg-slate-800/50 hover:bg-rose-500/20 text-slate-400 hover:text-rose-400 border border-white/10 hover:border-rose-500/30 transition-all"
                    title="Schließen"
                  >
                    <X size={20} />
                  </button>
                </div>

                {/* Settings Form */}
                {settingsLoading ? (
                  <div className="flex items-center justify-center py-12">
                    <Loader2 size={32} className="text-blue-400 animate-spin" />
                  </div>
                ) : (
                  <div className="space-y-6">
                    {/* Backup Frequency */}
                    <div className="space-y-2">
                      <label className="flex items-center gap-2 text-sm font-medium text-slate-300">
                        <Calendar size={16} className="text-blue-400" />
                        Häufigkeit
                      </label>
                      <select
                        value={backupFrequency}
                        onChange={(e) => setBackupFrequency(e.target.value)}
                        className="w-full px-4 py-2.5 bg-slate-800/50 border border-white/10 rounded-lg text-white focus:border-blue-500/50 focus:bg-slate-800 focus:outline-none transition-all appearance-none cursor-pointer"
                      >
                        <option value="daily">Täglich</option>
                        <option value="weekly">Wöchentlich (Sonntags)</option>
                        <option value="cron">Manuell (Erweitert)</option>
                      </select>
                      <p className="text-xs text-slate-500">
                        {backupFrequency === 'daily' && 'Backup wird jeden Tag zur angegebenen Uhrzeit erstellt'}
                        {backupFrequency === 'weekly' && 'Backup wird jeden Sonntag zur angegebenen Uhrzeit erstellt'}
                        {backupFrequency === 'cron' && 'Erweiterte Cron-Konfiguration (derzeit als täglich behandelt)'}
                      </p>
                    </div>

                    {/* Schedule Time */}
                    <div className="space-y-2">
                      <label className="flex items-center gap-2 text-sm font-medium text-slate-300">
                        <Clock size={16} className="text-blue-400" />
                        Backup-Zeitplan
                      </label>
                      <input
                        type="time"
                        value={backupSchedule}
                        onChange={(e) => setBackupSchedule(e.target.value)}
                        className="w-full md:w-64 px-4 py-2.5 bg-slate-800/50 border border-white/10 rounded-lg text-white font-mono focus:border-blue-500/50 focus:bg-slate-800 focus:outline-none transition-all"
                      />
                      <p className="text-xs text-slate-500">
                        Uhrzeit für das automatische Backup
                      </p>
                    </div>

                    {/* Retention Period */}
                    <div className="space-y-3">
                      <label className="flex items-center gap-2 text-sm font-medium text-slate-300">
                        <Calendar size={16} className="text-blue-400" />
                        Aufbewahrungszeitraum
                      </label>

                      {/* Preset Chips */}
                      <div className="flex items-center gap-2">
                        <span className="text-xs text-slate-400 mr-1">Presets:</span>
                        {[7, 14, 30].map((days) => (
                          <button
                            key={days}
                            onClick={() => setRetentionDays(days)}
                            className={`px-3 py-1.5 rounded-lg text-xs font-medium transition-all border ${retentionDays === days
                              ? 'bg-blue-500/20 text-blue-400 border-blue-500/40 shadow-[0_0_10px_rgba(59,130,246,0.3)]'
                              : 'bg-slate-800/50 text-slate-400 border-white/10 hover:bg-slate-800 hover:text-slate-300'
                              }`}
                          >
                            {days} Tage
                          </button>
                        ))}
                      </div>

                      <div className="flex items-center gap-4">
                        <input
                          type="range"
                          min="1"
                          max="30"
                          value={retentionDays}
                          onChange={(e) => setRetentionDays(parseInt(e.target.value))}
                          className="flex-1 h-2 bg-slate-700 rounded-lg appearance-none cursor-pointer accent-blue-500"
                        />
                        <div className="px-4 py-2 bg-slate-800 border border-white/10 rounded-lg min-w-[80px] text-center">
                          <span className="text-white font-mono font-medium">{retentionDays}</span>
                          <span className="text-slate-400 text-xs ml-1">Tage</span>
                        </div>
                      </div>
                      <p className="text-xs text-slate-500">
                        Ältere Backups werden automatisch gelöscht. Behalte die letzten <span className="text-blue-400 font-medium">{retentionDays}</span> Snapshots.
                      </p>
                    </div>

                    {/* Backup Path */}
                    <div className="space-y-2">
                      <label className="flex items-center gap-2 text-sm font-medium text-slate-300">
                        <FolderOpen size={16} className="text-blue-400" />
                        Speicherort
                      </label>
                      <div className="relative">
                        <input
                          type="text"
                          value={backupPath}
                          onChange={(e) => setBackupPath(e.target.value)}
                          placeholder="/mnt/backups"
                          className={`w-full px-4 py-2.5 pr-10 bg-slate-800/50 border rounded-lg text-white font-mono focus:bg-slate-800 focus:outline-none transition-all ${pathValidationStatus === 'valid'
                            ? 'border-emerald-500/50 focus:border-emerald-500/70'
                            : pathValidationStatus === 'warning'
                              ? 'border-amber-500/50 focus:border-amber-500/70'
                              : pathValidationStatus === 'error'
                                ? 'border-rose-500/50 focus:border-rose-500/70'
                                : 'border-white/10 focus:border-blue-500/50'
                            }`}
                        />
                        {/* Status Indicator */}
                        <div className="absolute right-3 top-1/2 -translate-y-1/2">
                          {pathValidating ? (
                            <Loader2 size={18} className="text-blue-400 animate-spin" />
                          ) : pathValidationStatus === 'valid' ? (
                            <CheckCircle size={18} className="text-emerald-400" />
                          ) : pathValidationStatus === 'warning' ? (
                            <AlertTriangle size={18} className="text-amber-400" />
                          ) : pathValidationStatus === 'error' ? (
                            <X size={18} className="text-rose-400" />
                          ) : null}
                        </div>
                      </div>
                      {/* Validation Message */}
                      {pathValidationMessage && (
                        <div className={`flex items-start gap-2 text-xs p-2 rounded-lg ${pathValidationStatus === 'valid'
                          ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/20'
                          : pathValidationStatus === 'warning'
                            ? 'bg-amber-500/10 text-amber-400 border border-amber-500/20'
                            : 'bg-rose-500/10 text-rose-400 border border-rose-500/20'
                          }`}>
                          {pathValidationStatus === 'valid' ? (
                            <CheckCircle size={14} className="shrink-0 mt-0.5" />
                          ) : pathValidationStatus === 'warning' ? (
                            <AlertTriangle size={14} className="shrink-0 mt-0.5" />
                          ) : (
                            <AlertTriangle size={14} className="shrink-0 mt-0.5" />
                          )}
                          <span>{pathValidationMessage}</span>
                        </div>
                      )}
                      <p className="text-xs text-slate-500">
                        Pfad zum Backup-Verzeichnis auf dem NAS (wird automatisch validiert)
                      </p>
                    </div>

                    {/* Auto-Backup Toggle */}
                    <div className="flex items-center justify-between p-4 bg-slate-800/30 rounded-lg border border-white/5">
                      <div className="flex items-center gap-3">
                        <div className={`p-2 rounded-lg ${autoBackupEnabled ? 'bg-emerald-500/20 border-emerald-500/30' : 'bg-slate-700 border-white/5'} border transition-colors`}>
                          <CheckCircle size={18} className={autoBackupEnabled ? 'text-emerald-400' : 'text-slate-500'} />
                        </div>
                        <div>
                          <p className="text-sm font-medium text-white">Automatische Backups</p>
                          <p className="text-xs text-slate-400">Aktiviert geplante tägliche Snapshots</p>
                        </div>
                      </div>
                      <button
                        onClick={() => setAutoBackupEnabled(!autoBackupEnabled)}
                        className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${autoBackupEnabled ? 'bg-emerald-500' : 'bg-slate-600'}`}
                      >
                        <span className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${autoBackupEnabled ? 'translate-x-6' : 'translate-x-1'}`} />
                      </button>
                    </div>

                    {/* Action Buttons */}
                    <div className="flex items-center justify-end gap-3 pt-4 border-t border-white/5">
                      <button
                        onClick={() => setShowSettingsModal(false)}
                        className="px-6 py-2.5 bg-slate-800/50 hover:bg-slate-800 text-slate-300 hover:text-white rounded-lg font-medium transition-all border border-white/10"
                      >
                        Abbrechen
                      </button>
                      <button
                        onClick={saveSettings}
                        disabled={settingsSaving}
                        className="flex items-center gap-2 px-6 py-2.5 bg-blue-500/20 hover:bg-blue-500/30 text-blue-400 rounded-lg font-medium transition-all border border-blue-500/30 disabled:opacity-50 disabled:cursor-not-allowed shadow-[0_0_15px_rgba(59,130,246,0.2)] hover:shadow-[0_0_20px_rgba(59,130,246,0.4)]"
                      >
                        {settingsSaving ? (
                          <>
                            <Loader2 size={18} className="animate-spin" />
                            <span>Speichere...</span>
                          </>
                        ) : (
                          <>
                            <Save size={18} />
                            <span>Einstellungen speichern</span>
                          </>
                        )}
                      </button>
                    </div>
                  </div>
                )}
              </div>
            </GlassCard>
          </div>
        </div>
      )}


      {/* Backups List */}
      <GlassCard>
        <div className="p-6">
          <div className="flex items-center justify-between mb-6">
            <div>
              <h3 className="text-white font-semibold text-lg tracking-tight flex items-center gap-2">
                <HardDrive size={20} className="text-blue-400" />
                Verfügbare Backups
              </h3>
              <p className="text-slate-400 text-xs mt-1">{backups.length} Snapshot{backups.length !== 1 ? 's' : ''}</p>
            </div>
          </div>

          {loading ? (
            <div className="flex flex-col items-center justify-center py-12">
              <Loader2 size={32} className="text-blue-400 animate-spin mb-3" />
              <p className="text-slate-400 text-sm">Lade Backups...</p>
            </div>
          ) : (
            <div className="overflow-x-auto -mx-6 px-6">
              <table className="w-full text-left border-collapse">
                <thead>
                  <tr className="text-xs text-slate-500 border-b border-white/5">
                    <th className="py-3 px-2 font-medium uppercase tracking-wider">Snapshot Name</th>
                    <th className="py-3 px-2 font-medium uppercase tracking-wider">Datum</th>
                    <th className="py-3 px-2 font-medium uppercase tracking-wider">Größe</th>
                    <th className="py-3 px-2 font-medium uppercase tracking-wider text-right">Aktionen</th>
                  </tr>
                </thead>
                <tbody className="text-sm">
                  {backups.map((b) => (
                    <tr
                      key={b.id}
                      className="group border-b border-white/5 last:border-0 hover:bg-white/5 transition-colors"
                    >
                      <td className="py-4 px-2">
                        <div className="flex items-center gap-3">
                          <div className="p-2 rounded-lg bg-slate-800 text-blue-400 group-hover:bg-blue-500/20 group-hover:border-blue-500/30 group-hover:shadow-[0_0_15px_rgba(59,130,246,0.15)] transition-all border border-white/5">
                            <Archive size={16} />
                          </div>
                          <span className="font-medium text-white">
                            {b.name || b.id}
                          </span>
                        </div>
                      </td>
                      <td className="py-4 px-2 text-slate-400 font-mono text-xs">
                        {new Date(b.modTime || b.created_at).toLocaleString()}
                      </td>
                      <td className="py-4 px-2 text-slate-400 font-mono text-xs">
                        {formatSize(b.size)}
                      </td>
                      <td className="py-4 px-2 text-right">
                        <div className="flex items-center justify-end gap-2">
                          <button
                            onClick={() => restoreBackup(b.id)}
                            disabled={busy}
                            title="System wiederherstellen"
                            className="p-2 rounded-lg bg-emerald-500/10 hover:bg-emerald-500/20 text-emerald-400 border border-emerald-500/20 transition-all disabled:opacity-50 disabled:cursor-not-allowed"
                          >
                            {processingId === b.id ? (
                              <Loader2 size={14} className="animate-spin" />
                            ) : (
                              <RefreshCw size={14} />
                            )}
                          </button>
                          <button
                            onClick={() => deleteBackup(b.id)}
                            disabled={busy}
                            title="Backup löschen"
                            className="p-2 rounded-lg bg-rose-500/10 hover:bg-rose-500/20 text-rose-400 border border-rose-500/20 transition-all disabled:opacity-50 disabled:cursor-not-allowed"
                          >
                            <Trash2 size={14} />
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                  {backups.length === 0 && (
                    <tr>
                      <td colSpan="4" className="py-16 text-center">
                        <div className="flex flex-col items-center justify-center text-slate-500">
                          <div className="p-4 bg-slate-800/50 rounded-full mb-4">
                            <HardDrive size={48} className="opacity-30" />
                          </div>
                          <p className="text-lg font-medium text-slate-400">Keine Backups gefunden</p>
                          <p className="text-sm mt-1 text-slate-500">Erstellen Sie Ihren ersten Snapshot oben rechts.</p>
                        </div>
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </GlassCard>

      {/* Info Footer */}
      <div className="flex items-start gap-3 p-4 rounded-xl bg-blue-500/5 border border-blue-500/10">
        <AlertTriangle size={18} className="shrink-0 mt-0.5 text-blue-400" />
        <p className="text-sm text-slate-300">
          Backups beinhalten alle Dateien aus dem <code className="px-1.5 py-0.5 bg-slate-800 rounded text-blue-400 font-mono text-xs">{backupPath}</code> Verzeichnis.
          Die Wiederherstellung eines Snapshots überschreibt alle aktuellen Dateien unwiderruflich.
        </p>
      </div>
    </div>
  );
}
