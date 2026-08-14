import { useState, useEffect } from "react";
import { useNavigate, Link } from "react-router-dom";
import { setAuth } from "../utils/auth";
import { loginWithWebAuthn, loginWithRecoveryCode } from "../lib/webauthn";
import { Mail, Lock, Loader2, LogIn, CloudLightning, ArrowRight, ShieldCheck, KeyRound } from "lucide-react";

const API_BASE =
  import.meta.env.VITE_API_BASE_URL ||
  window.location.origin;

export default function Login() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [csrfToken, setCsrfToken] = useState("");

  // Second-factor (MFA) state
  const [mfaRequired, setMfaRequired] = useState(false);
  const [mfaToken, setMfaToken] = useState("");
  const [mfaMethods, setMfaMethods] = useState([]);
  const [recoveryMode, setRecoveryMode] = useState(false);
  const [recoveryCode, setRecoveryCode] = useState("");

  const navigate = useNavigate();

  // Fetch CSRF Token on mount
  useEffect(() => {
    const fetchCsrf = async () => {
      try {
        const res = await fetch(`${API_BASE}/api/v1/auth/csrf`, {
          credentials: "include",
        });
        if (res.ok) {
          const data = await res.json();
          setCsrfToken(data.csrf_token || "");
        }
      } catch (err) {
        console.error("CSRF fetch failed:", err);
      }
    };
    fetchCsrf();
  }, []);

  // Finalize a successful login (WebAuthn finish or recovery-code login return
  // the same shape as the direct login response).
  const completeLogin = (data) => {
    setAuth({
      accessToken: data.access_token || "",
      refreshToken: data.refresh_token || "",
      csrfToken: data.csrf_token || csrfToken,
    });
    navigate("/dashboard", { replace: true });
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setLoading(true);
    setError("");

    try {
      const res = await fetch(`${API_BASE}/auth/login`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          ...(csrfToken && { "X-CSRF-Token": csrfToken }),
        },
        credentials: "include",
        body: JSON.stringify({ email, password }),
      });

      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data?.error?.message || "Login fehlgeschlagen");
      }

      const data = await res.json();

      // Second factor required: switch to the MFA step and start the WebAuthn
      // (YubiKey) ceremony. The user can fall back to a recovery code.
      if (data?.mfa_required) {
        setError("");
        setMfaToken(data.mfa_token);
        setMfaMethods(data.methods || []);
        setMfaRequired(true);
        await runWebAuthn(data.mfa_token);
        return;
      }

      completeLogin(data);
    } catch (err) {
      setError(err.message || "Login fehlgeschlagen");
    } finally {
      setLoading(false);
    }
  };

  // Drive the WebAuthn second factor. Kept separate so the user can retry it
  // from the MFA step without re-entering their password.
  const runWebAuthn = async (token) => {
    setLoading(true);
    setError("");
    try {
      const data = await loginWithWebAuthn(token, csrfToken);
      completeLogin(data);
    } catch (err) {
      setError(err.message || "Anmeldung mit Sicherheitsschlüssel fehlgeschlagen");
    } finally {
      setLoading(false);
    }
  };

  const handleRecoverySubmit = async (e) => {
    e.preventDefault();
    setLoading(true);
    setError("");
    try {
      const data = await loginWithRecoveryCode(mfaToken, recoveryCode.trim());
      completeLogin(data);
    } catch (err) {
      setError(err.message || "Recovery-Code ungültig");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-[#0a0a0c] text-slate-200 font-sans flex items-center justify-center p-4 relative overflow-hidden">

      {/* Animated Background Blobs */}
      <div className="fixed inset-0 z-0 pointer-events-none overflow-hidden">
        <div className="absolute top-[-10%] left-[-10%] w-[500px] h-[500px] bg-blue-600/20 rounded-full blur-[120px] animate-pulse-glow"></div>
        <div className="absolute bottom-[-10%] right-[-5%] w-[600px] h-[600px] bg-violet-600/10 rounded-full blur-[130px]"></div>
        <div className="absolute top-[40%] left-[30%] w-[300px] h-[300px] bg-cyan-500/10 rounded-full blur-[100px] opacity-60"></div>
      </div>

      {/* Login Card */}
      <div className="relative z-10 w-full max-w-md">

        {/* Glass Card */}
        <div className="relative overflow-hidden rounded-2xl border border-white/10 bg-slate-900/40 backdrop-blur-xl shadow-2xl">
          <div className="absolute top-0 left-0 w-full h-[1px] bg-gradient-to-r from-transparent via-white/20 to-transparent opacity-50"></div>

          <div className="p-8">

            {/* Header */}
            <div className="text-center mb-8">
              <div className="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-gradient-to-br from-blue-500 to-violet-600 mb-4 shadow-lg shadow-blue-500/30">
                <CloudLightning size={32} className="text-white" />
              </div>
              <h1 className="text-3xl font-bold text-white tracking-tight mb-2">
                {mfaRequired ? "Zweiter Faktor" : "Willkommen zurück"}
              </h1>
              <p className="text-slate-400 text-sm">
                {mfaRequired
                  ? "Bestätigen Sie die Anmeldung mit Ihrem Sicherheitsschlüssel"
                  : "Melden Sie sich an, um auf Ihr System zuzugreifen"}
              </p>
            </div>

            {/* Error Message */}
            {error && (
              <div className="mb-6 p-4 rounded-xl bg-rose-500/10 border border-rose-500/30 animate-in fade-in duration-300">
                <p className="text-rose-400 text-sm font-medium flex items-center gap-2">
                  <ShieldCheck size={16} className="text-rose-400" />
                  {error}
                </p>
              </div>
            )}

            {/* Login Form */}
            {!mfaRequired && (
            <form onSubmit={handleSubmit} className="space-y-5">

              {/* Email Input */}
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-2">
                  E-Mail Adresse
                </label>
                <div className="relative">
                  <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                    <Mail size={18} className="text-slate-400" />
                  </div>
                  <input
                    type="email"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    required
                    placeholder="name@beispiel.de"
                    className="w-full pl-10 pr-4 py-3 bg-slate-900/50 border border-white/10 rounded-xl text-white placeholder:text-slate-500 focus:border-blue-500/50 focus:ring-2 focus:ring-blue-500/20 focus:outline-none transition-all"
                  />
                </div>
              </div>

              {/* Password Input */}
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-2">
                  Passwort
                </label>
                <div className="relative">
                  <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                    <Lock size={18} className="text-slate-400" />
                  </div>
                  <input
                    type="password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    required
                    placeholder="••••••••"
                    className="w-full pl-10 pr-4 py-3 bg-slate-900/50 border border-white/10 rounded-xl text-white placeholder:text-slate-500 focus:border-blue-500/50 focus:ring-2 focus:ring-blue-500/20 focus:outline-none transition-all"
                  />
                </div>
              </div>

              {/* Submit Button */}
              <button
                type="submit"
                disabled={loading}
                className="w-full flex items-center justify-center gap-2 px-6 py-3 bg-blue-500/20 hover:bg-blue-500/30 text-blue-400 rounded-xl font-medium transition-all shadow-[0_0_20px_rgba(59,130,246,0.3)] hover:shadow-[0_0_30px_rgba(59,130,246,0.5)] disabled:opacity-50 disabled:cursor-not-allowed border border-blue-500/30 mt-6"
              >
                {loading ? (
                  <>
                    <Loader2 size={20} className="animate-spin" />
                    <span>Anmeldung läuft...</span>
                  </>
                ) : (
                  <>
                    <LogIn size={20} />
                    <span>Anmelden</span>
                  </>
                )}
              </button>
            </form>
            )}

            {/* MFA: WebAuthn (Security Key) step */}
            {mfaRequired && !recoveryMode && (
              <div className="space-y-5">
                <div className="flex flex-col items-center text-center gap-3 py-2">
                  <div className="inline-flex items-center justify-center w-14 h-14 rounded-2xl bg-blue-500/10 border border-blue-500/30">
                    <ShieldCheck size={26} className="text-blue-400" />
                  </div>
                  <p className="text-slate-400 text-sm">
                    Berühren Sie Ihren Sicherheitsschlüssel, um die Anmeldung abzuschließen.
                  </p>
                </div>

                <button
                  type="button"
                  onClick={() => runWebAuthn(mfaToken)}
                  disabled={loading}
                  className="w-full flex items-center justify-center gap-2 px-6 py-3 bg-blue-500/20 hover:bg-blue-500/30 text-blue-400 rounded-xl font-medium transition-all shadow-[0_0_20px_rgba(59,130,246,0.3)] hover:shadow-[0_0_30px_rgba(59,130,246,0.5)] disabled:opacity-50 disabled:cursor-not-allowed border border-blue-500/30"
                >
                  {loading ? (
                    <>
                      <Loader2 size={20} className="animate-spin" />
                      <span>Warte auf Sicherheitsschlüssel...</span>
                    </>
                  ) : (
                    <>
                      <ShieldCheck size={20} />
                      <span>Erneut mit Sicherheitsschlüssel</span>
                    </>
                  )}
                </button>

                <button
                  type="button"
                  onClick={() => {
                    setError("");
                    setRecoveryMode(true);
                  }}
                  className="w-full flex items-center justify-center gap-2 text-sm text-slate-400 hover:text-blue-300 font-medium transition-colors"
                >
                  <KeyRound size={16} />
                  Mit Recovery-Code anmelden
                </button>
              </div>
            )}

            {/* MFA: Recovery code fallback */}
            {mfaRequired && recoveryMode && (
              <form onSubmit={handleRecoverySubmit} className="space-y-5">
                <div>
                  <label className="block text-sm font-medium text-slate-300 mb-2">
                    Recovery-Code
                  </label>
                  <div className="relative">
                    <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                      <KeyRound size={18} className="text-slate-400" />
                    </div>
                    <input
                      type="text"
                      value={recoveryCode}
                      onChange={(e) => setRecoveryCode(e.target.value)}
                      required
                      autoComplete="one-time-code"
                      autoFocus
                      spellCheck={false}
                      placeholder="ABCDE-FGHJK"
                      className="w-full pl-10 pr-4 py-3 bg-slate-900/50 border border-white/10 rounded-xl text-white placeholder:text-slate-500 font-mono tracking-wider focus:border-blue-500/50 focus:ring-2 focus:ring-blue-500/20 focus:outline-none transition-all"
                    />
                  </div>
                  <p className="text-slate-500 text-xs mt-2">
                    Verwenden Sie einen Ihrer gespeicherten Backup-Codes. Jeder Code funktioniert nur einmal.
                  </p>
                </div>

                <button
                  type="submit"
                  disabled={loading || !recoveryCode.trim()}
                  className="w-full flex items-center justify-center gap-2 px-6 py-3 bg-blue-500/20 hover:bg-blue-500/30 text-blue-400 rounded-xl font-medium transition-all shadow-[0_0_20px_rgba(59,130,246,0.3)] hover:shadow-[0_0_30px_rgba(59,130,246,0.5)] disabled:opacity-50 disabled:cursor-not-allowed border border-blue-500/30"
                >
                  {loading ? (
                    <>
                      <Loader2 size={20} className="animate-spin" />
                      <span>Anmeldung läuft...</span>
                    </>
                  ) : (
                    <>
                      <LogIn size={20} />
                      <span>Anmelden</span>
                    </>
                  )}
                </button>

                <button
                  type="button"
                  onClick={() => {
                    setError("");
                    setRecoveryMode(false);
                  }}
                  className="w-full flex items-center justify-center gap-2 text-sm text-slate-400 hover:text-blue-300 font-medium transition-colors"
                >
                  <ShieldCheck size={16} />
                  Zurück zum Sicherheitsschlüssel
                </button>
              </form>
            )}

            {/* Register Link */}
            {!mfaRequired && (
            <div className="mt-6 pt-6 border-t border-white/5">
              <p className="text-center text-sm text-slate-400">
                Noch keinen Account?{" "}
                <Link
                  to="/register"
                  className="text-blue-400 hover:text-blue-300 font-medium inline-flex items-center gap-1 group transition-colors"
                >
                  Jetzt registrieren
                  <ArrowRight size={14} className="group-hover:translate-x-1 transition-transform" />
                </Link>
              </p>
            </div>
            )}
          </div>
        </div>

        {/* Footer Info */}
        <div className="mt-6 text-center">
          <p className="text-xs text-slate-500">
            Geschützte Verbindung · Ende-zu-Ende verschlüsselt
          </p>
        </div>
      </div>
    </div>
  );
}
