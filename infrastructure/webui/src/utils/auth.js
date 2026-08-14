/**
 * Authentication utilities for cookie-based auth
 * 
 * SECURITY: Access and Refresh tokens are now stored as HttpOnly cookies.
 * This prevents XSS attacks from stealing tokens via JavaScript.
 * 
 * Only the CSRF token is accessible to JavaScript (stored in localStorage)
 * because it needs to be sent as a header on each request.
 */

/**
 * Get CSRF token from localStorage
 * This is the only token we keep in localStorage - it's not sensitive
 * since it requires the HttpOnly session cookie to be valid
 */
export function getCSRFToken() {
  const csrfToken = localStorage.getItem("csrfToken") || localStorage.getItem("csrf_token") || "";
  return csrfToken;
}

/**
 * Get auth headers for API requests
 * Note: Access token is now sent automatically via HttpOnly cookie
 */
export function authHeaders() {
  const csrfToken = getCSRFToken();
  const headers = {};
  // CSRF token still needs to be sent as header
  if (csrfToken) headers["X-CSRF-Token"] = csrfToken;
  return headers;
}

/**
 * Save CSRF token after login
 * Note: Access and Refresh tokens are now handled by HttpOnly cookies
 */
export function setAuth({ csrfToken = "" }) {
  if (csrfToken) {
    localStorage.setItem("csrfToken", csrfToken);
    localStorage.setItem("csrf_token", csrfToken);
  }
}

/**
 * Clear all auth data on logout
 */
export function clearAuth() {
  // Clear CSRF token from localStorage
  localStorage.removeItem("csrfToken");
  localStorage.removeItem("csrf_token");

  // Legacy cleanup - remove old token keys if they exist
  localStorage.removeItem("accessToken");
  localStorage.removeItem("refreshToken");
  localStorage.removeItem("access_token");
  localStorage.removeItem("refresh_token");
}

/**
 * Check if user is authenticated
 * Since tokens are in HttpOnly cookies, we can't check them directly.
 * We use the presence of CSRF token as a proxy (set after successful login)
 * For more robust checking, use the /api/v1/auth/status endpoint
 */
export function isAuthenticated() {
  const csrfToken = getCSRFToken();
  return Boolean(csrfToken);
}

// ============================================
// DEPRECATED - Kept for backward compatibility
// These will be removed in a future version
// ============================================

/**
 * @deprecated Tokens are now in HttpOnly cookies
 * This function is kept for backward compatibility but returns empty tokens
 */
export function getAuth() {
  console.warn("[auth.js] getAuth() is deprecated. Tokens are now in HttpOnly cookies.");
  return {
    accessToken: "",
    refreshToken: "",
    csrfToken: getCSRFToken(),
  };
}
