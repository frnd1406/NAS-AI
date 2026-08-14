
import React, { useState, useEffect } from 'react';
import { Lock, Unlock, Zap, ShieldAlert, Loader2 } from 'lucide-react';
import { getSystemCapabilities } from '../lib/api';

export function EncryptionToggle({ mode, onChange }) {
    const [warning, setWarning] = useState(null);
    const [checking, setChecking] = useState(false);

    useEffect(() => {
        let mounted = true;

        if (mode === 'USER') {
            setChecking(true);
            // Check capabilities for a hypothetical 1GB file
            getSystemCapabilities(1024 * 1024 * 1024).then(caps => {
                if (!mounted) return;
                if (caps && caps.warning) {
                    setWarning(caps);
                } else {
                    setWarning(null);
                }
                setChecking(false);
            });
        } else {
            setWarning(null);
        }

        return () => { mounted = false; };
    }, [mode]);

    const isSecure = mode === 'USER';

    return (
        <div className="flex items-center gap-3">
            {/* The Toggle Switch */}
            <div
                className={`
                    relative flex items-center p-1 rounded-lg border transition-all cursor-pointer select-none
                    ${isSecure
                        ? 'bg-indigo-500/10 border-indigo-500/30'
                        : 'bg-emerald-500/10 border-emerald-500/30'
                    }
                `}
                onClick={() => onChange(isSecure ? 'NONE' : 'USER')}
            >
                {/* Labels */}
                <div className="flex items-center gap-4 px-2">
                    <div className={`flex items-center gap-1.5 transition-colors ${!isSecure ? 'text-emerald-400' : 'text-slate-500'}`}>
                        <Zap size={14} />
                        <span className="text-xs font-bold uppercase tracking-wider">Performance</span>
                    </div>
                    <div className={`flex items-center gap-1.5 transition-colors ${isSecure ? 'text-indigo-400' : 'text-slate-500'}`}>
                        <Lock size={14} />
                        <span className="text-xs font-bold uppercase tracking-wider">Security</span>
                    </div>
                </div>

                {/* Slider Thumb */}
                <div
                    className={`
                        absolute top-1 bottom-1 w-[calc(50%-4px)] rounded-md shadow-lg transition-all
                        flex items-center justify-center
                        ${isSecure
                            ? 'translate-x-[100%] bg-indigo-500 text-white'
                            : 'translate-x-0 bg-emerald-500 text-white'
                        }
                    `}
                >
                    {checking ? (
                        <Loader2 size={12} className="animate-spin" />
                    ) : isSecure ? (
                        <Lock size={12} />
                    ) : (
                        <Unlock size={12} />
                    )}
                </div>
            </div>

            {/* Warning Message */}
            {warning && isSecure && (
                <div className="hidden xl:flex flex-col animate-in fade-in slide-in-from-left-2 duration-300">
                    <div className="flex items-center gap-1.5 text-amber-400">
                        <ShieldAlert size={14} />
                        <span className="text-xs font-bold">Hohe Last</span>
                    </div>
                    <span className="text-[10px] text-amber-500/80">
                        ~{Math.round(warning.est_time_seconds)}s für 1GB
                    </span>
                </div>
            )}
        </div>
    );
}
