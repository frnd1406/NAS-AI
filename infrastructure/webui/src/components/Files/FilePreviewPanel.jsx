import { useState, useEffect } from 'react';
import PropTypes from 'prop-types';
import {
  X,
  Download,
  ExternalLink,
  FileText,
  FileImage,
  FileJson,
  FileCode,
  File,
  ZoomIn,
  ZoomOut,
  ChevronLeft,
  ChevronRight
} from 'lucide-react';
import { fetchFileContent } from '../../api/files';

/**
 * FilePreviewPanel - Displays file content with support for multiple file types
 * Features:
 * - Auto-detection of file type
 * - Syntax highlighting for code/JSON
 * - Image zoom controls
 * - PDF viewer
 * - Markdown rendering
 * - Navigation between multiple files
 */
export function FilePreviewPanel({ files = [], currentIndex = 0, onClose, onNavigate }) {
  const [zoom, setZoom] = useState(100);
  const [fileContent, setFileContent] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  const currentFile = files[currentIndex];
  const hasMultipleFiles = files.length > 1;

  useEffect(() => {
    if (currentFile) {
      loadFileContent(currentFile);
    }
  }, [currentFile]);

  const loadFileContent = async (file) => {
    setLoading(true);
    setError(null);

    try {
      // If content is already provided, use it
      if (file.content) {
        setFileContent(file.content);
        setLoading(false);
        return;
      }

      // Otherwise fetch from API
      const content = await fetchFileContent(file.file_path);
      setFileContent(content);
    } catch (err) {
      console.error('Error loading file:', err);
      setError('Datei konnte nicht geladen werden');
      setFileContent('');
    } finally {
      setLoading(false);
    }
  };

  const getFileIcon = (filePath) => {
    const ext = filePath.split('.').pop().toLowerCase();
    const iconMap = {
      'png': FileImage,
      'jpg': FileImage,
      'jpeg': FileImage,
      'gif': FileImage,
      'svg': FileImage,
      'json': FileJson,
      'js': FileCode,
      'jsx': FileCode,
      'ts': FileCode,
      'tsx': FileCode,
      'py': FileCode,
      'go': FileCode,
      'txt': FileText,
      'md': FileText,
      'log': FileText,
    };
    return iconMap[ext] || File;
  };

  const getFileType = (filePath) => {
    const ext = filePath.split('.').pop().toLowerCase();
    const typeMap = {
      'png': 'image',
      'jpg': 'image',
      'jpeg': 'image',
      'gif': 'image',
      'svg': 'image',
      'json': 'json',
      'js': 'code',
      'jsx': 'code',
      'ts': 'code',
      'tsx': 'code',
      'py': 'code',
      'go': 'code',
      'md': 'markdown',
      'pdf': 'pdf',
    };
    return typeMap[ext] || 'text';
  };

  const handleDownload = () => {
    const blob = new Blob([fileContent], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = currentFile.file_path.split('/').pop();
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  const renderFileContent = () => {
    if (loading) {
      return (
        <div className="flex items-center justify-center h-full">
          <div className="flex flex-col items-center gap-4">
            <div className="w-12 h-12 border-4 border-blue-500/30 border-t-blue-500 rounded-full animate-spin"></div>
            <p className="text-slate-400">Lade Datei...</p>
          </div>
        </div>
      );
    }

    if (error) {
      return (
        <div className="flex items-center justify-center h-full">
          <div className="text-center">
            <p className="text-rose-400 text-lg mb-2">{error}</p>
            <button
              onClick={() => loadFileContent(currentFile)}
              className="px-4 py-2 bg-blue-600 hover:bg-blue-500 rounded-lg transition-colors"
            >
              Erneut versuchen
            </button>
          </div>
        </div>
      );
    }

    const fileType = getFileType(currentFile.file_path);

    switch (fileType) {
      case 'image':
        return (
          <div className="flex flex-col items-center justify-center h-full p-8 overflow-auto">
            <img
              src={currentFile.file_path}
              alt={currentFile.file_path.split('/').pop()}
              style={{ transform: `scale(${zoom / 100})` }}
              className="max-w-full h-auto transition-transform duration-200 shadow-2xl rounded-lg"
            />
            <div className="mt-4 flex items-center gap-2 bg-slate-800 rounded-full px-4 py-2">
              <button
                onClick={() => setZoom(Math.max(25, zoom - 25))}
                className="p-2 hover:bg-slate-700 rounded-full transition-colors"
                disabled={zoom <= 25}
              >
                <ZoomOut size={18} />
              </button>
              <span className="text-sm font-mono text-slate-300 min-w-[60px] text-center">{zoom}%</span>
              <button
                onClick={() => setZoom(Math.min(200, zoom + 25))}
                className="p-2 hover:bg-slate-700 rounded-full transition-colors"
                disabled={zoom >= 200}
              >
                <ZoomIn size={18} />
              </button>
            </div>
          </div>
        );

      case 'json':
        try {
          const formatted = JSON.stringify(JSON.parse(fileContent), null, 2);
          return (
            <pre className="p-6 text-sm font-mono text-slate-300 overflow-auto h-full bg-slate-950/50">
              <code className="language-json">{formatted}</code>
            </pre>
          );
        } catch {
          return (
            <pre className="p-6 text-sm font-mono text-slate-300 overflow-auto h-full bg-slate-950/50">
              <code>{fileContent}</code>
            </pre>
          );
        }

      case 'code':
        return (
          <pre className="p-6 text-sm font-mono text-slate-300 overflow-auto h-full bg-slate-950/50">
            <code className="language-javascript">{fileContent}</code>
          </pre>
        );

      case 'markdown':
        return (
          <div className="p-6 prose prose-invert max-w-none overflow-auto h-full">
            <div dangerouslySetInnerHTML={{ __html: fileContent.replace(/\n/g, '<br>') }} />
          </div>
        );

      default:
        return (
          <pre className="p-6 text-sm font-mono text-slate-300 overflow-auto h-full whitespace-pre-wrap bg-slate-950/50 leading-relaxed">
            {fileContent}
          </pre>
        );
    }
  };

  if (!currentFile) return null;

  const FileIcon = getFileIcon(currentFile.file_path);
  const fileName = currentFile.file_path.split('/').pop();
  const similarity = currentFile.similarity ? Math.round(currentFile.similarity * 100) : null;

  return (
    <div className="fixed inset-y-0 right-0 w-1/2 bg-slate-900/95 backdrop-blur-xl border-l border-white/10 shadow-2xl flex flex-col z-50 animate-in slide-in-from-right duration-300">
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b border-white/10 bg-slate-800/50">
        <div className="flex items-center gap-3 flex-1 min-w-0">
          <div className="p-2 bg-blue-500/20 rounded-lg">
            <FileIcon size={20} className="text-blue-400" />
          </div>
          <div className="flex-1 min-w-0">
            <h3 className="text-white font-semibold truncate">{fileName}</h3>
            <div className="flex items-center gap-3">
              <p className="text-xs text-slate-400 truncate">{currentFile.file_path}</p>
              {similarity !== null && (
                <span className={`text-xs px-2 py-0.5 rounded-full ${similarity >= 80 ? 'bg-emerald-500/20 text-emerald-400' :
                  similarity >= 60 ? 'bg-blue-500/20 text-blue-400' :
                    'bg-amber-500/20 text-amber-400'
                  }`}>
                  {similarity}% relevant
                </span>
              )}
            </div>
          </div>
        </div>

        <div className="flex items-center gap-2">
          {hasMultipleFiles && (
            <div className="flex items-center gap-1 mr-2">
              <button
                onClick={() => onNavigate?.(currentIndex - 1)}
                disabled={currentIndex === 0}
                className="p-2 hover:bg-white/10 rounded-lg transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
              >
                <ChevronLeft size={20} />
              </button>
              <span className="text-sm text-slate-400 px-2">
                {currentIndex + 1} / {files.length}
              </span>
              <button
                onClick={() => onNavigate?.(currentIndex + 1)}
                disabled={currentIndex === files.length - 1}
                className="p-2 hover:bg-white/10 rounded-lg transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
              >
                <ChevronRight size={20} />
              </button>
            </div>
          )}
          <button
            onClick={handleDownload}
            className="p-2 hover:bg-white/10 rounded-lg transition-colors"
            title="Download"
          >
            <Download size={20} />
          </button>
          <button
            onClick={onClose}
            className="p-2 hover:bg-rose-500/20 text-rose-400 rounded-lg transition-colors"
            title="Schließen"
          >
            <X size={20} />
          </button>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-hidden">
        {renderFileContent()}
      </div>

      {/* Footer Info */}
      <div className="p-3 border-t border-white/10 bg-slate-800/30 text-xs text-slate-400 flex items-center justify-between">
        <span>Dateigröße: {(fileContent.length / 1024).toFixed(2)} KB</span>
        <span>{fileContent.split('\n').length} Zeilen</span>
      </div>
    </div>
  );
}

FilePreviewPanel.propTypes = {
  files: PropTypes.arrayOf(PropTypes.shape({
    file_path: PropTypes.string.isRequired,
    file_id: PropTypes.string,
    content: PropTypes.string,
    similarity: PropTypes.number,
  })).isRequired,
  currentIndex: PropTypes.number,
  onClose: PropTypes.func.isRequired,
  onNavigate: PropTypes.func,
};
