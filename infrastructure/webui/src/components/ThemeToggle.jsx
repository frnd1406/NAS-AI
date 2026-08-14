import { useState, useEffect, createContext, useContext } from 'react';
import { Sun, Moon } from 'lucide-react';

// Theme Context
const ThemeContext = createContext(null);

export function useTheme() {
    const context = useContext(ThemeContext);
    if (!context) {
        throw new Error('useTheme must be used within ThemeProvider');
    }
    return context;
}

export function ThemeProvider({ children }) {
    const [theme, setTheme] = useState(() => {
        // Check localStorage first, then system preference
        const stored = localStorage.getItem('nas-theme');
        if (stored) return stored;
        return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
    });

    useEffect(() => {
        const root = document.documentElement;
        if (theme === 'dark') {
            root.classList.add('dark');
        } else {
            root.classList.remove('dark');
        }
        localStorage.setItem('nas-theme', theme);
    }, [theme]);

    const toggleTheme = () => setTheme(t => t === 'dark' ? 'light' : 'dark');

    return (
        <ThemeContext.Provider value={{ theme, toggleTheme }}>
            {children}
        </ThemeContext.Provider>
    );
}

// Theme Toggle Button Component
export function ThemeToggle({ className = '' }) {
    const { theme, toggleTheme } = useTheme();
    const isDark = theme === 'dark';

    return (
        <button
            onClick={toggleTheme}
            className={`relative p-2.5 rounded-full bg-white/5 border border-white/10 text-slate-300 
        hover:bg-white/10 transition-all duration-300 group ${className}`}
            title={isDark ? 'Switch to Light Mode' : 'Switch to Dark Mode'}
        >
            <div className="relative w-5 h-5">
                <Sun
                    size={20}
                    className={`absolute inset-0 transition-all duration-300 ${isDark ? 'opacity-0 rotate-90 scale-0' : 'opacity-100 rotate-0 scale-100 text-amber-400'
                        }`}
                />
                <Moon
                    size={20}
                    className={`absolute inset-0 transition-all duration-300 ${isDark ? 'opacity-100 rotate-0 scale-100 text-blue-400' : 'opacity-0 -rotate-90 scale-0'
                        }`}
                />
            </div>
        </button>
    );
}

export default ThemeProvider;
