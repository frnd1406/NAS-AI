import { useState, useEffect, useCallback, createContext, useContext } from 'react';
import { X, CheckCircle, AlertCircle, Info, AlertTriangle } from 'lucide-react';

// Toast Context
const ToastContext = createContext(null);

export function useToast() {
    const context = useContext(ToastContext);
    if (!context) {
        throw new Error('useToast must be used within ToastProvider');
    }
    return context;
}

// Toast Types Config
const TOAST_TYPES = {
    success: {
        icon: CheckCircle,
        bg: 'bg-emerald-500/10',
        border: 'border-emerald-500/30',
        text: 'text-emerald-400',
        iconColor: 'text-emerald-400'
    },
    error: {
        icon: AlertCircle,
        bg: 'bg-red-500/10',
        border: 'border-red-500/30',
        text: 'text-red-400',
        iconColor: 'text-red-400'
    },
    warning: {
        icon: AlertTriangle,
        bg: 'bg-amber-500/10',
        border: 'border-amber-500/30',
        text: 'text-amber-400',
        iconColor: 'text-amber-400'
    },
    info: {
        icon: Info,
        bg: 'bg-blue-500/10',
        border: 'border-blue-500/30',
        text: 'text-blue-400',
        iconColor: 'text-blue-400'
    }
};

// Toast Component
function Toast({ id, message, type = 'info', onClose }) {
    const config = TOAST_TYPES[type];
    const Icon = config.icon;

    useEffect(() => {
        const timer = setTimeout(() => onClose(id), 4000);
        return () => clearTimeout(timer);
    }, [id, onClose]);

    return (
        <div
            className={`flex items-center gap-3 px-4 py-3 rounded-xl border backdrop-blur-xl shadow-2xl
        ${config.bg} ${config.border} animate-in slide-in-from-right-full fade-in duration-300`}
        >
            <Icon size={20} className={config.iconColor} />
            <p className={`text-sm font-medium ${config.text}`}>{message}</p>
            <button
                onClick={() => onClose(id)}
                className="ml-2 p-1 rounded-lg hover:bg-white/10 transition-colors"
            >
                <X size={14} className="text-slate-400" />
            </button>
        </div>
    );
}

// Toast Container Component
function ToastContainer({ toasts, removeToast }) {
    return (
        <div className="fixed bottom-6 right-6 z-[100] flex flex-col gap-2 pointer-events-auto">
            {toasts.map((toast) => (
                <Toast
                    key={toast.id}
                    id={toast.id}
                    message={toast.message}
                    type={toast.type}
                    onClose={removeToast}
                />
            ))}
        </div>
    );
}

// Toast Provider
export function ToastProvider({ children }) {
    const [toasts, setToasts] = useState([]);

    const addToast = useCallback((message, type = 'info') => {
        const id = Date.now() + Math.random();
        setToasts((prev) => [...prev, { id, message, type }]);
        return id;
    }, []);

    const removeToast = useCallback((id) => {
        setToasts((prev) => prev.filter((t) => t.id !== id));
    }, []);

    const toast = {
        success: (msg) => addToast(msg, 'success'),
        error: (msg) => addToast(msg, 'error'),
        warning: (msg) => addToast(msg, 'warning'),
        info: (msg) => addToast(msg, 'info'),
    };

    return (
        <ToastContext.Provider value={toast}>
            {children}
            <ToastContainer toasts={toasts} removeToast={removeToast} />
        </ToastContext.Provider>
    );
}

export default ToastProvider;
