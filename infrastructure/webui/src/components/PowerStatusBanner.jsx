import { useEffect, useRef, useState } from "react";
import { AlertTriangle, CheckCircle, Zap, X } from "lucide-react";
import { useUPSStatus } from "../hooks/useUPSStatus";

// How long the green "all clear" banner stays up after power is restored.
const CLEAR_DURATION_MS = 15000;

// Tone styling mirrors the app's Toast component (dark, translucent, hairline).
const TONES = {
  warn: {
    wrap: "bg-amber-500/10 border-amber-500/30",
    accent: "text-amber-400",
    dot: "bg-amber-400",
    Icon: AlertTriangle,
  },
  crit: {
    wrap: "bg-red-500/10 border-red-500/40",
    accent: "text-red-400",
    dot: "bg-red-400",
    Icon: AlertTriangle,
  },
  ok: {
    wrap: "bg-emerald-500/10 border-emerald-500/30",
    accent: "text-emerald-400",
    dot: "bg-emerald-400",
    Icon: CheckCircle,
  },
};

const COPY = {
  warn: {
    title: "Stromausfall — Betrieb auf Akku",
    body: "Netzstrom ausgefallen. NAS und Firewall laufen auf der USV und fahren bei niedrigem Akku automatisch sicher herunter.",
  },
  crit: {
    title: "Niedriger Akku — Herunterfahren eingeleitet",
    body: "Der USV-Akku ist fast leer. NAS und Firewall fahren jetzt sicher herunter, solange noch Reserve da ist.",
  },
  ok: {
    title: "Entwarnung — Netzstrom wieder da",
    body: "Der Stromausfall ist vorbei. NAS und Firewall laufen normal weiter; die USV lädt den Akku wieder auf.",
  },
};

function formatCharge(v) {
  return typeof v === "number" ? `${Math.round(v)} %` : "–";
}

// PowerStatusBanner shows a global alert when the UPS is on battery or low, and
// a transient "all clear" once mains power returns. Steady online = nothing.
export function PowerStatusBanner() {
  const { data } = useUPSStatus();
  const prevState = useRef(null);
  const [showClear, setShowClear] = useState(false);
  const [dismissed, setDismissed] = useState(false);

  const state = data?.available ? data.state : null;

  useEffect(() => {
    if (!state) return;
    const prev = prevState.current;
    // Transition from battery back to mains → show the all-clear for a while.
    if ((prev === "on_battery" || prev === "low_battery") && state === "online") {
      setShowClear(true);
      const timer = setTimeout(() => setShowClear(false), CLEAR_DURATION_MS);
      prevState.current = state;
      return () => clearTimeout(timer);
    }
    // A fresh alarm re-arms the dismiss button.
    if (state === "on_battery" || state === "low_battery") {
      setDismissed(false);
    }
    prevState.current = state;
  }, [state]);

  let tone = null;
  if (state === "low_battery") tone = "crit";
  else if (state === "on_battery") tone = "warn";
  else if (showClear) tone = "ok";

  if (!tone || dismissed) return null;

  const cfg = TONES[tone];
  const copy = COPY[tone];
  const Icon = cfg.Icon;
  const canDismiss = tone !== "crit"; // a critical shutdown warning isn't dismissible

  return (
    <div
      className={`mb-6 flex gap-3 rounded-xl border px-4 py-3.5 backdrop-blur-sm ${cfg.wrap}`}
      role="status"
      aria-live="polite"
    >
      <Icon size={22} className={`${cfg.accent} shrink-0 mt-0.5`} />
      <div className="flex-1 min-w-0">
        <div className={`flex items-center gap-2 text-sm font-medium ${cfg.accent}`}>
          <span className={`inline-block w-2 h-2 rounded-full ${cfg.dot} animate-pulse`} />
          {copy.title}
        </div>
        <p className="mt-1 text-[13px] leading-relaxed text-slate-300 max-w-2xl">
          {copy.body}
        </p>
        <div className="mt-2.5 flex flex-wrap gap-2 text-[11.5px] text-slate-200">
          <span className="inline-flex items-center gap-1.5 rounded-full border border-white/10 bg-white/5 px-2.5 py-1">
            <Zap size={13} className="text-slate-400" />
            {data?.model || "USV"}
          </span>
          <span className="inline-flex items-center gap-1.5 rounded-full border border-white/10 bg-white/5 px-2.5 py-1 tabular-nums">
            Akku {formatCharge(data?.battery_charge)}
          </span>
        </div>
      </div>
      {canDismiss && (
        <button
          onClick={() => setDismissed(true)}
          className="ml-1 self-start p-1 rounded-lg hover:bg-white/10 transition-colors"
          aria-label="Hinweis schließen"
        >
          <X size={14} className="text-slate-400" />
        </button>
      )}
    </div>
  );
}

export default PowerStatusBanner;
