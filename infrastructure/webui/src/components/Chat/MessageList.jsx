import React from 'react';
import { Bot, User, Loader2, FileText, CheckCircle2 } from 'lucide-react';

/**
 * Confidence Badge Component
 */
const ConfidenceBadge = ({ level }) => {
    if (!level) return null;

    const colors = {
        'HOCH': 'bg-green-500/20 text-green-400 border-green-500/30',
        'MITTEL': 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30',
        'NIEDRIG': 'bg-red-500/20 text-red-400 border-red-500/30'
    };

    return (
        <span className={`text-xs px-2 py-0.5 rounded border ${colors[level] || colors['MITTEL']} ml-2`}>
            {level}
        </span>
    );
};

/**
 * Single Chat Bubble Component (SRP: Render one message)
 */
const ChatBubble = ({ message }) => {
    const { role, content, sources, confidence, timestamp, isError, isLoading: msgLoading } = message;

    // Ghost Message Filter
    if (!content && !msgLoading && !isError) return null;

    const isUser = role === 'user';

    return (
        <div className={`flex gap-4 ${isUser ? 'flex-row-reverse' : ''}`}>
            {/* Avatar */}
            <div className={`w-8 h-8 rounded-lg flex-shrink-0 flex items-center justify-center ${isUser ? 'bg-indigo-500 text-white' : 'bg-slate-700 text-indigo-400'
                }`}>
                {isUser ? <User size={18} /> : <Bot size={18} />}
            </div>

            {/* Message Bubble */}
            <div className={`flex flex-col max-w-[80%] ${isUser ? 'items-end' : 'items-start'}`}>
                <div className={`rounded-2xl px-4 py-3 ${isUser
                        ? 'bg-indigo-600 text-white rounded-tr-none'
                        : 'bg-slate-800/80 text-slate-200 rounded-tl-none border border-white/5'
                    }`}>
                    <p className="whitespace-pre-wrap leading-relaxed">{content}</p>
                </div>

                {/* Metadata (Sources & Confidence) for AI messages */}
                {!isUser && !isError && (
                    <div className="mt-2 space-y-2 w-full">
                        {/* Confidence Badge */}
                        {confidence && (
                            <div className="flex items-center gap-2 text-xs text-slate-400">
                                <span>Konfidenz:</span>
                                <ConfidenceBadge level={confidence} />
                            </div>
                        )}

                        {/* Sources List */}
                        {sources && sources.length > 0 && (
                            <div className="bg-slate-900/50 rounded-lg p-3 border border-white/5 text-sm">
                                <div className="flex items-center gap-2 text-slate-400 mb-2 text-xs uppercase tracking-wider font-medium">
                                    <FileText size={12} />
                                    Quellen
                                </div>
                                <div className="space-y-2">
                                    {sources.map((source, sIdx) => (
                                        <div key={sIdx} className="flex items-start gap-2 text-slate-300 bg-white/5 p-2 rounded hover:bg-white/10 transition-colors cursor-pointer group">
                                            <div className="mt-0.5 text-indigo-400 group-hover:text-indigo-300">
                                                <CheckCircle2 size={14} />
                                            </div>
                                            <div className="flex-1 min-w-0">
                                                <p className="truncate font-medium text-xs text-indigo-300">{source.file_id}</p>
                                                <p className="text-xs text-slate-500 mt-0.5">Match: {Math.round(source.similarity * 100)}%</p>
                                            </div>
                                        </div>
                                    ))}
                                </div>
                            </div>
                        )}
                    </div>
                )}

                <span className="text-[10px] text-slate-500 mt-1 px-1">
                    {timestamp.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                </span>
            </div>
        </div>
    );
};

/**
 * Loading Indicator Component
 */
const LoadingBubble = () => (
    <div className="flex gap-4">
        <div className="w-8 h-8 rounded-lg bg-slate-700 text-indigo-400 flex-shrink-0 flex items-center justify-center">
            <Bot size={18} />
        </div>
        <div className="bg-slate-800/80 rounded-2xl rounded-tl-none px-4 py-3 border border-white/5 flex items-center gap-2">
            <Loader2 size={16} className="animate-spin text-indigo-400" />
            <span className="text-slate-400 text-sm">Analysiere Dokumente...</span>
        </div>
    </div>
);

/**
 * Message List Component (SRP: Render all messages)
 */
export function MessageList({ messages, isLoading, messagesEndRef }) {
    return (
        <div className="flex-1 overflow-y-auto p-4 space-y-6 scrollbar-thin scrollbar-thumb-white/10 scrollbar-track-transparent">
            {messages.map((msg, idx) => (
                <ChatBubble key={idx} message={msg} />
            ))}
            {isLoading && <LoadingBubble />}
            <div ref={messagesEndRef} />
        </div>
    );
}
