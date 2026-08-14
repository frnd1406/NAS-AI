import { useState } from 'react';
import { askAI, fetchSourceContent } from '../api/chat';

export function useChatWidget() {
    const [isOpen, setIsOpen] = useState(false);
    const [input, setInput] = useState('');
    const [isLoading, setIsLoading] = useState(false);
    const [messages, setMessages] = useState([
        {
            id: 1,
            role: 'ai',
            text: 'Hallo! Ich bin dein NAS.AI Assistant. Ich kann dir helfen, Informationen in deinen Dokumenten zu finden. Frag mich etwas!',
            type: 'text'
        }
    ]);

    // File preview state
    const [previewFiles, setPreviewFiles] = useState([]);
    const [currentPreviewIndex, setCurrentPreviewIndex] = useState(0);
    const [showPreview, setShowPreview] = useState(false);

    const toggleOpen = () => setIsOpen(!isOpen);

    const handleFileSelect = (messageFiles, fileIndex) => {
        setPreviewFiles(messageFiles);
        setCurrentPreviewIndex(fileIndex);
        setShowPreview(true);
    };

    const handlePreviewNavigate = (newIndex) => {
        setCurrentPreviewIndex(newIndex);
    };

    const closePreview = () => setShowPreview(false);

    const sendMessage = async (e) => {
        if (e) e.preventDefault();
        if (!input.trim() || isLoading) return;

        const userMessage = input.trim();
        setMessages(prev => [...prev, { id: Date.now(), role: 'user', text: userMessage, type: 'text' }]);
        setInput('');
        setIsLoading(true);

        try {
            const data = await askAI(userMessage);

            // Extract sources if available
            const sources = data.cited_sources || [];
            const answer = data.answer || "Ich konnte keine Antwort generieren.";

            // Add AI response
            setMessages(prev => [...prev, {
                id: Date.now() + 1,
                role: 'ai',
                text: answer,
                type: 'text'
            }]);

            // If we have sources, fetch content and show selector
            if (sources.length > 0) {
                const filesWithContent = await Promise.all(
                    sources.map(async (source) => {
                        try {
                            const content = await fetchSourceContent(source.file_path);
                            return {
                                ...source,
                                content
                            };
                        } catch {
                            return source;
                        }
                    })
                );

                setMessages(prev => [...prev, {
                    id: Date.now() + 2,
                    role: 'ai',
                    type: 'file_selector',
                    files: filesWithContent
                }]);
            }

        } catch (error) {
            console.error("Chat error:", error);
            setMessages(prev => [...prev, {
                id: Date.now() + 1,
                role: 'ai',
                text: 'Entschuldigung, ich konnte den Knowledge Agent nicht erreichen. Bitte versuche es später erneut.',
                type: 'text'
            }]);
        } finally {
            setIsLoading(false);
        }
    };

    return {
        // State
        isOpen,
        input,
        isLoading,
        messages,
        previewFiles,
        currentPreviewIndex,
        showPreview,

        // Actions
        toggleOpen,
        setInput,
        sendMessage,
        handleFileSelect,
        handlePreviewNavigate,
        closePreview
    };
}
