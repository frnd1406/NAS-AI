// File Preview Modal component

import { X, Download, Loader2 } from 'lucide-react';
import { GlassCard } from '../ui/GlassCard';
import { formatFileSize } from '../../utils/fileUtils';

export function FilePreviewModal({
    previewItem,
    previewContent,
    previewLoading,
    onClose,
    onDownload
}) {
    if (!previewItem) return null;

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-md p-4 animate-in fade-in duration-200">
            <div className="w-full max-w-4xl max-h-[90vh] animate-in zoom-in-95 duration-200">
                <GlassCard>
                    <div className="p-6">
                        {/* Modal Header */}
                        <div className="flex items-start justify-between mb-4">
                            <div>
                                <h2 className="text-xl font-bold text-white tracking-tight">{previewItem.name}</h2>
                                <p className="text-slate-400 text-sm mt-1">{formatFileSize(previewItem.size)}</p>
                            </div>
                            <button
                                onClick={onClose}
                                className="p-2 rounded-lg bg-slate-800/50 hover:bg-rose-500/20 text-slate-400 hover:text-rose-400 border border-white/10 hover:border-rose-500/30 transition-all"
                            >
                                <X size={20} />
                            </button>
                        </div>

                        {/* Preview Content */}
                        {previewLoading ? (
                            <div className="flex items-center justify-center py-12">
                                <Loader2 size={32} className="text-blue-400 animate-spin" />
                            </div>
                        ) : previewContent?.type === 'image' ? (
                            <div className="max-h-[60vh] overflow-auto rounded-lg bg-black/50 p-4">
                                <img
                                    src={previewContent.url}
                                    alt={previewItem.name}
                                    className="max-w-full mx-auto rounded"
                                />
                            </div>
                        ) : previewContent?.type === 'text' ? (
                            <div className="max-h-[60vh] overflow-auto rounded-lg bg-slate-900 p-4 border border-white/10">
                                <pre className="text-slate-300 text-sm font-mono whitespace-pre-wrap">
                                    {previewContent.content}
                                </pre>
                            </div>
                        ) : (
                            <div className="py-12 text-center text-slate-400">
                                <p>Preview not available</p>
                            </div>
                        )}

                        {/* Actions */}
                        <div className="flex items-center justify-end gap-3 mt-4 pt-4 border-t border-white/5">
                            <button
                                onClick={() => onDownload(previewItem)}
                                className="flex items-center gap-2 px-4 py-2 rounded-lg bg-emerald-500/10 hover:bg-emerald-500/20 text-emerald-400 border border-emerald-500/20 transition-all"
                            >
                                <Download size={16} />
                                <span>Download</span>
                            </button>
                            <button
                                onClick={onClose}
                                className="px-4 py-2 rounded-lg bg-slate-800/50 hover:bg-slate-800 text-slate-300 hover:text-white border border-white/10 transition-all"
                            >
                                Close
                            </button>
                        </div>
                    </div>
                </GlassCard>
            </div>
        </div>
    );
}
