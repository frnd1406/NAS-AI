import { authHeaders, getApiBaseUrl } from '../lib/api';
import { joinPath } from '../utils/fileUtils';

const API_BASE = getApiBaseUrl();

export async function fetchFiles(path) {
    const res = await fetch(
        `${API_BASE}/api/v1/storage/files?path=${encodeURIComponent(path)}`,
        {
            credentials: 'include',
            headers: authHeaders()
        }
    );
    if (!res.ok) {
        if (res.status === 401) throw new Error('Unauthorized');
        throw new Error(`HTTP ${res.status}: Failed to load files`);
    }
    const data = await res.json();
    // Filter out .trash directory
    return (data.items || []).filter(item => item.name !== '.trash');
}

export async function fetchFileContent(filePath) {
    const res = await fetch(`${API_BASE}/api/v1/files/content?path=${encodeURIComponent(filePath)}`, {
        credentials: 'include',
        headers: authHeaders()
    });
    if (!res.ok) throw new Error('Failed to load file content');
    return await res.text();
}

export async function fetchTrash() {
    const res = await fetch(
        `${API_BASE}/api/v1/storage/trash`,
        {
            credentials: 'include',
            headers: authHeaders()
        }
    );
    if (!res.ok) {
        throw new Error(`HTTP ${res.status}: Failed to load trash`);
    }
    const data = await res.json();
    return data.items || [];
}

export async function deleteFile(item, currentPath) {
    const target = joinPath(currentPath, item.name);
    const res = await fetch(
        `${API_BASE}/api/v1/storage/delete?path=${encodeURIComponent(target)}`,
        {
            method: 'DELETE',
            credentials: 'include',
            headers: authHeaders(),
        }
    );
    if (!res.ok) {
        throw new Error(`Delete failed: HTTP ${res.status}`);
    }
    return true;
}

export async function deleteFromTrash(item) {
    const res = await fetch(
        `${API_BASE}/api/v1/storage/trash/delete?path=${encodeURIComponent(item.name)}`,
        {
            method: 'DELETE',
            credentials: 'include',
            headers: authHeaders(),
        }
    );
    if (!res.ok) {
        throw new Error(`Failed to delete ${item.name} from trash: ${res.status}`);
    }
    return true;
}

export async function restoreFile(item) {
    const res = await fetch(
        `${API_BASE}/api/v1/storage/restore?path=${encodeURIComponent(item.originalPath || item.name)}`,
        {
            method: 'POST',
            credentials: 'include',
            headers: authHeaders(),
        }
    );
    if (!res.ok) {
        throw new Error(`Restore failed: HTTP ${res.status}`);
    }
    return true;
}

export async function renameFile(oldPath, newPath) {
    const res = await fetch(
        `${API_BASE}/api/v1/storage/rename`,
        {
            method: 'POST',
            credentials: 'include',
            headers: {
                ...authHeaders(),
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ oldPath, newPath }),
        }
    );
    if (!res.ok) {
        throw new Error(`Rename failed: HTTP ${res.status}`);
    }
    return true;
}

export async function createFolder(folderPath) {
    const res = await fetch(`${API_BASE}/api/v1/storage/mkdir`, {
        method: 'POST',
        credentials: 'include',
        headers: {
            ...authHeaders(),
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ path: folderPath }),
    });

    if (!res.ok) {
        throw new Error(`Ordner erstellen fehlgeschlagen: HTTP ${res.status}`);
    }
    return true;
}

export async function moveFile(sourcePath, destinationPath) {
    const res = await fetch(`${API_BASE}/api/v1/storage/move`, {
        method: 'POST',
        credentials: 'include',
        headers: {
            ...authHeaders(),
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ sourcePath, destinationPath }),
    });

    if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data.error || `Move failed: HTTP ${res.status}`);
    }
    return true;
}

export async function downloadFileUrl(item, currentPath) {
    const target = joinPath(currentPath, item.name);
    const res = await fetch(
        `${API_BASE}/api/v1/storage/download?path=${encodeURIComponent(target)}`,
        {
            credentials: 'include',
            headers: authHeaders(),
        }
    );
    if (!res.ok) {
        throw new Error(`Download failed: HTTP ${res.status}`);
    }
    const blob = await res.blob();
    return window.URL.createObjectURL(blob);
}

export async function batchDownloadZip(paths) {
    if (!paths || paths.length === 0) {
        throw new Error("No files selected for download");
    }

    const res = await fetch(`${API_BASE}/api/v1/storage/batch-download`, {
        method: "POST",
        credentials: "include", // Send auth cookie
        headers: {
            ...authHeaders(),
            "Content-Type": "application/json",
        },
        body: JSON.stringify({ paths }),
    });

    if (!res.ok) {
        throw new Error(`Batch download failed: ${res.status}`);
    }

    return await res.blob();
}

export async function downloadFolderZip(path) {
    const res = await fetch(`${API_BASE}/api/v1/storage/download-zip?path=${encodeURIComponent(path)}`, {
        method: "GET",
        credentials: "include", // Send auth cookie
        headers: authHeaders(),
    });

    if (!res.ok) {
        throw new Error(`ZIP download failed: ${res.status}`);
    }

    return await res.blob();
}

export async function uploadFile(file, path, optionsOrEncryptionOverride = 'auto') {
    const form = new FormData();
    form.append('file', file);
    form.append('path', path);

    let encryptionOverride = 'auto';
    let options = {};

    // Handle backward compatibility or object options
    if (typeof optionsOrEncryptionOverride === 'string') {
        encryptionOverride = optionsOrEncryptionOverride;
    } else if (typeof optionsOrEncryptionOverride === 'object') {
        options = optionsOrEncryptionOverride;
        encryptionOverride = options.encryptionOverride || 'auto';
    }

    // Smart Upload Selector: Pass encryption override to backend
    // Backend expects: AUTO, FORCE_USER, FORCE_NONE
    if (encryptionOverride && encryptionOverride !== 'auto') {
        const backendValue = encryptionOverride === 'force' ? 'FORCE_USER' : 'FORCE_NONE';
        form.append('encryption_override', backendValue);
    }

    // Pass encryption password if provided (for FORCE_USER mode)
    if (options.encryptionPassword) {
        form.append('encryption_password', options.encryptionPassword);
    }

    const headers = authHeaders();
    delete headers['Content-Type']; // Let browser set boundary

    const res = await fetch(`${API_BASE}/api/v1/storage/upload`, {
        method: 'POST',
        body: form,
        credentials: 'include',
        headers: headers,
    });

    if (res.status === 401) {
        throw new Error('Unauthorized');
    }

    if (!res.ok) {
        const errorText = await res.text().catch(() => 'No error details');
        throw new Error(`Upload failed for ${file.name}: HTTP ${res.status} - ${errorText}`);
    }

    return true;
}
