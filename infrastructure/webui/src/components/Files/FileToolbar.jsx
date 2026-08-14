// File Toolbar component with breadcrumbs and actions

import {
    Home,
    ChevronRight,
    UploadCloud,
    FolderPlus,
    RefreshCw,
    List,
    Grid3x3,
    Trash,
    Loader2,
    Shield,
    Lock,
    Unlock
} from 'lucide-react';

export function FileToolbar({
    breadcrumbs,
    fileCount,
    trashedCount,
    viewMode,
    showTrash,
    uploading,
    encryptionMode = 'auto',
    onModeChange,
    onNavigate,
    onUploadClick,
    onNewFolder,
    onRefresh,
    onViewModeChange,
    onToggleTrash,
}) {
    // Encryption mode definitions
    const modes = {
        auto: { icon: Shield, color: 'text-slate-400', bgColor: 'bg-slate-500/10 border-slate-500/20', label: 'Auto' },
        force: { icon: Lock, color: 'text-rose-400', bgColor: 'bg-rose-500/10 border-rose-500/20', label: 'Sicher' },
        none: { icon: Unlock, color: 'text-amber-400', bgColor: 'bg-amber-500/10 border-amber-500/20', label: 'Offen' }
    };
    const current = modes[encryptionMode] || modes.auto;
    const ModeIcon = current.icon;

    const cycleMode = () => {
        const next = encryptionMode === 'auto' ? 'force' : encryptionMode === 'force' ? 'none' : 'auto';
        onModeChange?.(next);
    };
    return (
        <div className="p-4 border-b border-white/5">
            <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-4">
                {/* Left: Breadcrumbs */}
                <div className="flex items-center gap-2 flex-wrap">
                    {breadcrumbs.map((crumb, index) => (
                        <div key={crumb.path} className="flex items-center gap-2">
                            {index === 0 ? (
                                <button
                                    onClick={() => onNavigate(crumb.path)}
                                    className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-white/5 hover:bg-white/10 text-slate-300 hover:text-white border border-white/10 transition-all"
                                >
                                    <Home size={16} />
                                    <span className="text-sm font-medium">{crumb.name}</span>
                                </button>
                            ) : (
                                <>
                                    <ChevronRight size={16} className="text-slate-600" />
                                    <button
                                        onClick={() => onNavigate(crumb.path)}
                                        className={`px-3 py-1.5 rounded-lg text-sm font-medium transition-all ${index === breadcrumbs.length - 1
                                            ? 'bg-blue-500/20 text-blue-400 border border-blue-500/30'
                                            : 'bg-white/5 hover:bg-white/10 text-slate-300 hover:text-white border border-white/10'
                                            }`}
                                    >
                                        {crumb.name}
                                    </button>
                                </>
                            )}
                        </div>
                    ))}
                    <div className="ml-2 px-2 py-1 rounded-lg bg-slate-800/50 border border-white/5">
                        <span className="text-xs text-slate-400">{fileCount} items</span>
                    </div>
                </div>

                {/* Right: Actions */}
                <div className="flex items-center gap-2 flex-wrap">
                    {/* Smart Upload Selector - Encryption Mode Toggle */}
                    <button
                        onClick={cycleMode}
                        className={`flex items-center gap-2 px-3 py-2 rounded-lg border ${current.color} ${current.bgColor} hover:opacity-80 transition-all`}
                        title={`Verschlüsselung: ${current.label}\n\nAuto = System entscheidet\nSicher = Immer verschlüsseln\nOffen = Nie verschlüsseln`}
                    >
                        <ModeIcon size={16} />
                        <span className="hidden sm:inline text-sm font-medium">{current.label}</span>
                    </button>

                    {/* Upload Button */}
                    <button
                        onClick={onUploadClick}
                        disabled={uploading}
                        className="flex items-center gap-2 px-4 py-2 rounded-lg bg-blue-500/20 hover:bg-blue-500/30 text-blue-400 border border-blue-500/30 disabled:opacity-50 disabled:cursor-not-allowed transition-all shadow-[0_0_15px_rgba(59,130,246,0.2)]"
                        title="Upload Files (Multiple)"
                    >
                        {uploading ? (
                            <>
                                <Loader2 size={16} className="animate-spin" />
                                <span className="hidden sm:inline text-sm font-medium">Hochladen...</span>
                            </>
                        ) : (
                            <>
                                <UploadCloud size={16} />
                                <span className="hidden sm:inline text-sm font-medium">Hochladen</span>
                            </>
                        )}
                    </button>

                    {/* New Folder Button */}
                    <button
                        onClick={onNewFolder}
                        className="flex items-center gap-2 px-4 py-2 rounded-lg bg-emerald-500/10 hover:bg-emerald-500/20 text-emerald-400 border border-emerald-500/20 transition-all"
                        title="New Folder"
                    >
                        <FolderPlus size={16} />
                        <span className="hidden sm:inline text-sm font-medium">New Folder</span>
                    </button>

                    {/* Refresh Button */}
                    <button
                        onClick={onRefresh}
                        className="p-2 rounded-lg bg-white/5 hover:bg-white/10 text-slate-400 hover:text-white border border-white/10 transition-all"
                        title="Refresh"
                    >
                        <RefreshCw size={16} />
                    </button>

                    {/* View Mode Toggle */}
                    <div className="flex items-center gap-1 p-1 rounded-lg bg-white/5 border border-white/10">
                        <button
                            onClick={() => onViewModeChange('list')}
                            className={`p-2 rounded ${viewMode === 'list' ? 'bg-blue-500/20 text-blue-400' : 'text-slate-400 hover:text-white'} transition-all`}
                            title="List View"
                        >
                            <List size={16} />
                        </button>
                        <button
                            onClick={() => onViewModeChange('grid')}
                            className={`p-2 rounded ${viewMode === 'grid' ? 'bg-blue-500/20 text-blue-400' : 'text-slate-400 hover:text-white'} transition-all`}
                            title="Grid View"
                        >
                            <Grid3x3 size={16} />
                        </button>
                    </div>

                    {/* Trash Button */}
                    <button
                        onClick={onToggleTrash}
                        className={`flex items-center gap-2 px-4 py-2 rounded-lg ${showTrash ? 'bg-rose-500/20 text-rose-400 border-rose-500/30' : 'bg-white/5 text-slate-300 border-white/10'} hover:bg-white/10 hover:text-white border transition-all`}
                    >
                        <Trash size={16} />
                        <span className="hidden sm:inline text-sm font-medium">Trash</span>
                        {trashedCount > 0 && (
                            <span className="px-2 py-0.5 rounded-full bg-rose-500 text-white text-xs font-bold">
                                {trashedCount}
                            </span>
                        )}
                    </button>
                </div>
            </div>
        </div>
    );
}
