
import { X, MessageSquare, ChevronRight, Sparkles, Loader2 } from 'lucide-react';
import { FileSelector } from '../Files/FileSelector';
import { FilePreviewPanel } from '../Files/FilePreviewPanel';
import { useChatWidget } from '../../hooks/useChatWidget';

/**
 * EnhancedChatWidget - Floating AI Chat with intelligent file preview
 * Features:
 * - Natural language Q&A
 * - Intelligent file selection (auto-open single file, ask for multiple)
 * - Live file preview with multi-format support
 * - Similarity-based relevance scoring
 */
export function EnhancedChatWidget() {
  const {
    isOpen,
    input,
    isLoading,
    messages,
    previewFiles,
    currentPreviewIndex,
    showPreview,
    toggleOpen,
    setInput,
    sendMessage,
    handleFileSelect,
    handlePreviewNavigate,
    closePreview
  } = useChatWidget();

  return (
    <>
      {/* Floating Action Button */}
      <button
        onClick={toggleOpen}
        className="fixed bottom-6 right-6 p-4 bg-gradient-to-br from-blue-600 to-violet-600 text-white rounded-full shadow-[0_0_30px_rgba(79,70,229,0.3)] hover:shadow-[0_0_40px_rgba(79,70,229,0.5)] hover:scale-105 transition-all duration-300 z-50"
      >
        {isOpen ? <X size={24} /> : <MessageSquare size={24} />}
        {!isOpen && (
          <span className="absolute -top-1 -right-1 w-3 h-3 bg-emerald-500 rounded-full border-2 border-[#0a0a0c] animate-pulse" />
        )}
      </button>

      {/* Chat Panel */}
      {isOpen && (
        <div className="fixed bottom-24 right-6 w-[450px] h-[600px] bg-slate-900/95 backdrop-blur-xl border border-white/10 rounded-2xl shadow-2xl flex flex-col z-40 animate-in slide-in-from-bottom-10 fade-in duration-300">
          {/* Header */}
          <div className="p-4 border-b border-white/10 flex items-center bg-gradient-to-r from-slate-800/50 to-slate-800/30 rounded-t-2xl">
            <div className={`w - 2 h - 2 rounded - full mr - 2 ${isLoading ? 'bg-amber-500 animate-pulse' : 'bg-emerald-500 animate-pulse'} `} />
            <span className="font-semibold text-white">NAS.AI Assistant</span>
            <Sparkles size={14} className="ml-2 text-violet-400" />
            <span className={`ml - auto text - xs px - 2 py - 1 rounded ${isLoading ? 'text-amber-400 bg-amber-900/30' : 'text-emerald-400 bg-emerald-900/30'} `}>
              {isLoading ? 'Denke nach...' : 'Online'}
            </span>
          </div>

          {/* Messages */}
          <div className="flex-1 overflow-y-auto p-4 space-y-4">
            {messages.map((msg) => (
              <div key={msg.id}>
                {msg.type === 'text' && (
                  <div className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'} `}>
                    <div className={`max - w - [85 %] rounded - 2xl px - 4 py - 3 text - sm ${msg.role === 'user'
                      ? 'bg-blue-600 text-white rounded-br-none'
                      : 'bg-slate-800 border border-white/5 text-slate-200 rounded-bl-none'
                      } `}>
                      <div className="whitespace-pre-wrap">{msg.text}</div>
                    </div>
                  </div>
                )}

                {msg.type === 'file_selector' && msg.files && (
                  <div className="bg-slate-800/30 rounded-xl p-3 border border-white/5">
                    <FileSelector
                      files={msg.files}
                      onSelectFile={(index) => handleFileSelect(msg.files, index)}
                      autoSelectSingle={true}
                    />
                  </div>
                )}
              </div>
            ))}

            {/* Loading indicator */}
            {isLoading && (
              <div className="flex justify-start">
                <div className="bg-slate-800 border border-white/5 text-slate-200 rounded-2xl rounded-bl-none px-4 py-3 text-sm flex items-center gap-3">
                  <Loader2 size={16} className="animate-spin text-blue-400" />
                  <span className="text-slate-400">Analysiere Dokumente...</span>
                </div>
              </div>
            )}
          </div>

          {/* Input */}
          <form onSubmit={sendMessage} className="p-4 border-t border-white/10 bg-slate-800/30 rounded-b-2xl">
            <div className="relative">
              <input
                type="text"
                className="w-full bg-slate-800 text-white pl-4 pr-12 py-3 rounded-xl border border-white/10 focus:outline-none focus:border-blue-500/50 transition-colors placeholder-slate-500 text-sm"
                placeholder="Frag etwas über deine Dokumente..."
                value={input}
                onChange={(e) => setInput(e.target.value)}
                disabled={isLoading}
              />
              <button
                type="submit"
                disabled={isLoading || !input.trim()}
                className="absolute right-2 top-2 p-1.5 bg-blue-600 rounded-lg text-white hover:bg-blue-500 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <ChevronRight size={18} />
              </button>
            </div>
            <div className="mt-2 text-xs text-slate-500 text-center">
              Powered by Ollama RAG • {messages.filter(m => m.role === 'user').length} Anfragen
            </div>
          </form>
        </div>
      )}

      {/* File Preview Panel */}
      {showPreview && previewFiles.length > 0 && (
        <FilePreviewPanel
          files={previewFiles}
          currentIndex={currentPreviewIndex}
          onClose={closePreview}
          onNavigate={handlePreviewNavigate}
        />
      )}
    </>
  );
}

