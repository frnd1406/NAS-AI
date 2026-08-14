// File Grid View component with selection support

import React, { memo } from 'react';
import { FolderOpen } from 'lucide-react';
import { FileCard } from './FileCard';

export const FileGridView = memo(function FileGridView({
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
    onMoveFile,
}) {
    if (files.length === 0) {
        return (
            <div className="py-12 text-center text-slate-400">
                <FolderOpen size={48} className="mx-auto mb-3 opacity-30" />
                <p className="text-sm">No files or folders</p>
            </div>
        );
    }

    return (
        <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-4">
            {files.map((item) => (
                <FileCard
                    key={item.name}
                    item={item}
                    onNavigate={onNavigate}
                    onPreview={onPreview}
                    onRename={onRename}
                    onDownload={onDownload}
                    onDelete={onDelete}
                    onToggleSelect={onToggleSelect}
                    isSelected={isSelected}
                    onContextMenu={onContextMenu}
                    renameTarget={renameTarget}
                    onRenameComplete={onRenameComplete}
                    onMoveFile={onMoveFile}
                />
            ))}
        </div>
    );
});
