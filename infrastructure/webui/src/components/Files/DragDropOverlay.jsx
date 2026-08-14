// Drag and Drop Overlay component

import { UploadCloud } from 'lucide-react';

export function DragDropOverlay({ isDragging }) {
    if (!isDragging) return null;

    return (
        <div className="fixed inset-0 z-40 flex items-center justify-center bg-black/70 backdrop-blur-md pointer-events-none animate-in fade-in duration-200">
            <div className="p-16 rounded-3xl border-4 border-dashed border-blue-400 bg-gradient-to-br from-blue-500/20 to-cyan-500/20 shadow-[0_0_60px_rgba(59,130,246,0.4)] transform scale-105 transition-transform">
                <UploadCloud size={80} className="text-blue-400 mx-auto mb-6 animate-bounce" />
                <p className="text-3xl font-bold text-blue-400 mb-2">Dateien hier ablegen</p>
                <p className="text-slate-200 text-base">Mehrere Dateien gleichzeitig möglich</p>
                <div className="mt-6 flex items-center justify-center gap-2">
                    <div className="w-3 h-3 rounded-full bg-blue-400 animate-pulse"></div>
                    <div className="w-3 h-3 rounded-full bg-blue-400 animate-pulse delay-75"></div>
                    <div className="w-3 h-3 rounded-full bg-blue-400 animate-pulse delay-150"></div>
                </div>
            </div>
        </div>
    );
}
