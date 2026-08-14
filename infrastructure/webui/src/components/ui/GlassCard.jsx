// Reusable Glass Card component with glassmorphism effect

export function GlassCard({ children, className = '' }) {
    return (
        <div className={`relative overflow-hidden rounded-2xl border border-white/10 bg-slate-900/40 backdrop-blur-xl shadow-2xl ${className}`}>
            <div className="absolute top-0 left-0 w-full h-[1px] bg-gradient-to-r from-transparent via-white/20 to-transparent opacity-50"></div>
            <div className="p-6 h-full flex flex-col">
                {children}
            </div>
        </div>
    );
}
