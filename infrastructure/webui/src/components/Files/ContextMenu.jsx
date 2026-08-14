// Context Menu component for Files
// Portal-based dropdown that appears at mouse position on right-click

import { useEffect, useRef } from 'react';
import { createPortal } from 'react-dom';
import { Eye, Edit3, Download, Trash2, Archive, FolderOpen } from 'lucide-react';

export function ContextMenu({
    isOpen,
    position,
    item,
    onClose,
    onOpen,
    onPreview,
    onRename,
    onDownload,
    onDelete,
}) {
    const menuRef = useRef(null);

    // Close on click outside or ESC
    useEffect(() => {
        if (!isOpen) return;

        const handleClickOutside = (e) => {
            if (menuRef.current && !menuRef.current.contains(e.target)) {
                onClose();
            }
        };

        const handleEscape = (e) => {
            if (e.key === 'Escape') {
                onClose();
            }
        };

        document.addEventListener('mousedown', handleClickOutside);
        document.addEventListener('keydown', handleEscape);

        return () => {
            document.removeEventListener('mousedown', handleClickOutside);
            document.removeEventListener('keydown', handleEscape);
        };
    }, [isOpen, onClose]);

    // Adjust position to stay within viewport
    useEffect(() => {
        if (!isOpen || !menuRef.current) return;

        const menu = menuRef.current;
        const rect = menu.getBoundingClientRect();
        const viewportWidth = window.innerWidth;
        const viewportHeight = window.innerHeight;

        let { x, y } = position;

        // Adjust horizontal position
        if (x + rect.width > viewportWidth - 10) {
            x = viewportWidth - rect.width - 10;
        }

        // Adjust vertical position
        if (y + rect.height > viewportHeight - 10) {
            y = viewportHeight - rect.height - 10;
        }

        menu.style.left = `${x}px`;
        menu.style.top = `${y}px`;
    }, [isOpen, position]);

    if (!isOpen || !item) return null;

    const canPreview = !item.isDir && ['txt', 'md', 'json', 'pdf', 'jpg', 'jpeg', 'png', 'gif', 'webp'].some(
        ext => item.name.toLowerCase().endsWith(`.${ext}`)
    );

    const menuItems = [
        // Open / Preview
        item.isDir
            ? { icon: FolderOpen, label: 'Öffnen', action: () => { onOpen?.(item); onClose(); }, color: 'blue' }
            : canPreview
                ? { icon: Eye, label: 'Vorschau', action: () => { onPreview?.(item); onClose(); }, color: 'violet' }
                : null,
        // Rename
        { icon: Edit3, label: 'Umbenennen', action: () => { onRename?.(item); onClose(); }, color: 'blue' },
        // Download
        {
            icon: item.isDir ? Archive : Download,
            label: item.isDir ? 'Als ZIP herunterladen' : 'Herunterladen',
            action: () => { onDownload?.(item); onClose(); },
            color: 'emerald'
        },
        // Divider
        { divider: true },
        // Delete
        { icon: Trash2, label: 'Löschen', action: () => { onDelete?.(item); onClose(); }, color: 'rose', danger: true },
    ].filter(Boolean);

    const menuContent = (
        <div
            ref={menuRef}
            className="fixed z-[9999] min-w-[180px] py-2 rounded-xl border border-white/10 bg-slate-900/95 backdrop-blur-xl shadow-2xl"
            style={{ left: position.x, top: position.y }}
        >
            {/* Header with item name */}
            <div className="px-3 py-2 border-b border-white/10 mb-1">
                <p className="text-xs text-slate-400 truncate max-w-[200px]">{item.name}</p>
            </div>

            {/* Menu Items */}
            {menuItems.map((menuItem, idx) => {
                if (menuItem.divider) {
                    return <div key={idx} className="my-1 border-t border-white/10" />;
                }

                const Icon = menuItem.icon;
                const colorClasses = {
                    blue: 'text-blue-400 hover:bg-blue-500/10',
                    violet: 'text-violet-400 hover:bg-violet-500/10',
                    emerald: 'text-emerald-400 hover:bg-emerald-500/10',
                    rose: 'text-rose-400 hover:bg-rose-500/10',
                };

                return (
                    <button
                        key={idx}
                        onClick={menuItem.action}
                        className={`w-full flex items-center gap-3 px-3 py-2 text-sm transition-all ${colorClasses[menuItem.color] || 'text-slate-300 hover:bg-white/5'}`}
                    >
                        <Icon size={16} />
                        <span>{menuItem.label}</span>
                    </button>
                );
            })}
        </div>
    );

    // Render as portal to body
    return createPortal(menuContent, document.body);
}
