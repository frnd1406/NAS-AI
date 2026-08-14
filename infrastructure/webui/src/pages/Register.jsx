import { useState, useEffect } from "react";
import { useNavigate, Link } from "react-router-dom";
import { setAuth } from "../utils/auth";
import {
  Mail,
  Lock,
  User,
  Loader2,
  UserPlus,
  CloudLightning,
  ArrowRight,
  ShieldCheck,
  CheckCircle,
  XCircle,
} from "lucide-react";

const API_BASE =
  import.meta.env.VITE_API_BASE_URL ||
  window.location.origin;

export default function Register() {
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [csrfToken, setCsrfToken] = useState("");
  const navigate = useNavigate();

  // Password validation state
  const [passwordStrength, setPasswordStrength] = useState({
    isLongEnough: false,
    hasNumber: false,
    hasSpecialChar: false,
  });

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

  // Validate password strength
  useEffect(() => {
    setPasswordStrength({
      isLongEnough: password.length >= 8,
      hasNumber: /\d/.test(password),
      hasSpecialChar: /[!@#$%^&*(),.?":{}|<>]/.test(password),
    });
  }, [password]);

  const isPasswordStrong =
    passwordStrength.isLongEnough &&
    passwordStrength.hasNumber &&
    passwordStrength.hasSpecialChar;

  const handleSubmit = async (e) => {
    e.preventDefault();
    setLoading(true);
    setError("");

    // Validate passwords match
    if (password !== confirmPassword) {
      setError("Passwörter stimmen nicht überein");
      setLoading(false);
      return;
    }

    // Validate password strength
    if (!isPasswordStrong) {
      setError("Passwort erfüllt nicht die Sicherheitsanforderungen");
      setLoading(false);
      return;
    }

    try {
      const res = await fetch(`${API_BASE}/auth/register`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          ...(csrfToken && { "X-CSRF-Token": csrfToken }),
        },
        credentials: "include",
        body: JSON.stringify({ username: name, email, password }),
      });

      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data?.error?.message || "Registrierung fehlgeschlagen");
      }

      const data = await res.json();
      setAuth({
        accessToken: data.access_token || "",
        refreshToken: data.refresh_token || "",
        csrfToken: data.csrf_token || csrfToken,
      });

      // Redirect to dashboard
      navigate("/dashboard", { replace: true });
    } catch (err) {
      setError(err.message || "Registrierung fehlgeschlagen");
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

      {/* Register Card */}
      <div className="relative z-10 w-full max-w-md">

        {/* Glass Card */}
        <div className="relative overflow-hidden rounded-2xl border border-white/10 bg-slate-900/40 backdrop-blur-xl shadow-2xl">
          <div className="absolute top-0 left-0 w-full h-[1px] bg-gradient-to-r from-transparent via-white/20 to-transparent opacity-50"></div>

          <div className="p-8">

            {/* Header */}
            <div className="text-center mb-8">
              <div className="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-gradient-to-br from-emerald-500 to-blue-600 mb-4 shadow-lg shadow-emerald-500/30">
                <CloudLightning size={32} className="text-white" />
              </div>
              <h1 className="text-3xl font-bold text-white tracking-tight mb-2">
                Account erstellen
              </h1>
              <p className="text-slate-400 text-sm">
                Werden Sie Teil des NAS.AI Systems
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

            {/* Register Form */}
            <form onSubmit={handleSubmit} className="space-y-4">

              {/* Name Input */}
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-2">
                  Name
                </label>
                <div className="relative">
                  <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                    <User size={18} className="text-slate-400" />
                  </div>
                  <input
                    type="text"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    required
                    placeholder="Max Mustermann"
                    className="w-full pl-10 pr-4 py-3 bg-slate-900/50 border border-white/10 rounded-xl text-white placeholder:text-slate-500 focus:border-blue-500/50 focus:ring-2 focus:ring-blue-500/20 focus:outline-none transition-all"
                  />
                </div>
              </div>

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

                {/* Password Strength Indicators */}
                {password && (
                  <div className="mt-3 space-y-2">
                    <div className="flex items-center gap-2 text-xs">
                      {passwordStrength.isLongEnough ? (
                        <CheckCircle size={14} className="text-emerald-400" />
                      ) : (
                        <XCircle size={14} className="text-slate-500" />
                      )}
                      <span className={passwordStrength.isLongEnough ? "text-emerald-400" : "text-slate-500"}>
                        Mindestens 8 Zeichen
                      </span>
                    </div>
                    <div className="flex items-center gap-2 text-xs">
                      {passwordStrength.hasNumber ? (
                        <CheckCircle size={14} className="text-emerald-400" />
                      ) : (
                        <XCircle size={14} className="text-slate-500" />
                      )}
                      <span className={passwordStrength.hasNumber ? "text-emerald-400" : "text-slate-500"}>
                        Enthält mindestens eine Zahl
                      </span>
                    </div>
                    <div className="flex items-center gap-2 text-xs">
                      {passwordStrength.hasSpecialChar ? (
                        <CheckCircle size={14} className="text-emerald-400" />
                      ) : (
                        <XCircle size={14} className="text-slate-500" />
                      )}
                      <span className={passwordStrength.hasSpecialChar ? "text-emerald-400" : "text-slate-500"}>
                        Enthält Sonderzeichen
                      </span>
                    </div>
                  </div>
                )}
              </div>

              {/* Confirm Password Input */}
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-2">
                  Passwort bestätigen
                </label>
                <div className="relative">
                  <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                    <Lock size={18} className="text-slate-400" />
                  </div>
                  <input
                    type="password"
                    value={confirmPassword}
                    onChange={(e) => setConfirmPassword(e.target.value)}
                    required
                    placeholder="••••••••"
                    className="w-full pl-10 pr-4 py-3 bg-slate-900/50 border border-white/10 rounded-xl text-white placeholder:text-slate-500 focus:border-blue-500/50 focus:ring-2 focus:ring-blue-500/20 focus:outline-none transition-all"
                  />
                </div>
                {confirmPassword && password !== confirmPassword && (
                  <p className="mt-2 text-xs text-rose-400 flex items-center gap-1">
                    <XCircle size={12} />
                    Passwörter stimmen nicht überein
                  </p>
                )}
              </div>

              {/* Submit Button */}
              <button
                type="submit"
                disabled={loading || !isPasswordStrong || password !== confirmPassword}
                className="w-full flex items-center justify-center gap-2 px-6 py-3 bg-emerald-500/20 hover:bg-emerald-500/30 text-emerald-400 rounded-xl font-medium transition-all shadow-[0_0_20px_rgba(16,185,129,0.3)] hover:shadow-[0_0_30px_rgba(16,185,129,0.5)] disabled:opacity-50 disabled:cursor-not-allowed border border-emerald-500/30 mt-6"
              >
                {loading ? (
                  <>
                    <Loader2 size={20} className="animate-spin" />
                    <span>Wird erstellt...</span>
                  </>
                ) : (
                  <>
                    <UserPlus size={20} />
                    <span>Account erstellen</span>
                  </>
                )}
              </button>
            </form>

            {/* Login Link */}
            <div className="mt-6 pt-6 border-t border-white/5">
              <p className="text-center text-sm text-slate-400">
                Bereits registriert?{" "}
                <Link
                  to="/login"
                  className="text-blue-400 hover:text-blue-300 font-medium inline-flex items-center gap-1 group transition-colors"
                >
                  Jetzt anmelden
                  <ArrowRight size={14} className="group-hover:translate-x-1 transition-transform" />
                </Link>
              </p>
            </div>
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
