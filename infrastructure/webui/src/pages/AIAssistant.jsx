import { useState, useRef, useEffect } from 'react';
import { Search, Loader2, Sparkles, FileText, ArrowRight, Brain, Zap } from 'lucide-react';

/**
 * AIAssistant - Dedicated full-page AI chat interface
 * Features:
 * - Search bar at top
 * - Answer on left with formatted file references
 * - Files panel on right (appears when results arrive)
 */
export default function AIAssistant() {
    const [query, setQuery] = useState('');
    const [isLoading, setIsLoading] = useState(false);
    const [hasResult, setHasResult] = useState(false);
    const [answer, setAnswer] = useState('');
    const [sources, setSources] = useState([]);
    const [confidence, setConfidence] = useState('');
    const inputRef = useRef(null);

    // Focus input on mount
    useEffect(() => {
        inputRef.current?.focus();
    }, []);

    const handleSubmit = async (e) => {
        e.preventDefault();
        if (!query.trim() || isLoading) return;

        setIsLoading(true);
        setHasResult(false);
        setAnswer('');
        setSources([]);

        try {
            const response = await fetch(`/api/v1/ask?q=${encodeURIComponent(query.trim())}`);
            if (!response.ok) throw new Error('API Error');

            const data = await response.json();

            setAnswer(data.answer || 'Keine Antwort verfügbar.');
            setSources(data.cited_sources || []);
            setConfidence(data.confidence || 'UNBEKANNT');
            setHasResult(true);

        } catch (error) {
            console.error('AI Error:', error);
            setAnswer('Entschuldigung, ich konnte keine Verbindung zum AI-Service herstellen.');
            setHasResult(true);
        } finally {
            setIsLoading(false);
        }
    };

    // Format answer text: highlight file references like [filename.txt]
    const formatAnswer = (text) => {
        if (!text) return null;

        // Split by file references pattern [filename]
        const parts = text.split(/(\[[^\]]+\.txt\]|\[[^\]]+\.json\]|\[[^\]]+\.md\])/g);

        return parts.map((part, index) => {
            if (part.match(/^\[.+\.(txt|json|md)\]$/)) {
                const filename = part.slice(1, -1);
                return (
                    <span
                        key={index}
                        className="inline-flex items-center gap-1 px-2 py-0.5 mx-1 bg-blue-500/20 text-blue-400 rounded-md text-sm font-medium border border-blue-500/30 hover:bg-blue-500/30 transition-colors cursor-pointer"
                    >
                        <FileText size={12} />
                        {filename}
                    </span>
                );
            }
            return <span key={index}>{part}</span>;
        });
    };

    return (
        <div className="min-h-[calc(100vh-120px)] flex flex-col">
            {/* Header with Search */}
            <div className="mb-4 sm:mb-8">
                <div className="flex items-center gap-3 mb-4 sm:mb-6">
                    <div className="p-2 sm:p-3 bg-gradient-to-br from-blue-500 to-violet-600 rounded-lg sm:rounded-xl shadow-lg shadow-blue-500/20">
                        <Brain size={24} className="text-white sm:hidden" />
                        <Brain size={28} className="text-white hidden sm:block" />
                    </div>
                    <div>
                        <h1 className="text-xl sm:text-2xl font-bold text-white">NAS.AI Assistant</h1>
                        <p className="text-xs sm:text-sm text-slate-400 hidden sm:block">Intelligente Dokumentensuche mit RAG</p>
                    </div>
                </div>

                {/* Search Bar */}
                <form onSubmit={handleSubmit} className="relative">
                    <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-2 sm:gap-3 p-2 bg-slate-800/80 backdrop-blur-sm rounded-xl sm:rounded-2xl border border-white/10 shadow-xl focus-within:border-blue-500/50 focus-within:shadow-blue-500/10 transition-all">
                        <div className="flex items-center gap-2 flex-1">
                            <div className="pl-2 sm:pl-4">
                                <Search size={20} className="text-slate-400 sm:hidden" />
                                <Search size={22} className="text-slate-400 hidden sm:block" />
                            </div>
                            <input
                                ref={inputRef}
                                type="text"
                                value={query}
                                onChange={(e) => setQuery(e.target.value)}
                                placeholder="Frage stellen..."
                                className="flex-1 bg-transparent text-white text-base sm:text-lg py-2 sm:py-3 outline-none placeholder:text-slate-500"
                                disabled={isLoading}
                            />
                        </div>
                        <button
                            type="submit"
                            disabled={isLoading || !query.trim()}
                            className="flex items-center justify-center gap-2 px-4 sm:px-6 py-2.5 sm:py-3 bg-gradient-to-r from-blue-600 to-violet-600 text-white font-semibold rounded-lg sm:rounded-xl hover:from-blue-500 hover:to-violet-500 disabled:opacity-50 disabled:cursor-not-allowed transition-all shadow-lg hover:shadow-blue-500/25 w-full sm:w-auto"
                        >
                            {isLoading ? (
                                <>
                                    <Loader2 size={18} className="animate-spin" />
                                    <span className="hidden sm:inline">Suche...</span>
                                </>
                            ) : (
                                <>
                                    <Sparkles size={18} />
                                    <span>Fragen</span>
                                </>
                            )}
                        </button>
                    </div>
                </form>
            </div>

            {/* Results Area */}
            {(isLoading || hasResult) && (
                <div className={`flex-1 grid gap-4 sm:gap-6 animate-fade-in-scale ${sources.length > 0 ? 'grid-cols-1 lg:grid-cols-3' : 'grid-cols-1'}`}>

                    {/* Left: Answer Panel */}
                    <div className={`${sources.length > 0 ? 'lg:col-span-2' : ''} bg-slate-800/60 backdrop-blur-sm rounded-xl sm:rounded-2xl border border-white/10 p-4 sm:p-6 shadow-xl order-2 lg:order-1`}>
                        <div className="flex items-center gap-2 mb-3 sm:mb-4 pb-3 sm:pb-4 border-b border-white/10">
                            <div className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" />
                            <span className="font-semibold text-white text-sm sm:text-base">NAS.AI Antwort</span>
                            {confidence && hasResult && (
                                <span className={`ml-auto text-xs px-2 py-1 rounded-full ${confidence === 'HOCH' ? 'bg-emerald-500/20 text-emerald-400' :
                                    confidence === 'MITTEL' ? 'bg-amber-500/20 text-amber-400' :
                                        'bg-slate-500/20 text-slate-400'
                                    }`}>
                                    {confidence}
                                </span>
                            )}
                        </div>

                        {isLoading ? (
                            <div className="flex items-center gap-3 text-slate-400">
                                <Loader2 size={20} className="animate-spin text-blue-400" />
                                <span className="text-sm sm:text-base">Analysiere Dokumente...</span>
                            </div>
                        ) : (
                            <div className="text-slate-200 leading-relaxed text-base sm:text-lg whitespace-pre-wrap">
                                {formatAnswer(answer)}
                            </div>
                        )}
                    </div>

                    {/* Right: Files Panel */}
                    {sources.length > 0 && (
                        <div className="bg-slate-800/60 backdrop-blur-sm rounded-xl sm:rounded-2xl border border-white/10 p-3 sm:p-5 shadow-xl animate-slide-in-right order-1 lg:order-2">
                            <div className="flex items-center gap-2 mb-3 sm:mb-4 pb-3 sm:pb-4 border-b border-white/10">
                                <Zap size={16} className="text-cyan-400 sm:hidden" />
                                <Zap size={18} className="text-cyan-400 hidden sm:block" />
                                <span className="font-semibold text-white text-sm sm:text-base">Quellen</span>
                                <span className="ml-auto text-xs text-slate-400">{sources.length} Dok.</span>
                            </div>

                            <div className="flex flex-col gap-2 sm:gap-3">
                                {sources.map((source, index) => {
                                    const fileName = source.file_path?.split('/').pop() || source.file_id;
                                    const similarity = source.similarity || 0;
                                    const percent = Math.round(similarity * 100);

                                    return (
                                        <button
                                            key={index}
                                            className="group flex items-center gap-2 sm:gap-3 p-3 sm:p-4 bg-slate-700/50 hover:bg-blue-600/30 active:bg-blue-600/40 rounded-lg sm:rounded-xl border border-white/5 hover:border-blue-500/40 transition-all duration-200 text-left animate-slide-in-right touch-manipulation"
                                            style={{ animationDelay: `${index * 100}ms` }}
                                        >
                                            <div className="p-1.5 sm:p-2 bg-blue-500/20 rounded-md sm:rounded-lg group-hover:bg-blue-500/30 transition-colors">
                                                <FileText size={16} className="text-blue-400 sm:hidden" />
                                                <FileText size={18} className="text-blue-400 hidden sm:block" />
                                            </div>
                                            <div className="flex-1 min-w-0">
                                                <p className="text-white font-medium truncate text-sm sm:text-base">{fileName}</p>
                                                <p className="text-xs text-slate-400 truncate hidden sm:block">{source.file_path}</p>
                                            </div>
                                            <span className={`text-xs font-semibold px-2 py-0.5 rounded-full ${percent >= 70 ? 'bg-emerald-500/20 text-emerald-400' :
                                                percent >= 50 ? 'bg-blue-500/20 text-blue-400' :
                                                    'bg-amber-500/20 text-amber-400'
                                                }`}>
                                                {percent}%
                                            </span>
                                        </button>
                                    );
                                })}
                            </div>
                        </div>
                    )}
                </div>
            )}

            {/* Empty State */}
            {!isLoading && !hasResult && (
                <div className="flex-1 flex flex-col items-center justify-center text-center py-10 sm:py-20 px-4">
                    <div className="p-4 sm:p-6 bg-gradient-to-br from-blue-500/10 to-violet-500/10 rounded-full mb-4 sm:mb-6 border border-white/5">
                        <Brain size={36} className="text-blue-400 sm:hidden" />
                        <Brain size={48} className="text-blue-400 hidden sm:block" />
                    </div>
                    <h2 className="text-lg sm:text-xl font-semibold text-white mb-2">Frag mich etwas!</h2>
                    <p className="text-sm sm:text-base text-slate-400 max-w-md mb-6 sm:mb-8">
                        Ich durchsuche deine Dokumente mit KI.
                    </p>
                    <div className="flex flex-wrap gap-2 justify-center max-w-full overflow-x-auto pb-2">
                        {[
                            'Krypto-Gewinne?',
                            'Finns Noten',
                            'Rechnungen',
                            'Nächste Reise'
                        ].map((suggestion, i) => (
                            <button
                                key={i}
                                onClick={() => setQuery(suggestion)}
                                className="px-3 sm:px-4 py-2 bg-slate-800/60 hover:bg-slate-700/60 active:bg-slate-600/60 text-slate-300 hover:text-white rounded-full text-xs sm:text-sm border border-white/10 hover:border-white/20 transition-all whitespace-nowrap touch-manipulation"
                            >
                                {suggestion}
                            </button>
                        ))}
                    </div>
                </div>
            )}
        </div>
    );
}
