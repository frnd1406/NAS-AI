import { useState, useCallback, useRef } from 'react';
import { apiRequest } from '../lib/api';

const INITIAL_MESSAGE = {
    role: 'assistant',
    content: 'Hallo! Ich bin dein AI-Assistent. Frag mich etwas über deine Dokumente, z.B. "Wie hoch waren die Serverkosten?"',
    timestamp: new Date()
};

/**
 * Custom hook for chat session state management (SRP: State Logic Only)
 * Features:
 * - Message state management
 * - Loading/Error states
 * - AbortController for request cancellation
 * - Retry logic for gateway errors
 * 
 * @returns {{ messages, isLoading, error, input, setInput, sendMessage, clearSession }}
 */
export function useChatSession() {
    const [messages, setMessages] = useState([INITIAL_MESSAGE]);
    const [input, setInput] = useState('');
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState(null);

    // AbortController ref for cancelling in-flight requests
    const abortControllerRef = useRef(null);

    // Helper function with retry logic for 502/504 errors
    const fetchWithRetry = async (url, options, signal, retryCount = 0) => {
        try {
            return await apiRequest(url, { ...options, signal });
        } catch (err) {
            // Don't retry if aborted
            if (err.name === 'AbortError') throw err;

            const status = err.status || err.response?.status;
            const isGatewayError = status === 502 || status === 504;

            if (isGatewayError && retryCount < 1) {
                console.warn(`Gateway error ${status}, retrying in 1s...`);
                await new Promise(resolve => setTimeout(resolve, 1000));
                return fetchWithRetry(url, options, signal, retryCount + 1);
            }
            throw err;
        }
    };

    const sendMessage = useCallback(async (messageContent) => {
        if (!messageContent.trim() || isLoading) return;

        // Abort any previous request
        if (abortControllerRef.current) {
            abortControllerRef.current.abort();
        }

        // Create new AbortController for this request
        const controller = new AbortController();
        abortControllerRef.current = controller;

        const userMessage = {
            role: 'user',
            content: messageContent,
            timestamp: new Date()
        };

        setMessages(prev => [...prev, userMessage]);
        setInput('');
        setIsLoading(true);
        setError(null);

        try {
            const data = await fetchWithRetry(
                `/api/v1/ask?q=${encodeURIComponent(messageContent)}`,
                { method: 'GET' },
                controller.signal
            );

            // Skip if aborted
            if (controller.signal.aborted) return;

            const aiMessage = {
                role: 'assistant',
                content: data.answer,
                sources: data.cited_sources || [],
                confidence: data.confidence,
                timestamp: new Date()
            };

            setMessages(prev => [...prev, aiMessage]);
        } catch (err) {
            // Ignore abort errors (user sent new message)
            if (err.name === 'AbortError') {
                console.log('Request aborted - new message sent');
                return;
            }

            console.error('Chat error:', err);
            const errorMessage = err.message || 'Ein unbekannter Fehler ist aufgetreten.';
            setError(errorMessage);

            setMessages(prev => [...prev, {
                role: 'assistant',
                content: 'Entschuldigung, es gab einen Fehler bei der Verarbeitung deiner Anfrage. Bitte versuche es später erneut.',
                isError: true,
                timestamp: new Date()
            }]);
        } finally {
            // Only update loading if not aborted
            if (!controller.signal.aborted) {
                setIsLoading(false);
            }
        }
    }, [isLoading]);

    const clearSession = useCallback(() => {
        // Abort any in-flight request
        if (abortControllerRef.current) {
            abortControllerRef.current.abort();
        }
        setMessages([INITIAL_MESSAGE]);
        setInput('');
        setError(null);
        setIsLoading(false);
    }, []);

    return {
        messages,
        isLoading,
        error,
        input,
        setInput,
        sendMessage,
        clearSession
    };
}
