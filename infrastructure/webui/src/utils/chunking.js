/**
 * Utility for client-side file chunking to support huge file uploads
 * without crashing the browser or server.
 */

// Default chunk size matches typical reliable upload size (50MB)
// Small enough for re-tries, large enough for efficiency.
const DESKTOP_CHUNK_SIZE = 50 * 1024 * 1024; // 50MB
const MOBILE_CHUNK_SIZE = 5 * 1024 * 1024;   // 5MB (Conservative for older iOS/Android)

/**
 * Determines optimal chunk size based on device capabilities
 * @returns {number} Chunk size in bytes
 */
export function getAdaptiveChunkSize() {
    // 1. Check for mobile User Agent
    const isMobile = /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent);

    // 2. Check Device Memory (if available, Chrome-only)
    // deviceMemory returns RAM in GB. If < 4GB, treat as constraints.
    const lowMemory = navigator.deviceMemory && navigator.deviceMemory < 4;

    if (isMobile || lowMemory) {
        console.log("📱 Mobile/Low-Memory device detected. Using conservative chunk size (5MB).");
        return MOBILE_CHUNK_SIZE;
    }

    return DESKTOP_CHUNK_SIZE;
}

const DEFAULT_CHUNK_SIZE = getAdaptiveChunkSize();

/**
 * Creates file chunks for processing/uploading
 * @param {File} file - The file object from input
 * @param {number} chunkSize - Size in bytes (default adaptive)
 * @returns {number} Total chunks count
 */
export function calculateTotalChunks(file, chunkSize = getAdaptiveChunkSize()) {
    return Math.ceil(file.size / chunkSize);
}

/**
 * Reads a specific chunk from a file
 * @param {File} file - The file object
 * @param {number} index - Chunk index (0-based)
 * @param {number} chunkSize - Size in bytes
 * @returns {Promise<ArrayBuffer>} The chunk data
 */
export async function readChunk(file, index, chunkSize = getAdaptiveChunkSize()) {
    const start = index * chunkSize;
    const end = Math.min(start + chunkSize, file.size);
    const blob = file.slice(start, end);

    return new Promise((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = (e) => resolve(e.target.result);
        reader.onerror = (e) => reject(e.target.error);
        reader.readAsArrayBuffer(blob);
    });
}

/**
 * Generator for iterating over file chunks
 * Usage: for await (const chunk of fileChunkIterator(file)) { ... }
 */
export async function* fileChunkIterator(file, chunkSize = getAdaptiveChunkSize()) {
    const totalChunks = calculateTotalChunks(file, chunkSize);

    for (let i = 0; i < totalChunks; i++) {
        const data = await readChunk(file, i, chunkSize);
        yield {
            index: i,
            total: totalChunks,
            data: data,
            isLast: i === totalChunks - 1
        };
    }
}
