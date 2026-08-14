import PropTypes from 'prop-types';
import {
  FileText,
  FileImage,
  FileJson,
  FileCode,
  File,
  ExternalLink,
  CheckCircle2,
  TrendingUp
} from 'lucide-react';

/**
 * FileSelector - Interactive file selection component
 * Shows list of files with similarity scores and allows user to choose which to preview
 */
export function FileSelector({ files, onSelectFile, autoSelectSingle = true }) {
  // Auto-select if only one file
  if (files.length === 1 && autoSelectSingle) {
    setTimeout(() => onSelectFile(0), 100);
    return (
      <div className="p-4 bg-blue-500/10 border border-blue-500/30 rounded-xl">
        <div className="flex items-center gap-2 text-blue-400">
          <CheckCircle2 size={20} className="animate-pulse" />
          <span className="text-sm font-medium">Öffne Datei automatisch...</span>
        </div>
      </div>
    );
  }

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

  const getRelevanceColor = (similarity) => {
    const score = similarity * 100;
    if (score >= 80) return 'emerald';
    if (score >= 60) return 'blue';
    if (score >= 40) return 'amber';
    return 'slate';
  };

  const getRelevanceLabel = (similarity) => {
    const score = similarity * 100;
    if (score >= 80) return 'Sehr relevant';
    if (score >= 60) return 'Relevant';
    if (score >= 40) return 'Möglicherweise relevant';
    return 'Niedrige Relevanz';
  };

  return (
    <div className="w-full">
      {/* Split Layout Container */}
      <div className="flex gap-3 rounded-2xl overflow-hidden bg-gradient-to-br from-slate-800/90 via-slate-800/80 to-blue-900/40 border border-white/10 shadow-xl">

        {/* Left Side - AI Response Area */}
        <div className="flex-1 p-5 bg-gradient-to-b from-blue-600/20 to-cyan-600/10 backdrop-blur-sm">
          <div className="flex items-center gap-2 mb-3">
            <div className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" />
            <span className="text-sm font-semibold text-white/90">NAS.AI</span>
          </div>
          <div className="text-sm text-slate-300 leading-relaxed">
            Ich habe <span className="text-cyan-400 font-semibold">{files.length} Dokumente</span> gefunden.
            Wähle rechts ein Dokument um Details zu sehen.
          </div>
        </div>

        {/* Right Side - Stacked File Buttons */}
        <div className="flex flex-col gap-2 p-3 min-w-[180px]">
          {files.map((file, index) => {
            const FileIcon = getFileIcon(file.file_path);
            const fileName = file.file_path.split('/').pop();
            const similarity = file.similarity || 0;
            const similarityPercent = Math.round(similarity * 100);
            const color = getRelevanceColor(similarity);

            return (
              <button
                key={index}
                onClick={() => onSelectFile(index)}
                className="group flex items-center gap-2 px-4 py-3 bg-slate-700/60 hover:bg-blue-600/40 border border-white/10 hover:border-blue-400/50 rounded-xl transition-all duration-200 hover:scale-[1.02] hover:shadow-lg hover:shadow-blue-500/20 text-left backdrop-blur-sm animate-slide-in-right"
                style={{
                  animationDelay: `${index * 150}ms`,
                  animationFillMode: 'backwards'
                }}
              >
                {/* File Icon */}
                <FileIcon size={18} className={`text-${color}-400 group-hover:text-white transition-colors`} />

                {/* File Name */}
                <div className="flex-1 min-w-0">
                  <p className="text-sm text-white font-medium truncate group-hover:text-white">
                    {fileName.length > 20 ? fileName.substring(0, 18) + '...' : fileName}
                  </p>
                </div>

                {/* Similarity Badge */}
                <span className={`text-xs px-2 py-0.5 rounded-full bg-${color}-500/30 text-${color}-300 font-semibold`}>
                  {similarityPercent}%
                </span>
              </button>
            );
          })}
        </div>
      </div>

      {/* Helper Text */}
      <div className="text-xs text-slate-500 text-center mt-2">
        Klicke auf ein Dokument um es im Detail zu sehen
      </div>
    </div>
  );
}

FileSelector.propTypes = {
  files: PropTypes.arrayOf(PropTypes.shape({
    file_path: PropTypes.string.isRequired,
    file_id: PropTypes.string,
    similarity: PropTypes.number,
    content: PropTypes.string,
  })).isRequired,
  onSelectFile: PropTypes.func.isRequired,
  autoSelectSingle: PropTypes.bool,
};
