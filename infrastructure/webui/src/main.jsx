import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClientProvider } from "@tanstack/react-query";
import App from "./App";
import "./index.css";
import { queryClient } from "./lib/queryClient";
import { ToastProvider } from "./components/ui/Toast";
import { ThemeProvider } from "./components/ThemeToggle";

/**
 * React.StrictMode is enabled for development mode.
 *
 * Benefits:
 * - Highlights potential problems in the application
 * - Detects unsafe lifecycle methods
 * - Warns about legacy string ref API usage
 * - Detects unexpected side effects
 *
 * Important Notes:
 * - StrictMode intentionally double-invokes useEffect in development
 *   to catch bugs with missing cleanup functions
 * - All components now use AbortController to handle cleanup properly
 * - StrictMode does NOT run in production builds
 *
 * Phase 3 Hardening:
 * ✅ Global ErrorBoundary implemented (react-error-boundary)
 * ✅ Dashboard.jsx: AbortController cleanup (BUG-JS-006 fixed)
 * ✅ All fetch() calls are now properly abortable
 * ✅ Toast notifications for user feedback
 * ✅ Theme toggle (dark/light mode)
 */
ReactDOM.createRoot(document.getElementById("root")).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <ToastProvider>
          <App />
        </ToastProvider>
      </ThemeProvider>
    </QueryClientProvider>
  </React.StrictMode>
);
