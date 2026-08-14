import { useState } from "react";
import { Lock, AlertTriangle, Shield, Copy, Download, Check } from "lucide-react";
import { generateRecoveryKey } from "../../lib/crypto"; // Keep only recovery key gen

export function VaultModal({ isOpen, onClose, onUnlock, onSetup, hasVault }) {
    const [mode, setMode] = useState("unlock"); // unlock, setup, recovery

    // Reset mode based on vault existence when opening
    if (isOpen && mode === "unlock" && !hasVault) {
        setMode("setup");
    } else if (isOpen && mode === "setup" && hasVault) {
        setMode("unlock");
    }
    const [password, setPassword] = useState("");
    const [confirmPassword, setConfirmPassword] = useState("");
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState("");
    const [recoveryKey, setRecoveryKey] = useState("");
    const [recoveryConfirmed, setRecoveryConfirmed] = useState(false);

    // Constants
    const MIN_PASS_LENGTH = 12;

    const handleUnlock = async (e) => {
        e.preventDefault();
        setLoading(true);
        setError("");

        try {
            await onUnlock(password);
            onClose();
        } catch (err) {
            setError(err.message || "Entsperren fehlgeschlagen");
        } finally {
            setLoading(false);
        }
    };

    const handleSetupStep1 = async (e) => {
        e.preventDefault();

        if (password !== confirmPassword) {
            setError("Passwörter stimmen nicht überein");
            return;
        }
        if (password.length < MIN_PASS_LENGTH) {
            setError(`Passwort muss mind. ${MIN_PASS_LENGTH} Zeichen lang sein`);
            return;
        }

        // Generate Recovery Key
        const key = generateRecoveryKey();
        setRecoveryKey(key);
        setMode("recovery_confirm");
    };

    const handleSetupFinalize = async () => {
        if (!recoveryConfirmed) return;
        setLoading(true);

        try {
            // Pass password directly to backend setup
            await onSetup(password);
            onClose();
        } catch (err) {
            setError("Setup failed: " + (err.message || "Server Error"));
        } finally {
            setLoading(false);
        }
    };

    const copyToClipboard = () => {
        navigator.clipboard.writeText(recoveryKey);
        // Show toast or tick
    };

    const downloadRecoveryKey = () => {
        const blob = new Blob([
            `NAS.AI VAULT RECOVERY KEY\n\nKEY: ${recoveryKey}\n\nKeep this safe! Without this key and your password, your data is lost forever.\nGenerated: ${new Date().toISOString()}`
        ], { type: "text/plain" });
        const url = URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download = "nas-vault-recovery-key.txt";
        a.click();
        URL.revokeObjectURL(url);
    };

    if (!isOpen) return null;

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-sm p-4">
            <div className="w-full max-w-md bg-slate-900 border border-white/10 rounded-2xl shadow-2xl overflow-hidden">

                {/* Header */}
                <div className="p-6 bg-slate-800/50 border-b border-white/5 text-center">
                    <div className="inline-flex items-center justify-center w-12 h-12 rounded-xl bg-amber-500/20 mb-3">
                        <Lock size={24} className="text-amber-400" />
                    </div>
                    <h2 className="text-xl font-bold text-white">
                        {mode === "unlock" ? "Secure Vault" : "Vault Setup"}
                    </h2>
                </div>

                {/* Content */}
                <div className="p-6 space-y-4">
                    {error && (
                        <div className="p-3 bg-rose-500/10 border border-rose-500/20 rounded-lg flex items-center gap-2 text-rose-400 text-sm">
                            <AlertTriangle size={16} />
                            {error}
                        </div>
                    )}

                    {mode === "unlock" && (
                        <form onSubmit={handleUnlock} className="space-y-4">
                            <div>
                                <label className="block text-sm text-slate-400 mb-1">Vault Passwort</label>
                                <input
                                    type="password"
                                    value={password}
                                    onChange={e => setPassword(e.target.value)}
                                    className="w-full px-4 py-3 bg-black/20 border border-white/10 rounded-xl text-white focus:outline-none focus:border-amber-500/50"
                                    autoFocus
                                />
                            </div>
                            <button
                                type="submit"
                                disabled={loading}
                                className="w-full py-3 bg-amber-500 hover:bg-amber-600 text-black font-bold rounded-xl transition-colors disabled:opacity-50"
                            >
                                {loading ? "Entschlüssele..." : "Entsperren"}
                            </button>
                            <button
                                type="button"
                                onClick={() => setMode("setup")}
                                className="w-full text-xs text-slate-500 hover:text-slate-300"
                            >
                                Erstmals einrichten?
                            </button>
                        </form>
                    )}

                    {mode === "setup" && (
                        <form onSubmit={handleSetupStep1} className="space-y-4">
                            <div>
                                <label className="block text-sm text-slate-400 mb-1">Neues Passwort</label>
                                <input
                                    type="password"
                                    value={password}
                                    onChange={e => setPassword(e.target.value)}
                                    className="w-full px-4 py-3 bg-black/20 border border-white/10 rounded-xl text-white focus:outline-none focus:border-amber-500/50"
                                />
                            </div>
                            <div>
                                <label className="block text-sm text-slate-400 mb-1">Bestätigen</label>
                                <input
                                    type="password"
                                    value={confirmPassword}
                                    onChange={e => setConfirmPassword(e.target.value)}
                                    className="w-full px-4 py-3 bg-black/20 border border-white/10 rounded-xl text-white focus:outline-none focus:border-amber-500/50"
                                />
                            </div>
                            <div className="text-xs text-slate-500 bg-slate-800/50 p-3 rounded-lg">
                                <AlertTriangle size={12} className="inline mr-1" />
                                Wenn Sie dieses Passwort vergessen, sind Ihre Daten unwiederbringlich verloren. Es gibt keinen Reset per E-Mail.
                            </div>
                            <button
                                type="submit"
                                className="w-full py-3 bg-amber-500 hover:bg-amber-600 text-black font-bold rounded-xl transition-colors"
                            >
                                Weiter
                            </button>
                        </form>
                    )}

                    {mode === "recovery_confirm" && (
                        <div className="space-y-6">
                            <div className="text-center">
                                <Shield size={32} className="mx-auto text-emerald-400 mb-2" />
                                <h3 className="text-lg font-bold text-white mb-1">Recovery Key Sichern</h3>
                                <p className="text-sm text-slate-400">
                                    Dies ist Ihr einziger Notfallschlüssel. Drucken Sie ihn aus oder speichern Sie ihn sicher (NICHT auf diesem Gerät).
                                </p>
                            </div>

                            <div className="bg-black/40 p-4 rounded-xl border border-amber-500/30 font-mono text-center text-amber-400 text-lg break-all select-all">
                                {recoveryKey}
                            </div>

                            <div className="flex gap-2 justify-center">
                                <button
                                    onClick={copyToClipboard}
                                    className="p-2 hover:bg-white/10 rounded-lg text-slate-400 hover:text-white transition-colors"
                                    title="Kopieren"
                                >
                                    <Copy size={20} />
                                </button>
                                <button
                                    onClick={downloadRecoveryKey}
                                    className="p-2 hover:bg-white/10 rounded-lg text-slate-400 hover:text-white transition-colors"
                                    title="Herunterladen"
                                >
                                    <Download size={20} />
                                </button>
                            </div>

                            <div className="flex items-start gap-3 p-3 bg-slate-800/50 rounded-lg cursor-pointer max-w-sm mx-auto" onClick={() => setRecoveryConfirmed(!recoveryConfirmed)}>
                                <div className={`mt-0.5 w-5 h-5 rounded border flex items-center justify-center ${recoveryConfirmed ? 'bg-emerald-500 border-emerald-500' : 'border-slate-500'}`}>
                                    {recoveryConfirmed && <Check size={14} className="text-black" />}
                                </div>
                                <p className="text-xs text-slate-300">
                                    Ich habe den Recovery Key gespeichert und verstanden, dass der Support meine Daten nicht wiederherstellen kann.
                                </p>
                            </div>

                            <button
                                onClick={handleSetupFinalize}
                                disabled={!recoveryConfirmed || loading}
                                className="w-full py-3 bg-emerald-500 hover:bg-emerald-600 text-black font-bold rounded-xl transition-colors disabled:opacity-50 disabled:grayscale"
                            >
                                {loading ? "Richte ein..." : "Vault Erstellen"}
                            </button>
                        </div>
                    )}

                </div>
            </div>
        </div>
    );
}
