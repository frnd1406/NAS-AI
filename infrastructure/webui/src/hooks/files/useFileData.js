// Hook for fetching file data
import { useState, useCallback } from 'react';
import * as fileApi from '../../api/files';

export function useFileData(initialPath = '/') {
    const [files, setFiles] = useState([]);
    const [trashedFiles, setTrashedFiles] = useState([]);
    const [path, setPath] = useState(initialPath);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState('');

    // Load files from a directory
    const loadFiles = useCallback(async (target = path) => {
        setLoading(true);
        setError('');
        try {
            const fetchedFiles = await fileApi.fetchFiles(target);
            setFiles(fetchedFiles);
            // Also update path if successful (e.g. if target was different from current path)
            // But usually we set path first.
        } catch (err) {
            setError(err.message || 'Unknown error');
            if (err.message === 'Unauthorized') {
                window.location.href = '/login';
            }
        } finally {
            setLoading(false);
        }
    }, [path]);

    // Load trashed files
    const loadTrash = useCallback(async () => {
        try {
            const fetchedTrash = await fileApi.fetchTrash();
            setTrashedFiles(fetchedTrash);
        } catch (err) {
            console.error('Failed to load trash:', err);
        }
    }, []);

    // Navigate to directory
    const navigateTo = useCallback((newPath) => {
        setPath(newPath);
        // We trigger loadFiles immediately with the new path
        // Note: loadFiles depends on [path]. If we call loadFiles(newPath), it uses the arg.
        // But if we just setPath, the effect in useFileManager might trigger loadFiles?
        // useFileStorage handled this by calling loadFiles(newPath) manually inside navigateTo.

        // However, useFileData should separate state update from side effect if possible,
        // but to keep consistent behavior with previous hook:
        loadFiles(newPath);
    }, [loadFiles]);

    const refresh = useCallback(() => loadFiles(path), [loadFiles, path]);

    return {
        files,
        trashedFiles,
        path,
        loading,
        error,
        setError,
        loadFiles,
        loadTrash,
        navigateTo,
        refresh
    };
}
