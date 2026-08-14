// New Folder Modal component

import { useState } from 'react';
import { X, FolderPlus, Loader2 } from 'lucide-react';
import { GlassCard } from '../ui/GlassCard';

export function NewFolderModal({ isOpen, onClose, onCreateFolder, currentPath }) {
    const [folderName, setFolderName] = useState('');
    const [creating, setCreating] = useState(false);

    if (!isOpen) return null;

    const handleCreate = async () => {
        if (!folderName.trim()) return;

        setCreating(true);
        const success = await onCreateFolder(folderName.trim());
        setCreating(false);

        if (success) {
            setFolderName('');
            onClose();
        }
    };

    const handleClose = () => {
        setFolderName('');
        onClose();
    };

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4 animate-in fade-in duration-200">
            <div className="w-full max-w-md animate-in zoom-in-95 duration-200">
                <GlassCard>
                    <div className="flex items-start justify-between mb-4">
                        <div className="flex items-center gap-3">
                            <div className="p-3 rounded-xl bg-blue-500/20 border border-blue-500/30">
                                <FolderPlus size={20} className="text-blue-400" />
                            </div>
                            <h2 className="text-xl font-bold text-white">Neuer Ordner</h2>
                        </div>
                        <button
                            onClick={handleClose}
                            className="p-2 rounded-lg bg-slate-800/50 hover:bg-rose-500/20 text-slate-400 hover:text-rose-400 border border-white/10 hover:border-rose-500/30 transition-all"
                        >
                            <X size={18} />
                        </button>
                    </div>

                    <div className="space-y-4">
                        <div>
                            <label className="block text-sm font-medium text-slate-300 mb-2">
                                Ordnername
                            </label>
                            <input
                                type="text"
                                value={folderName}
                                onChange={(e) => setFolderName(e.target.value)}
                                onKeyDown={(e) => {
                                    if (e.key === 'Enter' && folderName.trim()) handleCreate();
                                    if (e.key === 'Escape') handleClose();
                                }}
                                placeholder="Mein Ordner"
                                className="w-full px-4 py-2.5 bg-slate-800/50 border border-white/10 rounded-lg text-white focus:border-blue-500/50 focus:bg-slate-800 focus:outline-none transition-all"
                                autoFocus
                            />
                            <p className="text-xs text-slate-500 mt-1.5">
                                Wird erstellt in: <span className="text-blue-400 font-mono">{currentPath}</span>
                            </p>
                        </div>

                        <div className="flex items-center justify-end gap-3 pt-4 border-t border-white/5">
                            <button
                                onClick={handleClose}
                                className="px-4 py-2 rounded-lg bg-slate-800/50 hover:bg-slate-800 text-slate-300 hover:text-white border border-white/10 transition-all"
                            >
                                Abbrechen
                            </button>
                            <button
                                onClick={handleCreate}
                                disabled={creating || !folderName.trim()}
                                className="flex items-center gap-2 px-4 py-2 rounded-lg bg-blue-500/20 hover:bg-blue-500/30 text-blue-400 border border-blue-500/30 disabled:opacity-50 disabled:cursor-not-allowed transition-all shadow-[0_0_15px_rgba(59,130,246,0.2)]"
                            >
                                {creating ? (
                                    <>
                                        <Loader2 size={16} className="animate-spin" />
                                        <span>Erstelle...</span>
                                    </>
                                ) : (
                                    <>
                                        <FolderPlus size={16} />
                                        <span>Erstellen</span>
                                    </>
                                )}
                            </button>
                        </div>
                    </div>
                </GlassCard>
            </div>
        </div>
    );
}
