// Custom hook for file preview functionality

import { useState, useCallback } from 'react';
import { authHeaders } from '../../utils/auth';
import { joinPath, isImage, isText } from '../../utils/fileUtils';
import { useVault } from '../../context/VaultContext';

const API_BASE = import.meta.env.VITE_API_BASE_URL || window.location.origin;

export function useFilePreview() {
    const [previewItem, setPreviewItem] = useState(null);
    const [previewContent, setPreviewContent] = useState(null);
    const [previewLoading, setPreviewLoading] = useState(false);

    // Get vault password (memory only)
    const { password } = useVault() || {};

    const openPreview = useCallback(async (item, currentPath) => {
        if (item.isDir) return;

        setPreviewItem(item);
        setPreviewLoading(true);
        setPreviewContent(null);

        const target = joinPath(currentPath, item.name);

        // Prepare headers with optional password
        const headers = authHeaders();
        if (password) {
            headers['X-Encryption-Password'] = password;
        }

        try {
            // Check based on CLEAN name (ignoring .enc)
            const cleanName = item.name.replace(/\.enc$/i, '');

            if (isImage(cleanName)) {
                const res = await fetch(
                    `${API_BASE}/api/v1/storage/download?path=${encodeURIComponent(target)}`,
                    {
                        credentials: 'include',
                        headers: headers,
                    }
                );

                if (res.status === 423) { // Locked
                    throw new Error("Vault locked");
                }

                if (res.ok) {
                    const blob = await res.blob();
                    const url = window.URL.createObjectURL(blob);
                    setPreviewContent({ type: 'image', url });
                } else {
                    console.error("Preview download failed", res.status);
                }
            } else if (isText(cleanName)) {
                const res = await fetch(
                    `${API_BASE}/api/v1/storage/download?path=${encodeURIComponent(target)}`,
                    {
                        credentials: 'include',
                        headers: headers,
                    }
                );

                if (res.status === 423) { // Locked
                    throw new Error("Vault locked");
                }

                if (res.ok) {
                    const text = await res.text();
                    setPreviewContent({ type: 'text', content: text });
                }
            } else {
                setPreviewContent({ type: 'text', content: "No preview available for this file type." });
            }
        } catch (err) {
            console.error('Preview failed:', err);
            setPreviewContent({ type: 'text', content: `Preview failed: ${err.message}` });
        } finally {
            setPreviewLoading(false);
        }
    }, [password]);

    const closePreview = useCallback(() => {
        if (previewContent?.type === 'image' && previewContent.url) {
            window.URL.revokeObjectURL(previewContent.url);
        }
        setPreviewItem(null);
        setPreviewContent(null);
    }, [previewContent]);

    return {
        previewItem,
        previewContent,
        previewLoading,
        openPreview,
        closePreview,
    };
}
