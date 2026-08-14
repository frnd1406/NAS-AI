import React, { useState, useEffect, memo } from 'react';
import { Edit3, Check, X, Eye, Download, Trash2, FolderOpen, Archive, CheckSquare, Square, Lock, Unlock } from 'lucide-react';
import { FileIcon } from './FileIcon';
import { InlineRenameInput } from './InlineRenameInput';
import { formatFileSize, canPreview } from '../../utils/fileUtils';
import { useVault } from '../../context/VaultContext';

export const FileListView = memo(function FileListView({
    files,
    onNavigate,
    onPreview,
    onRename,
    onDownload,
    onDelete,
    onToggleSelect,
    isSelected,
    onContextMenu,
    renameTarget,
    onRenameComplete,
}) {
    const [renamingItem, setRenamingItem] = useState(null);

    // Vault state
    const { isUnlocked } = useVault();

    // Trigger rename from context menu or prop
    useEffect(() => {
        if (renameTarget) {
            setRenamingItem(renameTarget);
            onRenameComplete?.();
        }
    }, [renameTarget, onRenameComplete]);

    const handleRowClick = (e, item) => {
        // Single click: Select
        e.preventDefault();
        onToggleSelect?.(item.name);
    };

    const handleRowDoubleClick = (e, item) => {
        // Double click: Navigate or Preview
        e.preventDefault();
        if (item.isDir) {
            onNavigate(item);
        } else if (canPreview(item.name)) {
            onPreview(item);
        }
    };

    if (files.length === 0) {
        return (
            <div className="py-12 text-center text-slate-400">
                <FolderOpen size={48} className="mx-auto mb-3 opacity-30" />
                <p className="text-sm">No files or folders</p>
            </div>
        );
    }

    return (
        <div className="overflow-x-auto -mx-6 px-6">
            <table className="w-full text-left border-collapse">
                <thead>
                    <tr className="text-xs text-slate-500 border-b border-white/5">
                        <th className="py-3 px-2 font-medium uppercase tracking-wider w-10"></th>
                        <th className="py-3 px-2 font-medium uppercase tracking-wider">Name</th>
                        <th className="py-3 px-2 font-medium uppercase tracking-wider">Size</th>
                        <th className="py-3 px-2 font-medium uppercase tracking-wider">Modified</th>
                        <th className="py-3 px-2 font-medium uppercase tracking-wider text-right">Actions</th>
                    </tr>
                </thead>
                <tbody className="text-sm">
                    {files.map((item) => {
                        const isRenaming = renamingItem?.name === item.name;
                        const selected = isSelected?.(item.name) || false;
                        const isEncrypted = item?.name?.endsWith('.enc') || false;

                        return (
                            <tr
                                key={item.name}
                                className={`group border-b border-white/5 last:border-0 hover:bg-white/5 transition-colors cursor-pointer select-none ${selected ? 'bg-blue-500/10' : ''
                                    }`}
                                onClick={(e) => handleRowClick(e, item)}
                                onDoubleClick={(e) => handleRowDoubleClick(e, item)}
                                onContextMenu={(e) => onContextMenu?.(e, item)}
                                draggable={!isRenaming} // Basic drag support
                            >
                                {/* Checkbox */}
                                <td className="py-4 px-2" onClick={(e) => e.stopPropagation()}>
                                    <button
                                        onClick={() => onToggleSelect?.(item.name)}
                                        className={`p-1 rounded transition-all ${selected
                                            ? 'text-blue-400'
                                            : 'text-slate-500 hover:text-slate-300'
                                            }`}
                                    >
                                        {selected ? <CheckSquare size={18} /> : <Square size={18} />}
                                    </button>
                                </td>

                                <td className="py-4 px-2 font-medium text-white">
                                    {isRenaming ? (
                                        <div className="flex items-center gap-2" onClick={(e) => e.stopPropagation()}>
                                            <div className={`p-2 rounded-lg ${item.isDir ? 'bg-blue-500/20 text-blue-400' : 'bg-slate-800 text-slate-400'}`}>
                                                <FileIcon name={item.name} isDir={item.isDir} size={16} />
                                            </div>
                                            <InlineRenameInput
                                                initialName={item.name}
                                                onSubmit={async (newName) => {
                                                    await onRename(item, newName);
                                                    setRenamingItem(null);
                                                }}
                                                onCancel={() => setRenamingItem(null)}
                                                containerClassName="flex-1"
                                            />
                                        </div>
                                    ) : (
                                        <div className="flex items-center gap-3">
                                            <div className={`p-2 rounded-lg ${item.isDir ? 'bg-blue-500/20 text-blue-400' : 'bg-slate-800 text-slate-400'}`}>
                                                <FileIcon name={item.name} isDir={item.isDir} size={16} />
                                            </div>
                                            {isEncrypted && (
                                                <span className={`flex items-center ${isUnlocked ? 'text-emerald-400' : 'text-rose-400'}`}>
                                                    {isUnlocked ? <Unlock size={14} /> : <Lock size={14} />}
                                                </span>
                                            )}
                                            <span className={item.isDir ? "hover:text-blue-400 transition-colors" : ""}>
                                                {item.name}
                                            </span>
                                        </div>
                                    )}
                                </td>
                                <td className="py-4 px-2 text-slate-400 font-mono text-xs">
                                    {item.isDir ? "-" : formatFileSize(item.size)}
                                </td>
                                <td className="py-4 px-2 text-slate-400 text-xs">
                                    {new Date(item.modTime).toLocaleString()}
                                </td>
                                <td className="py-4 px-2 text-right">
                                    <div className="flex items-center justify-end gap-2" onClick={(e) => e.stopPropagation()}>
                                        {!item.isDir && canPreview(item.name) && (
                                            <button
                                                onClick={() => onPreview(item)}
                                                className="p-2 rounded-lg bg-violet-500/10 hover:bg-violet-500/20 text-violet-400 border border-violet-500/20 transition-all"
                                                title="Preview"
                                            >
                                                <Eye size={14} />
                                            </button>
                                        )}
                                        <button
                                            onClick={() => {
                                                setRenamingItem(item);
                                            }}
                                            className="p-2 rounded-lg bg-blue-500/10 hover:bg-blue-500/20 text-blue-400 border border-blue-500/20 transition-all"
                                            title="Rename"
                                        >
                                            <Edit3 size={14} />
                                        </button>
                                        <button
                                            onClick={() => onDownload(item)}
                                            className="p-2 rounded-lg bg-emerald-500/10 hover:bg-emerald-500/20 text-emerald-400 border border-emerald-500/20 transition-all"
                                            title={item.isDir ? "Download as ZIP" : "Download"}
                                        >
                                            {item.isDir ? <Archive size={14} /> : <Download size={14} />}
                                        </button>
                                        <button
                                            onClick={() => onDelete(item)}
                                            className="p-2 rounded-lg bg-rose-500/10 hover:bg-rose-500/20 text-rose-400 border border-rose-500/20 transition-all"
                                            title="Delete"
                                        >
                                            <Trash2 size={14} />
                                        </button>
                                    </div>
                                </td>
                            </tr>
                        );
                    })}
                </tbody>
            </table>
        </div>
    );
});
