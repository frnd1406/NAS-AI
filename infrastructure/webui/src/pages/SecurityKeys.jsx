import { useEffect, useState, useCallback } from "react";
import { getCSRFToken } from "../utils/auth";
import {
  registerCredential,
  listCredentials,
  deleteCredential,
  isWebAuthnSupported,
  getRecoveryCodesStatus,
  generateRecoveryCodes,
} from "../lib/webauthn";
import {
  KeyRound,
  Plus,
  Trash2,
  Loader2,
  ShieldCheck,
  LifeBuoy,
  Copy,
  Download,
  RefreshCw,
  AlertTriangle,
} from "lucide-react";

export default function SecurityKeys() {
  const [keys, setKeys] = useState([]);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [name, setName] = useState("");

  // Recovery codes (fallback second factor when the authenticator is lost).
  const [recoveryRemaining, setRecoveryRemaining] = useState(null);
  const [recoveryCodes, setRecoveryCodes] = useState(null); // set once, right after generation
  const [recoveryLoading, setRecoveryLoading] = useState(true);
  const [recoveryBusy, setRecoveryBusy] = useState(false);
  const [recoveryError, setRecoveryError] = useState("");
  const [copied, setCopied] = useState(false);

  const supported = isWebAuthnSupported();

  const refresh = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      setKeys(await listCredentials());
    } catch (err) {
      setError(err.message || "Fehler beim Laden");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const refreshRecovery = useCallback(async () => {
    setRecoveryLoading(true);
    setRecoveryError("");
    try {
      const status = await getRecoveryCodesStatus();
      setRecoveryRemaining(status.remaining ?? 0);
    } catch (err) {
      setRecoveryError(err.message || "Fehler beim Laden der Recovery-Codes");
    } finally {
      setRecoveryLoading(false);
    }
  }, []);

  useEffect(() => {
    refreshRecovery();
  }, [refreshRecovery]);

  const handleGenerateRecovery = async () => {
    if (
      recoveryRemaining > 0 &&
      !window.confirm(
        "Neue Recovery-Codes erzeugen? Deine bisherigen Codes werden dadurch ungültig."
      )
    ) {
      return;
    }
    setRecoveryBusy(true);
    setRecoveryError("");
    setCopied(false);
    try {
      const res = await generateRecoveryCodes(getCSRFToken());
      const codes = res.codes || [];
      setRecoveryCodes(codes);
      setRecoveryRemaining(res.count ?? codes.length);
    } catch (err) {
      setRecoveryError(err.message || "Recovery-Codes konnten nicht erzeugt werden");
    } finally {
      setRecoveryBusy(false);
    }
  };

  const copyRecoveryCodes = async () => {
    if (!recoveryCodes) return;
    try {
      await navigator.clipboard.writeText(recoveryCodes.join("\n"));
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      setRecoveryError("Kopieren nicht möglich – bitte Codes manuell markieren.");
    }
  };

  const downloadRecoveryCodes = () => {
    if (!recoveryCodes) return;
    const blob = new Blob(
      [
        "NAS.AI – Recovery-Codes\n",
        "Jeder Code ist genau einmal verwendbar. Sicher aufbewahren.\n\n",
        recoveryCodes.join("\n"),
        "\n",
      ],
      { type: "text/plain" }
    );
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "nas-ai-recovery-codes.txt";
    a.click();
    URL.revokeObjectURL(url);
  };

  const handleRegister = async (e) => {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      await registerCredential(name.trim() || "Security Key", getCSRFToken());
      setName("");
      await refresh();
    } catch (err) {
      setError(err.message || "Registrierung fehlgeschlagen");
    } finally {
      setBusy(false);
    }
  };

  const handleDelete = async (id) => {
    setBusy(true);
    setError("");
    try {
      await deleteCredential(id, getCSRFToken());
      await refresh();
    } catch (err) {
      setError(err.message || "Löschen fehlgeschlagen");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="max-w-2xl mx-auto p-6 space-y-6">
      <div className="flex items-center gap-3">
        <div className="inline-flex items-center justify-center w-10 h-10 rounded-xl bg-gradient-to-br from-blue-500 to-violet-600">
          <ShieldCheck size={20} className="text-white" />
        </div>
        <div>
          <h1 className="text-xl font-bold text-white">Sicherheitsschlüssel</h1>
          <p className="text-sm text-slate-400">
            YubiKey / FIDO2 als zweiten Faktor für die Anmeldung verwalten
          </p>
        </div>
      </div>

      {!supported && (
        <div className="p-4 rounded-xl bg-amber-500/10 border border-amber-500/30 text-amber-300 text-sm">
          Dieser Browser unterstützt WebAuthn nicht.
        </div>
      )}

      {error && (
        <div className="p-4 rounded-xl bg-rose-500/10 border border-rose-500/30 text-rose-400 text-sm">
          {error}
        </div>
      )}

      <form
        onSubmit={handleRegister}
        className="flex items-end gap-3 p-4 rounded-xl border border-white/10 bg-slate-900/40"
      >
        <div className="flex-1">
          <label className="block text-sm text-slate-300 mb-2">Name (optional)</label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="z. B. YubiKey 5C"
            className="w-full px-3 py-2 bg-slate-900/50 border border-white/10 rounded-lg text-white placeholder:text-slate-500 focus:border-blue-500/50 focus:outline-none"
          />
        </div>
        <button
          type="submit"
          disabled={busy || !supported}
          className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 disabled:opacity-50 text-white text-sm font-medium"
        >
          {busy ? <Loader2 size={16} className="animate-spin" /> : <Plus size={16} />}
          Schlüssel hinzufügen
        </button>
      </form>

      <div className="rounded-xl border border-white/10 bg-slate-900/40 divide-y divide-white/5">
        {loading ? (
          <div className="p-6 flex items-center justify-center text-slate-400">
            <Loader2 size={18} className="animate-spin" />
          </div>
        ) : keys.length === 0 ? (
          <div className="p-6 text-center text-slate-400 text-sm">
            Noch kein Sicherheitsschlüssel registriert.
          </div>
        ) : (
          keys.map((k) => (
            <div key={k.id} className="flex items-center justify-between p-4">
              <div className="flex items-center gap-3">
                <KeyRound size={18} className="text-blue-400" />
                <div>
                  <p className="text-white text-sm font-medium">{k.name || "Security Key"}</p>
                  <p className="text-slate-500 text-xs">
                    Hinzugefügt: {new Date(k.created_at).toLocaleString()}
                    {k.last_used_at && ` · Zuletzt: ${new Date(k.last_used_at).toLocaleString()}`}
                  </p>
                </div>
              </div>
              <button
                onClick={() => handleDelete(k.id)}
                disabled={busy}
                className="inline-flex items-center gap-1 px-3 py-1.5 rounded-lg bg-rose-500/10 hover:bg-rose-500/20 text-rose-400 text-sm disabled:opacity-50"
              >
                <Trash2 size={14} />
                Entfernen
              </button>
            </div>
          ))
        )}
      </div>

      {/* Recovery codes: fallback second factor for a lost authenticator */}
      <div className="rounded-xl border border-white/10 bg-slate-900/40 p-4 space-y-4">
        <div className="flex items-center justify-between gap-3">
          <div className="flex items-center gap-3">
            <LifeBuoy size={18} className="text-emerald-400" />
            <div>
              <p className="text-white text-sm font-medium">Recovery-Codes</p>
              <p className="text-slate-500 text-xs">
                Einmal-Codes, falls dein Sicherheitsschlüssel verloren geht
              </p>
            </div>
          </div>
          <button
            onClick={handleGenerateRecovery}
            disabled={recoveryBusy}
            className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 text-white text-sm font-medium"
          >
            {recoveryBusy ? (
              <Loader2 size={16} className="animate-spin" />
            ) : (
              <RefreshCw size={16} />
            )}
            {recoveryRemaining > 0 ? "Neu generieren" : "Codes erzeugen"}
          </button>
        </div>

        {recoveryError && (
          <div className="p-3 rounded-lg bg-rose-500/10 border border-rose-500/30 text-rose-400 text-sm">
            {recoveryError}
          </div>
        )}

        {!recoveryCodes && (
          <p className="text-sm text-slate-400">
            {recoveryLoading
              ? "Lädt …"
              : recoveryRemaining > 0
              ? `Noch ${recoveryRemaining} unbenutzte${
                  recoveryRemaining === 1 ? "r Code" : " Codes"
                } übrig.`
              : "Noch keine Recovery-Codes vorhanden. Erzeuge welche, um dich bei Verlust des Schlüssels nicht auszusperren."}
          </p>
        )}

        {recoveryCodes && (
          <div className="space-y-3">
            <div className="flex items-start gap-2 p-3 rounded-lg bg-amber-500/10 border border-amber-500/30 text-amber-300 text-sm">
              <AlertTriangle size={16} className="mt-0.5 shrink-0" />
              <span>
                Speichere diese Codes jetzt – sie werden{" "}
                <strong>nur dieses eine Mal</strong> angezeigt. Jeder Code ist
                einmal verwendbar; frühere Codes sind ab jetzt ungültig.
              </span>
            </div>
            <div className="grid grid-cols-2 gap-2 p-4 rounded-lg bg-slate-950/60 border border-white/10 font-mono text-sm text-slate-200">
              {recoveryCodes.map((code) => (
                <div key={code} className="tracking-widest">
                  {code}
                </div>
              ))}
            </div>
            <div className="flex gap-2">
              <button
                onClick={copyRecoveryCodes}
                className="inline-flex items-center gap-2 px-3 py-1.5 rounded-lg bg-slate-700/50 hover:bg-slate-700 text-slate-200 text-sm"
              >
                <Copy size={14} />
                {copied ? "Kopiert!" : "Kopieren"}
              </button>
              <button
                onClick={downloadRecoveryCodes}
                className="inline-flex items-center gap-2 px-3 py-1.5 rounded-lg bg-slate-700/50 hover:bg-slate-700 text-slate-200 text-sm"
              >
                <Download size={14} />
                Herunterladen
              </button>
              <button
                onClick={() => setRecoveryCodes(null)}
                className="inline-flex items-center gap-2 px-3 py-1.5 rounded-lg bg-slate-700/50 hover:bg-slate-700 text-slate-200 text-sm ml-auto"
              >
                Fertig
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
