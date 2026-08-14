import { useState, useEffect } from "react";
import { useSearchParams, useNavigate } from "react-router-dom";
import {
  Search as SearchIcon,
  FileText,
  Loader2,
  Sparkles,
  Brain,
  MessageSquare,
  CheckCircle2,
  ArrowRight
} from "lucide-react";
import { queryAI } from "../lib/api";

/**
 * Unified AI Knowledge Search
 * 
 * One input field - AI decides whether to:
 * - Return search results (files matching the query)
 * - Generate an answer (for questions)
 * 
 * Dynamic result limits based on AI intent classification.
 */
export default function Search() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const [query, setQuery] = useState(searchParams.get('q') || "");
  const [results, setResults] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [isFocused, setIsFocused] = useState(false);

  // Auto-search if query comes from URL
  useEffect(() => {
    const urlQuery = searchParams.get('q');
    if (urlQuery && urlQuery.trim()) {
      setQuery(urlQuery);
      performQuery(urlQuery);
    }
  }, [searchParams]);

  const performQuery = async (searchQuery) => {
    if (!searchQuery.trim()) {
      setError("Bitte geben Sie einen Suchbegriff ein");
      return;
    }

    setLoading(true);
    setError("");
    setResults(null);

    try {
      const data = await queryAI(searchQuery);
      setResults(data);
    } catch (err) {
      setError(err.message || "Fehler bei der Anfrage");
      console.error("Query error:", err);
    } finally {
      setLoading(false);
    }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    performQuery(query);
    // Update URL for sharing
    navigate(`/search?q=${encodeURIComponent(query)}`, { replace: true });
  };

  const highlightText = (text, searchQuery) => {
    if (!searchQuery || !text) return text;

    const words = searchQuery.toLowerCase().split(/\s+/);
    let highlighted = text;

    words.forEach(word => {
      if (word.length > 2) { // Only highlight words > 2 chars
        const regex = new RegExp(`(${word})`, 'gi');
        highlighted = highlighted.replace(regex, '<mark class="bg-yellow-500/30 text-yellow-200 rounded px-0.5">$1</mark>');
      }
    });

    return highlighted;
  };

  const IntentBadge = ({ intent }) => {
    if (!intent) return null;

    const isSearch = intent.type === "search";

    return (
      <div className={`inline-flex items-center gap-2 px-3 py-1.5 rounded-full text-xs font-medium ${isSearch
          ? 'bg-blue-500/20 text-blue-300 border border-blue-500/30'
          : 'bg-violet-500/20 text-violet-300 border border-violet-500/30'
        }`}>
        {isSearch ? <SearchIcon size={12} /> : <Brain size={12} />}
        <span>{isSearch ? 'Suche' : 'AI Antwort'}</span>
        <span className="opacity-60">•</span>
        <span className="opacity-80">{intent.limit} Dokumente</span>
      </div>
    );
  };

  const ConfidenceBadge = ({ level }) => {
    if (!level) return null;

    const config = {
      'HOCH': { bg: 'bg-green-500/20', text: 'text-green-400', border: 'border-green-500/30' },
      'MITTEL': { bg: 'bg-yellow-500/20', text: 'text-yellow-400', border: 'border-yellow-500/30' },
      'NIEDRIG': { bg: 'bg-red-500/20', text: 'text-red-400', border: 'border-red-500/30' }
    };

    const colors = config[level] || config['MITTEL'];

    return (
      <span className={`inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded border ${colors.bg} ${colors.text} ${colors.border}`}>
        <CheckCircle2 size={10} />
        Konfidenz: {level}
      </span>
    );
  };

  return (
    <div className="space-y-8 max-w-5xl mx-auto">
      {/* Header */}
      <div className="text-center mb-8">
        <div className="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-gradient-to-br from-indigo-500 to-violet-600 mb-4 shadow-lg shadow-indigo-500/25">
          <Brain size={32} className="text-white" />
        </div>
        <h1 className="text-4xl font-bold text-white mb-3 tracking-tight">
          NAS.AI Knowledge Search
        </h1>
        <p className="text-slate-400 max-w-lg mx-auto">
          Frag mich alles – ich finde die passenden Dokumente oder beantworte deine Frage direkt
        </p>
      </div>

      {/* Unified Search Box */}
      <div className="relative w-full max-w-3xl mx-auto">
        <div className={`relative group transition-all duration-300 ${isFocused ? 'scale-[1.01]' : ''}`}>
          {/* Glow Background */}
          <div className={`absolute -inset-1 bg-gradient-to-r from-indigo-500 via-violet-500 to-purple-500 rounded-2xl blur-lg opacity-20 group-hover:opacity-40 transition duration-500 ${loading ? 'animate-pulse' : ''}`} />

          <form onSubmit={handleSubmit} className="relative flex items-center bg-slate-900/90 border border-white/10 rounded-xl shadow-2xl backdrop-blur-xl">
            <div className="pl-5 text-slate-400">
              {loading ? (
                <div className="relative">
                  <Brain className="text-violet-400" size={24} />
                  <span className="absolute -top-1 -right-1 w-2 h-2 bg-violet-400 rounded-full animate-ping" />
                </div>
              ) : (
                <Sparkles className={`transition-colors ${isFocused ? 'text-violet-400' : 'text-slate-500'}`} size={24} />
              )}
            </div>
            <input
              type="text"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              onFocus={() => setIsFocused(true)}
              onBlur={() => setIsFocused(false)}
              placeholder="Was möchtest du wissen? (z.B. 'Alle Rechnungen 2024' oder 'Was kosten unsere Server?')"
              className="w-full bg-transparent text-white px-4 py-5 focus:outline-none placeholder-slate-500 text-lg"
              disabled={loading}
            />
            <div className="pr-4 flex items-center gap-2">
              <kbd className="hidden sm:inline-block px-2.5 py-1 text-xs font-semibold text-slate-500 bg-slate-800 border border-slate-700 rounded-lg">↵</kbd>
              <button
                type="submit"
                disabled={loading || !query.trim()}
                className="p-3 rounded-xl bg-gradient-to-r from-indigo-600 to-violet-600 hover:from-indigo-500 hover:to-violet-500 text-white disabled:opacity-50 disabled:cursor-not-allowed transition-all shadow-lg shadow-indigo-500/20"
              >
                {loading ? <Loader2 size={20} className="animate-spin" /> : <ArrowRight size={20} />}
              </button>
            </div>
          </form>
        </div>

        {/* Error Message */}
        {error && (
          <div className="mt-4 p-4 bg-red-500/10 border border-red-500/20 rounded-xl">
            <p className="text-red-400">{error}</p>
          </div>
        )}
      </div>

      {/* AI Thinking Indicator */}
      {loading && (
        <div className="max-w-3xl mx-auto">
          <div className="p-4 bg-gradient-to-r from-indigo-500/10 via-violet-500/15 to-indigo-500/10 rounded-xl border border-violet-500/20 animate-pulse">
            <div className="flex items-center gap-3">
              <Brain className="text-violet-400 animate-pulse" size={20} />
              <span className="text-violet-300 text-sm font-medium">AI analysiert deine Anfrage...</span>
            </div>
          </div>
        </div>
      )}

      {/* Results */}
      {results && (
        <div className="space-y-6 max-w-4xl mx-auto animate-fade-in-scale">
          {/* Intent Badge */}
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <IntentBadge intent={results.intent} />
              {results.confidence && <ConfidenceBadge level={results.confidence} />}
            </div>
            <span className="text-sm text-slate-500">
              {results.mode === 'search'
                ? `${results.files?.length || 0} Ergebnisse`
                : `${results.all_candidates || 0} Quellen analysiert`
              }
            </span>
          </div>

          {/* ANSWER MODE */}
          {results.mode === 'answer' && (
            <div className="space-y-4">
              {/* AI Answer Box */}
              <div className="relative">
                <div className="absolute -inset-[1px] bg-gradient-to-r from-indigo-500 via-violet-500 to-purple-500 rounded-2xl opacity-40" />
                <div className="relative p-6 bg-slate-900/95 rounded-2xl backdrop-blur-xl">
                  <div className="flex items-start gap-4">
                    <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-indigo-500 to-violet-600 flex items-center justify-center flex-shrink-0 shadow-lg shadow-indigo-500/25">
                      <MessageSquare size={20} className="text-white" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="text-white text-lg leading-relaxed whitespace-pre-wrap">
                        {results.answer}
                      </p>
                    </div>
                  </div>
                </div>
              </div>

              {/* Sources */}
              {results.sources && results.sources.length > 0 && (
                <div className="bg-slate-800/50 rounded-xl p-4 border border-white/5">
                  <div className="flex items-center gap-2 text-slate-400 mb-3 text-xs uppercase tracking-wider font-medium">
                    <FileText size={12} />
                    Quellen ({results.sources.length})
                  </div>
                  <div className="grid gap-2 sm:grid-cols-2">
                    {results.sources.map((source, idx) => (
                      <div
                        key={idx}
                        className="flex items-center gap-3 p-3 bg-white/5 rounded-lg hover:bg-white/10 transition-colors cursor-pointer group"
                      >
                        <div className="w-8 h-8 rounded-lg bg-indigo-500/20 flex items-center justify-center">
                          <FileText size={14} className="text-indigo-400" />
                        </div>
                        <div className="flex-1 min-w-0">
                          <p className="text-sm text-white truncate group-hover:text-indigo-300 transition-colors">
                            {source.file_id || source.file_path?.split('/').pop()}
                          </p>
                          <p className="text-xs text-slate-500">
                            {Math.round(source.similarity * 100)}% Match
                          </p>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}

          {/* SEARCH MODE */}
          {results.mode === 'search' && results.files && (
            <div className="space-y-3">
              {results.files.length > 0 ? (
                results.files.map((file, idx) => {
                  const similarity = Math.round(file.similarity * 100);
                  const snippet = file.content?.length > 250
                    ? file.content.substring(0, 250) + "..."
                    : file.content;

                  return (
                    <div
                      key={idx}
                      className="group p-5 bg-slate-800/50 backdrop-blur-sm rounded-xl border border-white/5
                        hover:border-indigo-500/30 hover:bg-slate-800/70 transition-all cursor-pointer
                        hover:shadow-lg hover:shadow-indigo-500/5"
                      style={{ animationDelay: `${idx * 50}ms` }}
                    >
                      {/* File Header */}
                      <div className="flex items-start justify-between mb-3">
                        <div className="flex items-center gap-3">
                          <div className="p-2.5 bg-indigo-500/20 rounded-lg group-hover:bg-indigo-500/30 transition-colors">
                            <FileText size={18} className="text-indigo-400" />
                          </div>
                          <div>
                            <h3 className="font-semibold text-white group-hover:text-indigo-300 transition-colors">
                              {file.file_id || file.file_path?.split('/').pop()}
                            </h3>
                            <p className="text-xs text-slate-500 truncate max-w-xs">
                              {file.file_path?.replace('/mnt/data/', '')}
                            </p>
                          </div>
                        </div>

                        {/* Similarity Score */}
                        <div className="flex flex-col items-end gap-1">
                          <span className={`text-sm font-medium ${similarity >= 80 ? 'text-green-400' :
                              similarity >= 60 ? 'text-yellow-400' :
                                'text-slate-400'
                            }`}>
                            {similarity}%
                          </span>
                          <div className="w-16 h-1.5 bg-slate-700/50 rounded-full overflow-hidden">
                            <div
                              className={`h-full rounded-full transition-all ${similarity >= 80 ? 'bg-green-500' :
                                  similarity >= 60 ? 'bg-yellow-500' :
                                    'bg-slate-500'
                                }`}
                              style={{ width: `${similarity}%` }}
                            />
                          </div>
                        </div>
                      </div>

                      {/* Content Snippet */}
                      {snippet && (
                        <div
                          className="text-sm text-slate-300 leading-relaxed"
                          dangerouslySetInnerHTML={{
                            __html: highlightText(snippet, query)
                          }}
                        />
                      )}
                    </div>
                  );
                })
              ) : (
                <div className="text-center py-12 bg-slate-800/50 rounded-xl border border-white/10">
                  <SearchIcon size={48} className="mx-auto text-slate-600 mb-4" />
                  <p className="text-slate-400 text-lg">Keine Ergebnisse gefunden</p>
                  <p className="text-slate-500 text-sm mt-2">
                    Versuche es mit anderen Suchbegriffen oder formuliere eine Frage
                  </p>
                </div>
              )}
            </div>
          )}
        </div>
      )}

      {/* Initial State - Suggestions */}
      {!results && !loading && !error && (
        <div className="text-center py-12">
          <div className="inline-flex p-5 bg-slate-800/50 rounded-full mb-6 border border-white/5">
            <SearchIcon size={40} className="text-slate-500" />
          </div>
          <p className="text-slate-400 text-lg mb-2">
            Gib deinen Suchbegriff oder deine Frage ein
          </p>
          <p className="text-slate-500 text-sm mb-8">
            Die AI erkennt automatisch, ob du nach Dateien suchst oder eine Antwort möchtest
          </p>

          {/* Quick Suggestions */}
          <div className="flex flex-wrap justify-center gap-2 max-w-xl mx-auto">
            {[
              { label: "Alle Rechnungen 2024", icon: SearchIcon },
              { label: "Was kosten die Server?", icon: Brain },
              { label: "Vertrag Müller", icon: SearchIcon },
              { label: "Wie hoch war der Umsatz?", icon: Brain }
            ].map((suggestion, idx) => (
              <button
                key={idx}
                onClick={() => {
                  setQuery(suggestion.label);
                  performQuery(suggestion.label);
                }}
                className="inline-flex items-center gap-2 px-4 py-2.5 bg-slate-800/60 border border-white/10 rounded-xl text-sm text-slate-300 hover:border-indigo-500/50 hover:text-indigo-300 hover:bg-slate-800 transition-all group"
              >
                <suggestion.icon size={14} className="text-slate-500 group-hover:text-indigo-400 transition-colors" />
                {suggestion.label}
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
