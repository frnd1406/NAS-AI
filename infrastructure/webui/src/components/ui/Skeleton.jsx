// Skeleton Loading Components for premium loading states

// Base Skeleton with shimmer animation
export function Skeleton({ className = '' }) {
    return (
        <div
            className={`animate-pulse bg-slate-700/50 rounded-lg ${className}`}
            style={{
                background: 'linear-gradient(90deg, rgba(51,65,85,0.5) 0%, rgba(71,85,105,0.5) 50%, rgba(51,65,85,0.5) 100%)',
                backgroundSize: '200% 100%',
                animation: 'shimmer 1.5s ease-in-out infinite',
            }}
        />
    );
}

// Card Skeleton (for Dashboard stat cards)
export function CardSkeleton() {
    return (
        <div className="relative overflow-hidden rounded-2xl border border-white/10 bg-slate-900/40 backdrop-blur-xl p-6">
            <div className="flex items-start justify-between mb-6">
                <div className="flex items-center gap-3">
                    <Skeleton className="w-12 h-12 rounded-xl" />
                    <div className="space-y-2">
                        <Skeleton className="w-20 h-3" />
                        <Skeleton className="w-32 h-5" />
                    </div>
                </div>
                <Skeleton className="w-16 h-6 rounded-full" />
            </div>
            <div className="space-y-4">
                <Skeleton className="w-full h-16 rounded-xl" />
                <Skeleton className="w-full h-16 rounded-xl" />
                <Skeleton className="w-full h-16 rounded-xl" />
            </div>
        </div>
    );
}

// Search Result Skeleton
export function SearchResultSkeleton() {
    return (
        <div className="p-6 bg-slate-800/50 backdrop-blur-sm rounded-xl border border-white/10">
            <div className="flex items-start justify-between mb-3">
                <div className="flex items-center gap-3">
                    <Skeleton className="w-10 h-10 rounded-lg" />
                    <div className="space-y-2">
                        <Skeleton className="w-48 h-5" />
                        <Skeleton className="w-64 h-3" />
                    </div>
                </div>
                <div className="space-y-2">
                    <Skeleton className="w-20 h-4" />
                    <Skeleton className="w-24 h-2 rounded-full" />
                </div>
            </div>
            <Skeleton className="w-full h-12 mt-4" />
        </div>
    );
}

// File List Skeleton
export function FileListSkeleton({ count = 5 }) {
    return (
        <div className="space-y-2">
            {Array.from({ length: count }).map((_, i) => (
                <div key={i} className="flex items-center gap-4 p-4 bg-slate-800/30 rounded-xl border border-white/5">
                    <Skeleton className="w-10 h-10 rounded-lg" />
                    <div className="flex-1 space-y-2">
                        <Skeleton className="w-48 h-4" />
                        <Skeleton className="w-24 h-3" />
                    </div>
                    <Skeleton className="w-16 h-8 rounded-lg" />
                </div>
            ))}
        </div>
    );
}

// Dashboard Full Skeleton
export function DashboardSkeleton() {
    return (
        <div className="space-y-6 animate-in fade-in duration-500">
            {/* Header */}
            <div>
                <Skeleton className="w-48 h-8 mb-2" />
                <Skeleton className="w-64 h-4" />
            </div>

            {/* Main Grid */}
            <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
                <div className="lg:col-span-2">
                    <CardSkeleton />
                </div>
                <CardSkeleton />
            </div>

            {/* Bottom Card */}
            <div className="p-6 rounded-2xl border border-white/10 bg-slate-900/40">
                <div className="flex items-center justify-between">
                    <div className="flex items-center gap-4">
                        <Skeleton className="w-14 h-14 rounded-xl" />
                        <div className="space-y-2">
                            <Skeleton className="w-24 h-3" />
                            <Skeleton className="w-32 h-8" />
                            <Skeleton className="w-40 h-3" />
                        </div>
                    </div>
                    <div className="text-right space-y-2">
                        <Skeleton className="w-24 h-3" />
                        <Skeleton className="w-16 h-8" />
                        <Skeleton className="w-20 h-3" />
                    </div>
                </div>
            </div>
        </div>
    );
}

export default Skeleton;
