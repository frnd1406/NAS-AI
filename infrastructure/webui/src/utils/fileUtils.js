// File utilities for the Files page

/**
 * Join two path segments safely
 */
export function joinPath(base, name) {
    if (!base || base === "/") {
        return `/${name}`;
    }
    return `${base.replace(/\/+$/, "")}/${name}`;
}

/**
 * Format file size in human-readable format
 */
export function formatFileSize(bytes) {
    if (!bytes || bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

// File extension categories
const IMAGE_EXTENSIONS = ['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg', 'bmp'];
const TEXT_EXTENSIONS = ['txt', 'md', 'log', 'json', 'xml', 'yaml', 'yml', 'csv'];
const CODE_EXTENSIONS = ['js', 'jsx', 'ts', 'tsx', 'py', 'go', 'rs', 'java', 'cpp', 'c', 'h', 'css', 'html', 'sh'];
const ARCHIVE_EXTENSIONS = ['zip', 'tar', 'gz', 'rar', '7z', 'bz2'];
const VIDEO_EXTENSIONS = ['mp4', 'avi', 'mkv', 'mov', 'wmv', 'flv', 'webm'];
const AUDIO_EXTENSIONS = ['mp3', 'wav', 'flac', 'ogg', 'aac', 'm4a'];

/**
 * Get file extension from filename
 * Strips .enc suffix for type detection
 */
export function getFileExtension(name) {
    if (!name) return '';
    // Strip .enc if present to detect underlying type
    const cleanName = name.replace(/\.enc$/i, '');
    return cleanName.split('.').pop()?.toLowerCase() || '';
}

/**
 * Check if file is an image
 */
export function isImage(name) {
    return IMAGE_EXTENSIONS.includes(getFileExtension(name));
}

/**
 * Check if file is a text/code file
 */
export function isText(name) {
    const ext = getFileExtension(name);
    return TEXT_EXTENSIONS.includes(ext) || CODE_EXTENSIONS.includes(ext);
}

/**
 * Check if file is a video
 */
export function isVideo(name) {
    return VIDEO_EXTENSIONS.includes(getFileExtension(name));
}

/**
 * Check if file is audio
 */
export function isAudio(name) {
    return AUDIO_EXTENSIONS.includes(getFileExtension(name));
}

/**
 * Check if file is an archive
 */
export function isArchive(name) {
    return ARCHIVE_EXTENSIONS.includes(getFileExtension(name));
}

/**
 * Check if file is code
 */
export function isCode(name) {
    return CODE_EXTENSIONS.includes(getFileExtension(name));
}

/**
 * Get file type category
 */
export function getFileType(name, isDir) {
    if (isDir) return 'folder';

    const ext = getFileExtension(name);

    if (IMAGE_EXTENSIONS.includes(ext)) return 'image';
    if (TEXT_EXTENSIONS.includes(ext)) return 'text';
    if (CODE_EXTENSIONS.includes(ext)) return 'code';
    if (ARCHIVE_EXTENSIONS.includes(ext)) return 'archive';
    if (VIDEO_EXTENSIONS.includes(ext)) return 'video';
    if (AUDIO_EXTENSIONS.includes(ext)) return 'audio';

    return 'file';
}

/**
 * Check if file can be previewed
 * Handles .enc files by stripping extension for check
 */
export function canPreview(name) {
    // Strip .enc suffix transparency
    const cleanName = name.replace(/\.enc$/i, '');
    return isImage(cleanName) || isText(cleanName);
}

/**
 * Get breadcrumbs from path
 */
export function getBreadcrumbs(path) {
    if (path === '/') return [{ name: 'Home', path: '/' }];

    const parts = path.split('/').filter(Boolean);
    const breadcrumbs = [{ name: 'Home', path: '/' }];

    let currentPath = '';
    parts.forEach((part) => {
        currentPath += `/${part}`;
        breadcrumbs.push({ name: part, path: currentPath });
    });

    return breadcrumbs;
}
