import React, { useRef, useEffect } from 'react';
import { Send, Bot, Loader2, Trash2, AlertCircle } from 'lucide-react';
import { useChatSession } from '../../hooks/useChatSession';
import { MessageList } from './MessageList';

/**
 * ChatInterface Component (SRP: Layout & UI Orchestration Only)
 * 
 * Responsibilities:
 * - Layout structure (header, messages area, input)
 * - Auto-scroll behavior
 * - Form submission handling
 * 
 * Delegations:
 * - State logic → useChatSession hook
 * - Message rendering → MessageList component
 */
const ChatInterface = () => {
    const {
        messages,
        isLoading,
        error,
        input,
        setInput,
        sendMessage,
        clearSession
    } = useChatSession();

    const messagesEndRef = useRef(null);

    // Auto-scroll to bottom when messages change
    useEffect(() => {
        messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
    }, [messages]);

    const handleSubmit = (e) => {
        e.preventDefault();
        sendMessage(input);
    };

    return (
        <div className="flex flex-col h-[600px] bg-[#0f172a] rounded-xl border border-white/10 overflow-hidden shadow-2xl">
            {/* Header */}
            <div className="p-4 border-b border-white/10 bg-[#1e293b]/50 backdrop-blur flex items-center justify-between">
                <div className="flex items-center gap-3">
                    <div className="w-8 h-8 rounded-lg bg-indigo-500/20 flex items-center justify-center text-indigo-400">
                        <Bot size={20} />
                    </div>
                    <div>
                        <h3 className="font-medium text-white">AI Knowledge Assistant</h3>
                        <p className="text-xs text-slate-400">Powered by Llama 3.2 & RAG</p>
                    </div>
                </div>
                {/* Clear Session Button */}
                {messages.length > 1 && (
                    <button
                        onClick={clearSession}
                        className="p-2 text-slate-400 hover:text-white hover:bg-white/10 rounded-lg transition-colors"
                        title="Chat leeren"
                    >
                        <Trash2 size={18} />
                    </button>
                )}
            </div>

            {/* Error Banner */}
            {error && (
                <div className="px-4 py-2 bg-red-500/10 border-b border-red-500/20 flex items-center gap-2 text-red-400 text-sm">
                    <AlertCircle size={16} />
                    <span>{error}</span>
                </div>
            )}

            {/* Messages Area (Delegated to MessageList) */}
            <MessageList
                messages={messages}
                isLoading={isLoading}
                messagesEndRef={messagesEndRef}
            />

            {/* Input Area */}
            <form onSubmit={handleSubmit} className="p-4 bg-[#1e293b]/50 border-t border-white/10">
                <div className="relative flex gap-2">
                    <input
                        type="text"
                        value={input}
                        onChange={(e) => setInput(e.target.value)}
                        placeholder="Stelle eine Frage an deine Dokumente..."
                        className="flex-1 bg-slate-900/50 border border-white/10 rounded-xl px-4 py-3 text-slate-200 placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500/50 focus:border-indigo-500/50 transition-all"
                        disabled={isLoading}
                    />
                    <button
                        type="submit"
                        disabled={!input.trim() || isLoading}
                        className="bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 disabled:cursor-not-allowed text-white p-3 rounded-xl transition-all shadow-lg shadow-indigo-500/20 flex items-center justify-center min-w-[50px]"
                    >
                        {isLoading ? <Loader2 size={20} className="animate-spin" /> : <Send size={20} />}
                    </button>
                </div>
            </form>
        </div>
    );
};

export default ChatInterface;
