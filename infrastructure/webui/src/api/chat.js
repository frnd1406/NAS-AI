// Chat API module
import { apiRequest } from '../lib/api';

/**
 * Send a question to the AI
 * @param {string} question - The user's question
 * @returns {Promise<Object>} - The AI response including answer and sources
 */
export async function askAI(question) {
    return apiRequest(`/api/v1/ask?q=${encodeURIComponent(question)}`);
}

/**
 * Fetch generic content for a file path (used by Chat to preview sources)
 * @param {string} path - File path
 * @returns {Promise<string>} - File content
 */
export async function fetchSourceContent(path) {
    const response = await fetch(`/api/v1/files/content?path=${encodeURIComponent(path)}`);
    if (!response.ok) {
        throw new Error('Failed to fetch content');
    }
    return response.text();
}
