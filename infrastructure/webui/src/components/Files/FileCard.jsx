// File Card component for Grid View with selection support

import { useState, useEffect, useCallback, memo } from 'react';
import { Edit3, Check, X, Eye, Download, Trash2, Archive, CheckSquare, Square, Lock, Unlock } from 'lucide-react';
import { FileIcon } from './FileIcon';
import { formatFileSize, canPreview } from '../../utils/fileUtils';
import { InlineRenameInput } from './InlineRenameInput';
import { useLongPress } from '../../hooks/useLongPress';
import { useVault } from '../../context/VaultContext';

export const FileCard = memo(function FileCard({
    item,
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
    onMoveFile,
}) {
    const [isRenaming, setIsRenaming] = useState(false);
    const [isDragOver, setIsDragOver] = useState(false);

    // Vault state
    const { isUnlocked } = useVault();
    const isEncrypted = item?.name?.endsWith('.enc') || false;

    // Long-press handler for mobile
    const handleLongPress = useCallback(() => {
        if (!item.isDir && canPreview(item.name)) {
            onPreview?.(item);
        } else {
            onContextMenu?.({ preventDefault: () => { }, clientX: 100, clientY: 100 }, item);
        }
    }, [item, onPreview, onContextMenu]);

    const longPressProps = useLongPress(handleLongPress, {
        delay: 500,
    });

    const selected = isSelected?.(item.name) || false;

    // Trigger rename from context menu
    useEffect(() => {
        if (renameTarget && renameTarget.name === item.name) {
            setIsRenaming(true);
            onRenameComplete?.();
        }
    }, [renameTarget, item.name, onRenameComplete]);

    // Interaction Handlers
    const handleClick = (e) => {
        if (isRenaming) return;
        // Single click selects
        e.preventDefault();
        onToggleSelect?.(item.name);
    };

    const handleDoubleClick = (e) => {
        if (isRenaming) return;
        // Double click navigates or previews
        e.preventDefault();
        e.stopPropagation();
        if (item.isDir) {
            onNavigate(item);
        } else if (canPreview(item.name)) {
            onPreview(item);
        }
    };

    // Drag handlers
    const handleDragStart = (e) => {
        e.dataTransfer.setData('application/json', JSON.stringify(item));
        e.dataTransfer.effectAllowed = 'move';
    };

    const handleDragOver = (e) => {
        if (!item.isDir) return;
        e.preventDefault();
        e.dataTransfer.dropEffect = 'move';
        setIsDragOver(true);
    };

    const handleDragLeave = () => {
        setIsDragOver(false);
    };

    const handleDrop = async (e) => {
        if (!item.isDir) return;
        e.preventDefault();
        setIsDragOver(false);

        try {
            const data = e.dataTransfer.getData('application/json');
            if (!data) return;
            const sourceItem = JSON.parse(data);
            if (sourceItem.name === item.name) return; // Can't drop on itself
            await onMoveFile?.(sourceItem, item);
        } catch (err) {
            console.error('Drop failed:', err);
        }
    };

    return (
        <div
            className={`group relative overflow-hidden rounded-xl border transition-all cursor-pointer select-none ${selected
                ? 'border-blue-500/50 bg-blue-500/10'
                : isDragOver
                    ? 'border-emerald-500/50 bg-emerald-500/10'
                    : 'border-white/10 bg-slate-900/40 dark:bg-slate-900/40 hover:bg-white/5'
                }`}
            onClick={handleClick}
            onDoubleClick={handleDoubleClick}
            onContextMenu={(e) => onContextMenu?.(e, item)}
            draggable={!isRenaming}
            onDragStart={handleDragStart}
            onDragOver={handleDragOver}
            onDragLeave={handleDragLeave}
            onDrop={handleDrop}
            {...longPressProps}
        >
            {/* Selection Checkbox */}
            <div
                className="absolute top-2 left-2 z-10"
                onClick={(e) => e.stopPropagation()}
            >
                <button
                    onClick={() => onToggleSelect?.(item.name)}
                    className={`p-1 rounded transition-all ${selected
                        ? 'text-blue-400 bg-blue-500/20'
                        : 'text-slate-500 hover:text-slate-300 opacity-0 group-hover:opacity-100 bg-slate-800/80'
                        }`}
                >
                    {selected ? <CheckSquare size={16} /> : <Square size={16} />}
                </button>
            </div>

            <div className="p-4 flex flex-col items-center text-center">
                {/* Icon with Encryption Indicator */}
                <div className={`relative p-4 rounded-xl mb-3 ${item.isDir ? 'bg-blue-500/20 text-blue-400' : 'bg-slate-800/50 text-slate-400'} group-hover:scale-110 transition-transform`}>
                    <FileIcon name={item.name} isDir={item.isDir} size={32} />
                    {isEncrypted && (
                        <div className={`absolute -top-1 -right-1 p-1 rounded-full ${isUnlocked ? 'bg-emerald-500/20' : 'bg-rose-500/20'}`}>
                            {isUnlocked ? (
                                <Unlock size={14} className="text-emerald-400" />
                            ) : (
                                <Lock size={14} className="text-rose-400" />
                            )}
                        </div>
                    )}
                </div>

                {/* Name */}
                {isRenaming ? (
                    <InlineRenameInput
                        initialName={item.name}
                        onSubmit={async (newName) => {
                            await onRename(item, newName);
                            setIsRenaming(false);
                        }}
                        onCancel={() => setIsRenaming(false)}
                        inputClassName="text-xs px-2 py-1"
                    />
                ) : (
                    <p className="text-sm font-medium text-white truncate w-full px-2 group-hover:text-blue-400 transition-colors">
                        {item.name}
                    </p>
                )}

                {/* Size */}
                {!item.isDir && (
                    <p className="text-xs text-slate-500 mt-1">{formatFileSize(item.size)}</p>
                )}

                {/* Actions */}
                <div className="flex items-center gap-1 mt-3 opacity-0 group-hover:opacity-100 transition-opacity" onClick={(e) => e.stopPropagation()}>
                    {!item.isDir && canPreview(item.name) && (
                        <button
                            onClick={() => onPreview(item)}
                            className="p-1.5 rounded-lg bg-violet-500/10 hover:bg-violet-500/20 text-violet-400 border border-violet-500/20 transition-all"
                            title="Preview"
                        >
                            <Eye size={12} />
                        </button>
                    )}
                    <button
                        onClick={() => {
                            setIsRenaming(true);
                            setNewName(item.name);
                        }}
                        className="p-1.5 rounded-lg bg-blue-500/10 hover:bg-blue-500/20 text-blue-400 border border-blue-500/20 transition-all"
                        title="Rename"
                    >
                        <Edit3 size={12} />
                    </button>
                    <button
                        onClick={() => onDownload(item)}
                        className="p-1.5 rounded-lg bg-emerald-500/10 hover:bg-emerald-500/20 text-emerald-400 border border-emerald-500/20 transition-all"
                        title={item.isDir ? "Download as ZIP" : "Download"}
                    >
                        {item.isDir ? <Archive size={12} /> : <Download size={12} />}
                    </button>
                    <button
                        onClick={() => onDelete(item)}
                        className="p-1.5 rounded-lg bg-rose-500/10 hover:bg-rose-500/20 text-rose-400 border border-rose-500/20 transition-all"
                        title="Delete"
                    >
                        <Trash2 size={12} />
                    </button>
                </div>
            </div>
        </div>
    );
});
