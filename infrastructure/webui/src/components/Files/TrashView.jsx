import { RefreshCw, Trash, Trash2 } from 'lucide-react';
import { GlassCard } from '../ui/GlassCard';
import { FileIcon } from './FileIcon';

export function TrashView({ trashedFiles, onRefresh, onRestore, onEmptyTrash }) {
    return (
        <GlassCard>
            <div className="flex items-center justify-between mb-6">
                <div>
                    <h3 className="text-white font-semibold text-lg tracking-tight flex items-center gap-2">
                        <Trash className="text-rose-400" size={20} />
                        Papierkorb
                    </h3>
                    <p className="text-slate-400 text-xs mt-1">{trashedFiles.length} gelöschte Dateien</p>
                </div>
                <div className="flex items-center gap-2">
                    {trashedFiles.length > 0 && onEmptyTrash && (
                        <button
                            onClick={onEmptyTrash}
                            className="flex items-center gap-2 px-3 py-2 rounded-lg bg-rose-500/10 hover:bg-rose-500/20 text-rose-400 border border-rose-500/20 transition-all"
                            title="Papierkorb leeren"
                        >
                            <Trash2 size={16} />
                            <span className="text-sm font-medium">Leeren</span>
                        </button>
                    )}
                    <button
                        onClick={onRefresh}
                        className="p-2 rounded-lg bg-white/5 hover:bg-white/10 text-slate-400 hover:text-white transition-all"
                        title="Refresh"
                    >
                        <RefreshCw size={18} />
                    </button>
                </div>
            </div>

            {trashedFiles.length === 0 ? (
                <div className="py-12 text-center text-slate-400">
                    <Trash size={48} className="mx-auto mb-3 opacity-30" />
                    <p className="text-sm">Papierkorb ist leer</p>
                </div>
            ) : (
                <div className="space-y-2">
                    {trashedFiles.map((item, idx) => (
                        <div
                            key={idx}
                            className="flex items-center justify-between p-3 rounded-lg bg-white/5 border border-white/5 hover:bg-white/10 transition-all"
                        >
                            <div className="flex items-center gap-3">
                                <div className="p-2 rounded-lg bg-slate-800/50 text-slate-400">
                                    <FileIcon name={item.name} isDir={item.isDir} size={16} />
                                </div>
                                <div>
                                    <p className="text-white font-medium text-sm">{item.name}</p>
                                    <p className="text-slate-500 text-xs">{item.originalPath || 'Unknown path'}</p>
                                </div>
                            </div>
                            <button
                                onClick={() => onRestore(item)}
                                className="flex items-center gap-2 px-4 py-2 rounded-lg bg-emerald-500/10 hover:bg-emerald-500/20 text-emerald-400 border border-emerald-500/20 transition-all"
                            >
                                <RefreshCw size={14} />
                                <span className="text-sm font-medium">Wiederherstellen</span>
                            </button>
                        </div>
                    ))}
                </div>
            )}
        </GlassCard>
    );
}
