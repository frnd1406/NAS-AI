const envBaseUrl = (import.meta.env.VITE_API_BASE_URL || "").trim();

function deriveDefaultBaseUrl() {
  if (typeof window === "undefined") {
    return "http://localhost:8080";
  }
  const { protocol, hostname } = window.location;

  // In production (via nginx), use relative URLs so requests go through the proxy
  // Only use explicit port for localhost development
  if (hostname === "localhost" || hostname === "127.0.0.1") {
    const defaultPort = protocol === "https:" ? "8443" : "8080";
    return `${protocol}//${hostname}:${defaultPort}`;
  }

  // Production: use empty base URL (relative paths work via nginx proxy)
  return "";
}

function normalizeBaseUrl(url) {
  const base = url || deriveDefaultBaseUrl();
  return base.replace(/\/+$/, "");
}

const API_BASE_URL = normalizeBaseUrl(envBaseUrl);
const LOGOUT_COUNTDOWN_SECONDS = 4;

// Local Fallback Storage Keys
const LOCAL_IP_KEY = 'nas_last_known_ip';
const LAST_SEEN_KEY = 'nas_last_seen';

/**
 * Extract and store local IPs from API responses for offline fallback.
 * @param {object} data - API response data
 */
function cacheLocalIPFromResponse(data) {
  if (typeof localStorage === 'undefined') return;
  if (data?.local_ips && Array.isArray(data.local_ips) && data.local_ips.length > 0) {
    localStorage.setItem(LOCAL_IP_KEY, data.local_ips[0]);
    localStorage.setItem(LAST_SEEN_KEY, Date.now().toString());
  }
}

/**
 * Get the cached local IP for fallback.
 * @returns {string|null}
 */
export function getCachedLocalIP() {
  if (typeof localStorage === 'undefined') return null;
  return localStorage.getItem(LOCAL_IP_KEY);
}

/**
 * Get the timestamp when local IP was last seen.
 * @returns {number|null}
 */
export function getLastSeenTimestamp() {
  if (typeof localStorage === 'undefined') return null;
  const ts = localStorage.getItem(LAST_SEEN_KEY);
  return ts ? parseInt(ts, 10) : null;
}

function buildUrl(path = "") {
  if (!path.startsWith("/")) {
    return `${API_BASE_URL}/${path}`;
  }
  return `${API_BASE_URL}${path}`;
}

export function getApiBaseUrl() {
  return API_BASE_URL;
}

let logoutOverlay;
let logoutCountdownInterval;
let logoutRedirectScheduled = false;

function ensureLogoutStyles() {
  if (typeof document === "undefined") return;
  if (document.getElementById("session-warning-styles")) return;

  const style = document.createElement("style");
  style.id = "session-warning-styles";
  style.textContent = `
    .session-warning-overlay {
      position: fixed;
      inset: 0;
      display: flex;
      align-items: center;
      justify-content: center;
      background: rgba(0,0,0,0.35);
      backdrop-filter: blur(12px);
      z-index: 9999;
      opacity: 0;
      pointer-events: none;
      transition: opacity 180ms ease;
    }
    .session-warning-overlay.is-visible {
      opacity: 1;
      pointer-events: auto;
    }
    .session-warning-card {
      max-width: 420px;
      width: 90%;
      padding: 24px 26px;
      border-radius: 18px;
      background: linear-gradient(135deg, rgba(239,68,68,0.25), rgba(127,29,29,0.35));
      border: 1px solid rgba(248,113,113,0.55);
      box-shadow: 0 20px 70px rgba(239,68,68,0.35);
      color: #fff;
      backdrop-filter: blur(18px);
      font-family: 'Inter', system-ui, -apple-system, sans-serif;
    }
    .session-warning-pill {
      display: inline-flex;
      align-items: center;
      gap: 8px;
      padding: 6px 12px;
      border-radius: 999px;
      background: rgba(248,113,113,0.28);
      border: 1px solid rgba(248,113,113,0.55);
      text-transform: uppercase;
      letter-spacing: 0.08em;
      font-size: 12px;
      font-weight: 700;
    }
    .session-warning-title {
      margin: 14px 0 6px 0;
      font-size: 22px;
      font-weight: 800;
      letter-spacing: 0.01em;
    }
    .session-warning-body {
      margin: 0 0 12px 0;
      color: rgba(255,255,255,0.9);
      line-height: 1.4;
      font-size: 15px;
    }
    .session-warning-timer {
      font-weight: 800;
      font-size: 17px;
      color: #fecdd3;
    }
    .session-warning-subtle {
      margin: 0;
      color: rgba(255,255,255,0.78);
      font-size: 13px;
    }
  `;

  document.head.appendChild(style);
}

function showLogoutOverlay(seconds) {
  if (typeof document === "undefined") return;
  ensureLogoutStyles();

  let overlay = document.getElementById("session-warning-overlay");

  if (!overlay) {
    overlay = document.createElement("div");
    overlay.id = "session-warning-overlay";
    overlay.className = "session-warning-overlay";
    overlay.innerHTML = `
      <div class="session-warning-card">
        <div class="session-warning-pill">Warnung · Session läuft ab</div>
        <div class="session-warning-title">Gleich wirst du abgemeldet</div>
        <p class="session-warning-body">
          Wir konnten deinen Token nicht erneuern. Du wirst in
          <span class="session-warning-timer" data-session-timer></span>
          abgemeldet.
        </p>
        <p class="session-warning-subtle">Bitte melde dich erneut an, um weiterzuarbeiten.</p>
      </div>
    `;
    document.body.appendChild(overlay);
  }

  // Make sure to show it
  requestAnimationFrame(() => {
    overlay.classList.add("is-visible");
  });


  const timerEl = overlay.querySelector("[data-session-timer]");
  if (!timerEl) return;

  let remaining = seconds;
  timerEl.textContent = `${remaining}s`;

  if (logoutCountdownInterval) {
    clearInterval(logoutCountdownInterval);
  }

  logoutCountdownInterval = setInterval(() => {
    remaining -= 1;
    if (remaining <= 0) {
      clearInterval(logoutCountdownInterval);
      return;
    }
    timerEl.textContent = `${remaining}s`;
  }, 1000);
}

/**
 * Clear auth data on logout
 * Note: HttpOnly cookies will be cleared by the server on logout
 * We only clear the CSRF token from localStorage
 */
function clearAuth() {
  localStorage.removeItem("csrfToken");
  localStorage.removeItem("csrf_token");
  // Legacy cleanup
  localStorage.removeItem("accessToken");
  localStorage.removeItem("refreshToken");
  localStorage.removeItem("access_token");
  localStorage.removeItem("refresh_token");
}

/**
 * Refresh access token using HttpOnly cookie
 * The refresh token is automatically sent via cookie
 * Server will set new access_token cookie on success
 */
async function refreshAccessToken() {
  try {
    const res = await fetch(buildUrl("/auth/refresh"), {
      method: "POST",
      credentials: "include", // Send cookies
      headers: { "Content-Type": "application/json" },
    });

    if (!res.ok) {
      return false;
    }

    const data = await res.json().catch(() => null);

    // Update CSRF token if provided
    if (data?.csrf_token) {
      localStorage.setItem("csrfToken", data.csrf_token);
    }

    return true; // Success - new access token is in cookie
  } catch (err) {
    return false;
  }
}

function redirectToLogin() {
  if (typeof window === "undefined") {
    clearAuth();
    return;
  }

  if (logoutRedirectScheduled) return;
  logoutRedirectScheduled = true;

  clearAuth();
  showLogoutOverlay(LOGOUT_COUNTDOWN_SECONDS);

  setTimeout(() => {
    window.location.href = "/login";
  }, LOGOUT_COUNTDOWN_SECONDS * 1000);
}

/**
 * Build headers for API requests
 * Note: Access token is now sent automatically via HttpOnly cookie
 * We only need to include CSRF token for state-changing requests
 */
function buildHeaders(headersOverride = {}) {
  const csrfToken = localStorage.getItem("csrfToken") || localStorage.getItem("csrf_token") || "";
  const headers = {
    "Content-Type": "application/json",
    ...headersOverride,
  };

  // CSRF token still needs to be sent as header
  if (csrfToken && !headers["X-CSRF-Token"]) {
    headers["X-CSRF-Token"] = csrfToken;
  }

  return headers;
}

function extractErrorMessage(res, data) {
  return data?.error?.message || data?.error || res.statusText || "Request failed";
}

/**
 * Perform HTTP request with cookie-based auth
 * Access token is sent automatically via HttpOnly cookie
 */
async function performRequest(path, options) {
  let res;
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), options.timeout || 30000);

  try {
    res = await fetch(buildUrl(path), {
      ...options,
      signal: controller.signal,
      credentials: 'include', // IMPORTANT: Send cookies with request
      headers: buildHeaders(options.headers || {}),
    });
  } catch (err) {
    if (err.name === 'AbortError') {
      throw new Error(`Request timed out after ${(options.timeout || 30000) / 1000} seconds`);
    }
    throw new Error(`Cannot reach API at ${API_BASE_URL} (${err.message})`);
  } finally {
    clearTimeout(timeoutId);
  }

  const isJson = res.headers.get("content-type")?.includes("application/json");
  let data = null;

  if (isJson) {
    try {
      data = await res.json();
    } catch (err) {
      data = null;
    }
  }

  return { res, data };
}

export async function apiRequest(path, options = {}) {
  const firstAttempt = await performRequest(path, options);

  if (firstAttempt.res.ok) {
    // Cache local IP for offline fallback feature
    cacheLocalIPFromResponse(firstAttempt.data);
    return firstAttempt.data;
  }

  if (firstAttempt.res.status === 401) {
    const refreshed = await refreshAccessToken();

    if (refreshed) {
      const retry = await performRequest(path, options);

      if (retry.res.ok) {
        return retry.data;
      }

      if (retry.res.status === 401) {
        redirectToLogin();
      }

      const retryMessage = extractErrorMessage(retry.res, retry.data);
      const retryError = new Error(retryMessage);
      retryError.status = retry.res.status;
      throw retryError;
    }

    redirectToLogin();
    const refreshError = new Error("Session expired. Please log in again.");
    refreshError.status = 401;
    throw refreshError;
  }

  // GLOBAL VAULT INTERCEPTOR: Handle 423 Locked (Vault is locked)
  // Redirect to vault unlock page immediately
  if (firstAttempt.res.status === 423) {
    // Dispatch event for global modal handling
    if (typeof window !== "undefined") {
      window.dispatchEvent(new CustomEvent('vault-locked'));
    }
    const vaultError = new Error("Vault is locked. Please unlock to access encrypted files.");
    vaultError.status = 423;
    throw vaultError;
  }

  const message = extractErrorMessage(firstAttempt.res, firstAttempt.data);
  const error = new Error(message);
  error.status = firstAttempt.res.status;
  throw error;
}

/**
 * Search for documents using semantic search
 * @param {string} query - Search query
 * @returns {Promise<{query: string, results: Array}>}
 */
export async function searchFiles(query) {
  if (!query || !query.trim()) {
    throw new Error("Search query is required");
  }

  const encodedQuery = encodeURIComponent(query.trim());
  const response = await apiRequest(`/api/v1/search?q=${encodedQuery}`, {
    method: "GET",
  });

  return response;
}

/**
 * Unified AI Query - AI decides whether to search or answer
 * Supports both async (default) and sync (?sync=true) modes
 * 
 * @param {string} query - User query (question or search)
 * @param {Object} options - Optional configuration
 * @param {function} options.onProgress - Progress callback (status, poll count)
 * @param {boolean} options.sync - Force sync mode (default: false)
 * @returns {Promise<{
 *   mode: "search" | "answer",
 *   intent: { type: string, count_hint: string, limit: number },
 *   files?: Array<{ file_id: string, file_path: string, content: string, similarity: number }>,
 *   answer?: string,
 *   sources?: Array<{ file_id: string, file_path: string, similarity: number }>,
 *   confidence?: string,
 *   query: string
 * }>}
 */
export async function queryAI(query, options = {}) {
  if (!query?.trim()) {
    throw new Error("Query is required");
  }

  const { onProgress, sync = false } = options;
  const endpoint = sync ? "/api/v1/query?sync=true" : "/api/v1/query";

  // Step 1: Submit query
  const submitResponse = await apiRequest(endpoint, {
    method: "POST",
    body: JSON.stringify({ query: query.trim() }),
    timeout: sync ? 120000 : 10000, // Longer timeout for sync mode
  });

  // Check if this is a direct response (sync mode or backward compat)
  if (!submitResponse.job_id) {
    // Direct response - return as-is
    return submitResponse;
  }

  // Async mode - poll for result
  const jobId = submitResponse.job_id;
  const maxPolls = 80; // 80 * 1.5s = 120s max (matches JOB_TIMEOUT)
  const pollInterval = 1500; // 1.5 seconds

  if (onProgress) {
    onProgress({ status: "pending", poll: 0, jobId });
  }

  for (let i = 0; i < maxPolls; i++) {
    await new Promise(resolve => setTimeout(resolve, pollInterval));

    try {
      const statusResponse = await apiRequest(`/api/v1/jobs/${jobId}`, {
        timeout: 5000,
      });

      if (onProgress) {
        onProgress({
          status: statusResponse.status,
          poll: i + 1,
          jobId,
          elapsed: Math.round((i + 1) * pollInterval / 1000)
        });
      }

      if (statusResponse.status === "completed") {
        return statusResponse;
      }

      if (statusResponse.status === "failed") {
        throw new Error(statusResponse.error || "AI processing failed");
      }

      // Still pending or processing - continue polling
    } catch (err) {
      // If job not found yet, might be a race condition - keep trying
      if (err.status === 404 && i < 3) {
        continue;
      }
      throw err;
    }
  }

  throw new Error("AI processing timed out - please try again");
}

/**
 * Get auth headers for API requests
 * Note: Access token is now sent automatically via HttpOnly cookie
 */
export function authHeaders() {
  const csrfToken = localStorage.getItem("csrfToken") || localStorage.getItem("csrf_token") || "";
  const headers = {};
  // CSRF token still needs to be sent as header
  if (csrfToken) headers["X-CSRF-Token"] = csrfToken;
  return headers;
}

/**
 * Get system capabilities for performance estimation
 * @param {number} fileSizeBytes - Size of the file in bytes
 * @returns {Promise<{est_time_seconds: number, warning: boolean}>}
 */
export async function getSystemCapabilities(fileSizeBytes) {
  try {
    const res = await apiRequest(`/api/system/capabilities?size=${fileSizeBytes}`, {
      method: 'GET',
    });
    return res;
  } catch (err) {
    console.warn('Failed to get system capabilities:', err);
    return { est_time_seconds: 0, warning: false };
  }
}
