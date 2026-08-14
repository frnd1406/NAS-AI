// InlineRenameInput - Shared component for inline file/folder renaming

import React, { useState, useEffect, useRef } from 'react';
import { Check, X } from 'lucide-react';

export function InlineRenameInput({
    initialName,
    onSubmit,
    onCancel,
    inputClassName = '',
    containerClassName = ''
}) {
    const [name, setName] = useState(initialName || '');
    const inputRef = useRef(null);

    useEffect(() => {
        // Focus and select text on mount
        if (inputRef.current) {
            inputRef.current.focus();
            inputRef.current.select();
        }
    }, []);

    const handleSubmit = async () => {
        if (name && name !== initialName) {
            await onSubmit(name);
        } else {
            onCancel();
        }
    };

    const handleKeyDown = (e) => {
        if (e.key === 'Enter') {
            e.preventDefault();
            handleSubmit();
        }
        if (e.key === 'Escape') {
            e.preventDefault();
            onCancel();
        }
    };

    return (
        <div
            className={`flex items-center gap-2 ${containerClassName}`}
            onClick={(e) => e.stopPropagation()}
        >
            <input
                ref={inputRef}
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                onKeyDown={handleKeyDown}
                onBlur={onCancel}
                className={`flex-1 px-3 py-1.5 bg-slate-800 border border-white/10 rounded-lg text-white focus:outline-none focus:border-blue-500 ${inputClassName}`}
            />
            <button
                onMouseDown={(e) => e.preventDefault()} // Prevent blur before click
                onClick={handleSubmit}
                className="p-2 rounded-lg bg-emerald-500/20 text-emerald-400 hover:bg-emerald-500/30 transition-colors"
            >
                <Check size={14} />
            </button>
            <button
                onMouseDown={(e) => e.preventDefault()}
                onClick={onCancel}
                className="p-2 rounded-lg bg-rose-500/20 text-rose-400 hover:bg-rose-500/30 transition-colors"
            >
                <X size={14} />
            </button>
        </div>
    );
}
