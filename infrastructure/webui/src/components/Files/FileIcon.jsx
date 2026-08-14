// File icon component - returns appropriate icon based on file type

import {
    FolderOpen,
    File,
    Image as ImageIcon,
    FileText,
    FileArchive,
    FileCode,
    FileVideo,
    FileAudio,
} from 'lucide-react';
import { getFileType } from '../../utils/fileUtils';

const ICON_MAP = {
    folder: FolderOpen,
    image: ImageIcon,
    text: FileText,
    code: FileCode,
    archive: FileArchive,
    video: FileVideo,
    audio: FileAudio,
    file: File,
};

export function getFileIcon(name, isDir) {
    const type = getFileType(name, isDir);
    return ICON_MAP[type] || File;
}

export function FileIcon({ name, isDir, size = 16, className = '' }) {
    const Icon = getFileIcon(name, isDir);
    return <Icon size={size} className={className} />;
}
