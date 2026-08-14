import { useEffect, useState, useRef, useCallback, useMemo } from 'react';

// Hooks
import { useFileData } from './files/useFileData';
import { useFileOperations } from './files/useFileOperations';
import { useFilePreview } from './files/useFilePreview';
import { useDragAndDrop } from './useDragAndDrop';
import { useFileSelection } from './files/useFileSelection';
import { useVault } from '../context/VaultContext';

// Utils
import { getBreadcrumbs, joinPath } from '../utils/fileUtils';

export function useFileManager(initialPath = '/') {
    // Vault context
    const { isUnlocked, password } = useVault();

    // Data Hook (Fetching)
    const {
        files,
        trashedFiles,
        path,
        loading,
        error: fetchError,
        setError: setFetchError,
        loadFiles,
        loadTrash,
        navigateTo,
        refresh
    } = useFileData(initialPath);

    // Operations Hook (Mutations)
    // Pass refresh/load functions to operations so they can update state after actions
    const {
        uploading,
        opError,
        setOpError,
        uploadFiles,
        downloadFile,
        deleteFile,
        emptyTrash: rawEmptyTrash,
        restoreFile,
        renameFile,
        createFolder,
        batchDownload,
        downloadFolderAsZip,
        batchDelete,
        moveFile
    } = useFileOperations(loadFiles, loadTrash, isUnlocked ? password : null);

    // Combine errors
    const error = fetchError || opError;

    // Preview Hook
    const {
        previewItem,
        previewContent,
        previewLoading,
        openPreview,
        closePreview,
    } = useFilePreview();

    // Local UI State
    const [searchQuery, setSearchQuery] = useState('');
    const [viewMode, setViewMode] = useState('list');
    const [showTrash, setShowTrash] = useState(false);
    const [showNewFolderModal, setShowNewFolderModal] = useState(false);
    const [encryptionMode, setEncryptionMode] = useState('auto');
    const [contextMenu, setContextMenu] = useState({ isOpen: false, position: { x: 0, y: 0 }, item: null });
    const [renameTarget, setRenameTarget] = useState(null);

    // File Input Ref
    const fileInputRef = useRef(null);

    // Filter Logic
    const filteredFiles = useMemo(() => {
        if (!searchQuery.trim()) return files;
        const query = searchQuery.toLowerCase();
        return files.filter(f => f.name.toLowerCase().includes(query));
    }, [files, searchQuery]);

    // Selection Hook
    const {
        selectedCount,
        selectedItems,
        allSelected,
        toggleSelect,
        clearSelection,
        toggleSelectAll,
        isSelected,
    } = useFileSelection(filteredFiles);

    // Drag & Drop Hook
    const handleFilesDropped = useCallback((droppedFiles) => {
        uploadFiles(droppedFiles, path);
    }, [uploadFiles, path]);

    const { isDragging, dragProps } = useDragAndDrop(handleFilesDropped);

    // Initial Load
    useEffect(() => {
        loadFiles('/');
        loadTrash();
    }, [loadFiles, loadTrash]);

    // Path Change Logic
    useEffect(() => {
        clearSelection();
        setSearchQuery('');
    }, [path, clearSelection]);

    // --- Handlers ---

    const handleNavigate = useCallback((item) => {
        if (!item.isDir) return;
        const nextPath = joinPath(path, item.name);
        navigateTo(nextPath);
    }, [path, navigateTo]);

    const handlePreview = useCallback((item) => {
        openPreview(item, path);
    }, [openPreview, path]);

    const handleRename = useCallback(async (item, newName) => {
        return await renameFile(item, newName, path);
    }, [renameFile, path]);

    const handleDownload = useCallback((item) => {
        if (item.isDir) {
            downloadFolderAsZip(joinPath(path, item.name));
        } else {
            downloadFile(item, path);
        }
    }, [downloadFile, downloadFolderAsZip, path]);

    const handleDelete = useCallback((item) => {
        deleteFile(item, path);
    }, [deleteFile, path]);

    const handleContextMenu = useCallback((e, item) => {
        e.preventDefault();
        setContextMenu({
            isOpen: true,
            position: { x: e.clientX, y: e.clientY },
            item
        });
    }, []);

    const handleContextMenuClose = useCallback(() => {
        setContextMenu({ ...contextMenu, isOpen: false });
    }, [contextMenu]);

    const handleRenameFromContext = useCallback((item) => {
        setRenameTarget(item);
    }, []);

    const handleMoveFile = useCallback(async (sourceItem, targetFolder) => {
        const sourcePath = joinPath(path, sourceItem.name);
        const destinationPath = joinPath(path, targetFolder.name, sourceItem.name);
        return await moveFile(sourcePath, destinationPath, path);
    }, [path, moveFile]);

    const handleCreateFolder = useCallback(async (folderName) => {
        return await createFolder(folderName, path);
    }, [createFolder, path]);

    const handleUploadClick = useCallback(() => {
        fileInputRef.current?.click();
    }, []);

    const handleFileInputChange = useCallback((e) => {
        const selectedFiles = e.target.files;
        if (selectedFiles && selectedFiles.length > 0) {
            uploadFiles(Array.from(selectedFiles), path, { encryptionMode });
        }
        e.target.value = '';
    }, [uploadFiles, path, encryptionMode]);

    const handleToggleTrash = useCallback(() => {
        setShowTrash(prev => {
            const next = !prev;
            if (next) loadTrash();
            return next;
        });
    }, [loadTrash]);

    const handleBatchDownload = useCallback(() => {
        const names = selectedItems.map(f => f.name);
        batchDownload(names, path);
    }, [selectedItems, batchDownload, path]);

    const handleBatchDelete = useCallback(() => {
        const names = selectedItems.map(f => f.name);
        batchDelete(names, path);
        clearSelection();
    }, [selectedItems, batchDelete, path, clearSelection]);

    const handleToggleSelectAll = useCallback(() => {
        const allNames = filteredFiles.map(f => f.name);
        toggleSelectAll(allNames);
    }, [filteredFiles, toggleSelectAll]);

    const handleRestore = useCallback((item) => {
        restoreFile(item, path);
    }, [restoreFile, path]);

    // Wrapper for emptyTrash to match expected signature if any (emptyTrash expects trashedFiles)
    const handleEmptyTrash = useCallback(() => {
        rawEmptyTrash(trashedFiles);
    }, [rawEmptyTrash, trashedFiles]);


    const breadcrumbs = useMemo(() => getBreadcrumbs(path), [path]);

    // Facade Interface
    return {
        files: {
            data: filteredFiles, // Exposing filtered files as primary data
            allFiles: files, // Raw files if needed
            trashed: trashedFiles,
            path,
            breadcrumbs,
            isLoading: loading,
            isUploading: uploading,
            error,
            refresh: refresh,
            loadTrash,
        },
        selection: {
            selectedItems,
            selectedCount,
            allSelected,
            toggle: toggleSelect,
            toggleAll: handleToggleSelectAll,
            clear: clearSelection,
            isSelected,
        },
        actions: {
            upload: uploadFiles,
            handleUploadClick,
            handleFileInputChange,
            delete: handleDelete,
            batchDelete: handleBatchDelete,
            download: handleDownload,
            batchDownload: handleBatchDownload,
            createFolder: handleCreateFolder,
            rename: handleRename,
            move: handleMoveFile,
            navigate: handleNavigate,
            restore: handleRestore,
            emptyTrash: handleEmptyTrash,
        },
        preview: {
            item: previewItem,
            content: previewContent,
            isLoading: previewLoading,
            open: handlePreview,
            close: closePreview,
        },
        ui: {
            viewMode,
            setViewMode,
            searchQuery,
            setSearchQuery,
            showTrash,
            toggleTrash: handleToggleTrash,
            modals: {
                newFolder: {
                    isOpen: showNewFolderModal,
                    setOpen: setShowNewFolderModal
                }
            },
            contextMenu: {
                state: contextMenu,
                setState: setContextMenu, // In case manual control needed
                handleOpen: handleContextMenu,
                handleClose: handleContextMenuClose
            },
            renameTarget: {
                item: renameTarget,
                setItem: setRenameTarget,
                handleRenameFromContext
            },
            encryptionMode: {
                value: encryptionMode,
                set: setEncryptionMode
            },
            dnd: {
                isDragging,
                props: dragProps
            },
            fileInputRef
        }
    };
}
