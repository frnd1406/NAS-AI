import React from 'react';
import { Search, X, Download, Trash2, CheckSquare, Square } from 'lucide-react';
import { FileToolbar } from './FileToolbar';

export function FileHeader({
    breadcrumbs,
    fileCount,
    trashedCount,
    viewMode,
    showTrash,
    uploading,
    encryptionMode,
    searchQuery,
    selectedItems,
    selectedCount,
    allSelected,
    onModeChange,
    onNavigate,
    onUploadClick,
    onNewFolder,
    onRefresh,
    onViewModeChange,
    onToggleTrash,
    setSearchQuery,
    onToggleSelectAll,
    onBatchDownload,
    onBatchDelete,
    onClearSelection
}) {
    return (
        <div className="flex flex-col">
            {/* Toolbar */}
            <FileToolbar
                breadcrumbs={breadcrumbs}
                fileCount={fileCount}
                trashedCount={trashedCount}
                viewMode={viewMode}
                showTrash={showTrash}
                uploading={uploading}
                encryptionMode={encryptionMode}
                onModeChange={onModeChange}
                onNavigate={onNavigate}
                onUploadClick={onUploadClick}
                onNewFolder={onNewFolder}
                onRefresh={onRefresh}
                onViewModeChange={onViewModeChange}
                onToggleTrash={onToggleTrash}
            />

            {/* Search & Selection Bar */}
            {!showTrash && (
                <div className="px-4 py-3 border-b border-white/5 flex flex-wrap items-center gap-3 bg-slate-800/20 backdrop-blur-sm">
                    {/* Search Input */}
                    <div className="relative flex-1 min-w-[200px] max-w-md">
                        <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" />
                        <input
                            type="text"
                            placeholder="Dateien filtern..."
                            value={searchQuery}
                            onChange={(e) => setSearchQuery(e.target.value)}
                            className="w-full pl-10 pr-10 py-2 bg-slate-800/50 border border-white/10 rounded-lg text-white text-sm placeholder-slate-500 focus:outline-none focus:border-blue-500/50 transition-all"
                        />
                        {searchQuery && (
                            <button
                                onClick={() => setSearchQuery('')}
                                className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-white"
                            >
                                <X size={14} />
                            </button>
                        )}
                    </div>

                    {/* Select All Toggle */}
                    <button
                        onClick={onToggleSelectAll}
                        className={`flex items-center gap-2 px-3 py-2 rounded-lg border transition-all text-sm ${allSelected
                            ? 'bg-blue-500/20 text-blue-400 border-blue-500/30'
                            : 'bg-white/5 text-slate-400 border-white/10 hover:bg-white/10 hover:text-white'
                            }`}
                    >
                        {allSelected ? <CheckSquare size={16} /> : <Square size={16} />}
                        <span className="hidden sm:inline">Alle auswählen</span>
                    </button>

                    {/* Selection Actions */}
                    {selectedCount > 0 && (
                        <div className="flex items-center gap-2 ml-auto animate-in fade-in slide-in-from-right-4 duration-200">
                            <span className="text-sm text-slate-400 hidden md:inline">
                                {selectedCount} ausgewählt
                            </span>
                            <button
                                onClick={onBatchDownload}
                                className="flex items-center gap-2 px-3 py-2 rounded-lg bg-emerald-500/10 hover:bg-emerald-500/20 text-emerald-400 border border-emerald-500/20 transition-all text-sm"
                                title="Als ZIP herunterladen"
                            >
                                <Download size={16} />
                                <span className="hidden sm:inline">ZIP</span>
                            </button>
                            <button
                                onClick={onBatchDelete}
                                className="flex items-center gap-2 px-3 py-2 rounded-lg bg-rose-500/10 hover:bg-rose-500/20 text-rose-400 border border-rose-500/20 transition-all text-sm"
                                title="Ausgewählte löschen"
                            >
                                <Trash2 size={16} />
                                <span className="hidden sm:inline">Löschen</span>
                            </button>
                            <button
                                onClick={onClearSelection}
                                className="p-2 rounded-lg bg-white/5 hover:bg-white/10 text-slate-400 hover:text-white border border-white/10 transition-all"
                                title="Auswahl aufheben"
                            >
                                <X size={16} />
                            </button>
                        </div>
                    )}
                </div>
            )}
        </div>
    );
}
