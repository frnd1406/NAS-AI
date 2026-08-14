import { AlertTriangle, RefreshCw, Home } from "lucide-react";
import { useNavigate } from "react-router-dom";

export default function ErrorFallback({ error, resetErrorBoundary }) {
  const navigate = useNavigate();

  const handleGoHome = () => {
    navigate("/dashboard");
    resetErrorBoundary();
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-950 via-slate-900 to-slate-800 flex items-center justify-center p-6">
      <div className="max-w-2xl w-full">
        {/* Glass Card Error Container */}
        <div className="relative overflow-hidden rounded-2xl border border-white/10 bg-slate-900/40 backdrop-blur-xl shadow-2xl p-8">
          {/* Top Gradient Line */}
          <div className="absolute top-0 left-0 w-full h-[1px] bg-gradient-to-r from-transparent via-red-500/50 to-transparent"></div>

          {/* Error Icon */}
          <div className="flex justify-center mb-6">
            <div className="p-4 rounded-full bg-red-500/20 border border-red-500/30">
              <AlertTriangle size={48} className="text-red-400" />
            </div>
          </div>

          {/* Error Message */}
          <div className="text-center mb-8">
            <h1 className="text-3xl font-bold text-white mb-3">
              Oops! Etwas ist schiefgelaufen
            </h1>
            <p className="text-slate-400 text-lg mb-6">
              Die Anwendung ist auf einen unerwarteten Fehler gestoßen.
            </p>

            {/* Error Details (Collapsed by default in production) */}
            {import.meta.env.DEV && error && (
              <details className="text-left mt-4 p-4 rounded-lg bg-slate-950/50 border border-red-500/20">
                <summary className="cursor-pointer text-red-400 font-semibold mb-2 hover:text-red-300 transition-colors">
                  Fehlerdetails anzeigen
                </summary>
                <div className="mt-3 space-y-2">
                  <p className="text-red-300 font-mono text-sm">
                    <strong>Fehler:</strong> {error.message}
                  </p>
                  {error.stack && (
                    <pre className="text-slate-400 text-xs overflow-auto max-h-48 p-3 rounded bg-slate-950/80 border border-slate-700/50">
                      {error.stack}
                    </pre>
                  )}
                </div>
              </details>
            )}
          </div>

          {/* Action Buttons */}
          <div className="flex flex-col sm:flex-row gap-4 justify-center">
            {/* Retry Button */}
            <button
              onClick={resetErrorBoundary}
              className="flex items-center justify-center gap-2 px-6 py-3 rounded-xl bg-blue-600 hover:bg-blue-500 text-white font-semibold transition-all duration-200 shadow-lg hover:shadow-blue-500/50 hover:scale-105"
            >
              <RefreshCw size={20} />
              <span>Erneut versuchen</span>
            </button>

            {/* Go Home Button */}
            <button
              onClick={handleGoHome}
              className="flex items-center justify-center gap-2 px-6 py-3 rounded-xl bg-slate-700 hover:bg-slate-600 text-white font-semibold transition-all duration-200 shadow-lg hover:scale-105"
            >
              <Home size={20} />
              <span>Zum Dashboard</span>
            </button>
          </div>

          {/* Help Text */}
          <div className="mt-8 text-center">
            <p className="text-slate-500 text-sm">
              Wenn das Problem weiterhin besteht, kontaktieren Sie bitte den Support.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
