// Files page - Refactored using Hook Facade
import React from 'react';
import { Loader2 } from 'lucide-react';

// Hooks
import { useFileManager } from '../hooks/useFileManager';

// Components
import { GlassCard } from '../components/ui/GlassCard';
import { FileHeader } from '../components/Files/FileHeader';
import { FileGridView } from '../components/Files/FileGridView';
import { FileListView } from '../components/Files/FileListView';
import { TrashView } from '../components/Files/TrashView';
import { DragDropOverlay } from '../components/Files/DragDropOverlay';
import { NewFolderModal } from '../components/Files/NewFolderModal';
import { FilePreviewModal } from '../components/Files/FilePreviewModal';
import { ContextMenu } from '../components/Files/ContextMenu';

export default function Files({ initialPath = '/' }) {
  // Use the Facade Hook
  const {
    files,
    selection,
    actions,
    preview,
    ui
  } = useFileManager(initialPath);

  return (
    <div className="space-y-6">
      {/* Error Banner */}
      {files.error && (
        <div className="rounded-xl border border-rose-500/30 bg-rose-500/10 p-4">
          <p className="text-rose-400 text-sm font-medium">{files.error}</p>
        </div>
      )}

      {/* Hidden Upload Input */}
      <input
        ref={ui.fileInputRef}
        type="file"
        multiple
        onChange={actions.handleFileInputChange}
        className="hidden"
      />

      {/* Modals & Overlays */}
      <NewFolderModal
        isOpen={ui.modals.newFolder.isOpen}
        onClose={() => ui.modals.newFolder.setOpen(false)}
        onCreateFolder={actions.createFolder}
        currentPath={files.path}
      />

      <FilePreviewModal
        previewItem={preview.item}
        previewContent={preview.content}
        previewLoading={preview.isLoading}
        onClose={preview.close}
        onDownload={actions.download}
      />

      <ContextMenu
        isOpen={ui.contextMenu.state.isOpen}
        position={ui.contextMenu.state.position}
        item={ui.contextMenu.state.item}
        onClose={ui.contextMenu.handleClose}
        onOpen={actions.navigate}
        onPreview={preview.open}
        onRename={ui.renameTarget.handleRenameFromContext}
        onDownload={actions.download}
        onDelete={actions.delete}
      />

      {/* Main View */}
      {ui.showTrash ? (
        <TrashView
          trashedFiles={files.trashed}
          onRefresh={files.loadTrash}
          onRestore={actions.restore}
          onEmptyTrash={actions.emptyTrash}
        />
      ) : (
        <div className="relative" {...ui.dnd.props}>
          <DragDropOverlay isDragging={ui.dnd.isDragging} />

          <GlassCard className="!p-0">
            <FileHeader
              breadcrumbs={files.breadcrumbs}
              fileCount={files.data.length}
              trashedCount={files.trashed.length}
              viewMode={ui.viewMode}
              showTrash={ui.showTrash}
              uploading={files.isUploading}
              encryptionMode={ui.encryptionMode.value}
              searchQuery={ui.searchQuery}
              selectedItems={selection.selectedItems}
              selectedCount={selection.selectedCount}
              allSelected={selection.allSelected}
              onModeChange={ui.encryptionMode.set}
              onNavigate={actions.navigate}
              onUploadClick={actions.handleUploadClick}
              onNewFolder={() => ui.modals.newFolder.setOpen(true)}
              onRefresh={files.refresh}
              onViewModeChange={ui.setViewMode}
              onToggleTrash={ui.toggleTrash}
              setSearchQuery={ui.setSearchQuery}
              onToggleSelectAll={selection.toggleAll}
              onBatchDownload={actions.batchDownload}
              onBatchDelete={actions.batchDelete}
              onClearSelection={selection.clear}
            />

            <div className="p-6">
              {files.isLoading ? (
                <div className="flex flex-col items-center justify-center py-12">
                  <Loader2 size={32} className="text-blue-400 animate-spin mb-3" />
                  <p className="text-slate-400 text-sm">Dateien werden geladen...</p>
                </div>
              ) : ui.viewMode === 'grid' ? (
                <FileGridView
                  files={files.data}
                  onNavigate={actions.navigate}
                  onPreview={preview.open}
                  onRename={actions.rename}
                  onDownload={actions.download}
                  onDelete={actions.delete}
                  selectedItems={selection.selectedItems}
                  onToggleSelect={selection.toggle}
                  isSelected={selection.isSelected}
                  onContextMenu={ui.contextMenu.handleOpen}
                  renameTarget={ui.renameTarget.item}
                  onRenameComplete={() => ui.renameTarget.setItem(null)}
                  onMoveFile={actions.move}
                />
              ) : (
                <FileListView
                  files={files.data}
                  onNavigate={actions.navigate}
                  onPreview={preview.open}
                  onRename={actions.rename}
                  onDownload={actions.download}
                  onDelete={actions.delete}
                  selectedItems={selection.selectedItems}
                  onToggleSelect={selection.toggle}
                  isSelected={selection.isSelected}
                  onContextMenu={ui.contextMenu.handleOpen}
                  renameTarget={ui.renameTarget.item}
                  onRenameComplete={() => ui.renameTarget.setItem(null)}
                  onMoveFile={actions.move}
                />
              )}
            </div>
          </GlassCard>
        </div>
      )}
    </div>
  );
}
