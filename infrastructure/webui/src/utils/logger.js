/**
 * FIX [BUG-JS-010]: Production-safe logging utility
 * Logs to console in development, sends critical errors to backend
 */

const isDevelopment = import.meta.env.MODE === 'development';

const logToServer = (level, ...args) => {
  try {
    const message = args.map(arg => {
      if (arg instanceof Error) return arg.stack || arg.toString();
      if (typeof arg === 'object') {
        try {
          return JSON.stringify(arg);
        } catch (e) {
          return '[Complex Object]';
        }
      }
      return String(arg);
    }).join(' ');

    const payload = {
      level,
      message,
      url: window.location.href,
      time: new Date().toISOString(),
      context: {
        userAgent: navigator.userAgent
      }
    };

    // Fire and forget
    fetch('/api/v1/system/logs/frontend', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(payload),
      keepalive: true // Ensure request survives page navigation
    }).catch(() => {
      // Prevent infinite loops if logging fails
    });
  } catch (e) {
    // Absolute safety net
  }
};

export const logger = {
  error: (...args) => {
    if (isDevelopment) {
      console.error('[ERROR]', ...args);
    }
    // Always send errors to backend for analysis
    logToServer('error', ...args);
  },

  warn: (...args) => {
    if (isDevelopment) {
      console.warn('[WARN]', ...args);
    }
  },

  info: (...args) => {
    if (isDevelopment) {
      console.info('[INFO]', ...args);
    }
  },

  debug: (...args) => {
    if (isDevelopment) {
      console.debug('[DEBUG]', ...args);
    }
  },
};

export default logger;
