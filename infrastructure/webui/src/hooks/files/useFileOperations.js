// Hook for file operations (mutations)
import { useState, useCallback } from 'react';
import * as fileApi from '../../api/files';
import { joinPath } from '../../utils/fileUtils';

export function useFileOperations(loadFiles, loadTrash, vaultPassword = null) {
    const [uploading, setUploading] = useState(false);
    const [opError, setOpError] = useState('');

    // Upload files with optional encryption mode override
    const uploadFiles = useCallback(async (filesToUpload, currentPath, options = {}) => {
        if (!filesToUpload || filesToUpload.length === 0) return;

        const { encryptionMode = 'auto' } = options;

        setUploading(true);
        setOpError('');

        try {
            for (let i = 0; i < filesToUpload.length; i++) {
                const file = filesToUpload[i];
                console.log(`Uploading file ${i + 1}/${filesToUpload.length}:`, file.name);

                const isVault = (currentPath.startsWith('vault') || currentPath.startsWith('/vault'));

                if (isVault) {
                    if (!vaultPassword) throw new Error("Vault locked: Cannot upload");

                    // Use standard upload with Forced Encryption + Password
                    await fileApi.uploadFile(file, currentPath, {
                        encryptionOverride: 'force',
                        encryptionPassword: vaultPassword
                    });
                } else {
                    await fileApi.uploadFile(file, currentPath, encryptionMode);
                }
            }

            await loadFiles(currentPath);
        } catch (err) {
            console.error('Upload error:', err);
            setOpError(err.message || 'Unknown error');
            if (err.message === 'Unauthorized') {
                window.location.href = '/login';
            }
        } finally {
            setUploading(false);
        }
    }, [loadFiles, vaultPassword]);

    // Download file
    const downloadFile = useCallback(async (item, currentPath) => {
        try {
            const url = await fileApi.downloadFileUrl(item, currentPath);
            const a = document.createElement('a');
            a.href = url;
            a.download = item.name;
            a.click();
            window.URL.revokeObjectURL(url);
        } catch (err) {
            setOpError(err.message);
        }
    }, []);

    // Delete file (Move to Trash)
    const deleteFile = useCallback(async (item, currentPath) => {
        if (item.name === '.trash') {
            setOpError('Der Papierkorb kann nicht gelöscht werden');
            return;
        }

        if (!window.confirm(`"${item.name}" wirklich löschen?`)) return;

        try {
            await fileApi.deleteFile(item, currentPath);
            await loadFiles(currentPath);
            await loadTrash();
        } catch (err) {
            setOpError(err.message);
        }
    }, [loadFiles, loadTrash]);

    // Permanently delete file from Trash
    const deleteFromTrash = useCallback(async (item) => {
        try {
            return await fileApi.deleteFromTrash(item);
        } catch (err) {
            console.error(err);
            return false;
        }
    }, []);

    // Empty Trash
    const emptyTrash = useCallback(async (trashedFiles) => {
        if (!window.confirm('Papierkorb endgültig leeren? Diese Aktion kann nicht rückgängig gemacht werden.')) return;

        try {
            // Optimistic approach: Run all deletes in parallel
            const deletePromises = trashedFiles.map(item => deleteFromTrash(item));
            await Promise.all(deletePromises);

            await loadTrash();
        } catch (err) {
            setOpError('Fehler beim Leeren des Papierkorbs');
        }
    }, [deleteFromTrash, loadTrash]);

    // Restore file from trash
    const restoreFile = useCallback(async (item, currentPath) => {
        try {
            await fileApi.restoreFile(item);
            await loadTrash();
            await loadFiles(currentPath);
        } catch (err) {
            setOpError(err.message);
        }
    }, [loadFiles, loadTrash]);

    // Rename file
    const renameFile = useCallback(async (item, newName, currentPath) => {
        if (!newName || newName === item.name) return false;

        const oldPath = joinPath(currentPath, item.name);
        const newPath = joinPath(currentPath, newName);

        try {
            await fileApi.renameFile(oldPath, newPath);
            await loadFiles(currentPath);
            return true;
        } catch (err) {
            setOpError(err.message);
            return false;
        }
    }, [loadFiles]);

    // Create folder
    const createFolder = useCallback(async (folderName, currentPath) => {
        if (!folderName || folderName.trim() === '') {
            setOpError('Ordnername darf nicht leer sein');
            return false;
        }

        try {
            const folderPath = joinPath(currentPath, folderName.trim());
            await fileApi.createFolder(folderPath);
            await loadFiles(currentPath);
            return true;
        } catch (err) {
            setOpError(err.message || 'Fehler beim Erstellen des Ordners');
            return false;
        }
    }, [loadFiles]);

    // Batch download multiple files as ZIP
    const batchDownload = useCallback(async (selectedNames, currentPath) => {
        if (!selectedNames || selectedNames.length === 0) {
            setOpError('Keine Dateien ausgewählt');
            return;
        }

        try {
            const paths = selectedNames.map(name => joinPath(currentPath, name));
            const blob = await fileApi.batchDownloadZip(paths);
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = 'download.zip';
            a.click();
            window.URL.revokeObjectURL(url);
        } catch (err) {
            setOpError(err.message);
        }
    }, []);

    // Download folder as ZIP
    const downloadFolderAsZip = useCallback(async (folderPath) => {
        try {
            const blob = await fileApi.downloadFolderZip(folderPath);
            const folderName = folderPath.split('/').filter(Boolean).pop() || 'folder';
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = `${folderName}.zip`;
            a.click();
            window.URL.revokeObjectURL(url);
        } catch (err) {
            setOpError(err.message);
        }
    }, []);

    // Batch Delete
    const batchDelete = useCallback(async (selectedNames, currentPath) => {
        if (!selectedNames || selectedNames.length === 0) return;

        const count = selectedNames.length;
        if (!window.confirm(`${count} Dateien/Ordner wirklich löschen?`)) return;

        try {
            for (const name of selectedNames) {
                // Mock item object
                await fileApi.deleteFile({ name }, currentPath);
            }
            await loadFiles(currentPath);
            await loadTrash();
        } catch (err) {
            setOpError(err.message);
        }
    }, [loadFiles, loadTrash]);

    // Move file
    const moveFile = useCallback(async (sourcePath, destinationPath, currentPath) => {
        try {
            await fileApi.moveFile(sourcePath, destinationPath);
            await loadFiles(currentPath);
            return true;
        } catch (err) {
            setOpError(err.message);
            return false;
        }
    }, [loadFiles]);

    return {
        uploading,
        opError,
        setOpError,
        uploadFiles,
        downloadFile,
        deleteFile,
        deleteFromTrash,
        emptyTrash,
        restoreFile,
        renameFile,
        createFolder,
        batchDownload,
        downloadFolderAsZip,
        batchDelete,
        moveFile
    };
}
