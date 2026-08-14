/**
 * WebAuthn / FIDO2 (YubiKey) client helpers.
 *
 * The backend speaks the standard WebAuthn JSON (go-webauthn), which encodes
 * binary values as base64url. The browser's WebAuthn API works with
 * ArrayBuffers, so these helpers convert between the two and drive the
 * registration and second-factor login ceremonies.
 */

const API_BASE = import.meta.env.VITE_API_BASE_URL || window.location.origin;

/** True when the browser exposes the WebAuthn API. */
export function isWebAuthnSupported() {
  return typeof window !== "undefined" && !!window.PublicKeyCredential;
}

function base64urlToBuffer(value) {
  const pad = "=".repeat((4 - (value.length % 4)) % 4);
  const base64 = (value + pad).replace(/-/g, "+").replace(/_/g, "/");
  const binary = atob(base64);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
  return bytes.buffer;
}

function bufferToBase64url(buffer) {
  const bytes = new Uint8Array(buffer);
  let binary = "";
  for (const b of bytes) binary += String.fromCharCode(b);
  return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

function csrfHeader(csrfToken) {
  return csrfToken ? { "X-CSRF-Token": csrfToken } : {};
}

// Convert server-provided creation options into the browser's expected shape.
function toCreationOptions(options) {
  const pk = options.publicKey;
  pk.challenge = base64urlToBuffer(pk.challenge);
  pk.user.id = base64urlToBuffer(pk.user.id);
  if (pk.excludeCredentials) {
    pk.excludeCredentials = pk.excludeCredentials.map((c) => ({
      ...c,
      id: base64urlToBuffer(c.id),
    }));
  }
  return pk;
}

// Convert server-provided request options into the browser's expected shape.
function toRequestOptions(options) {
  const pk = options.publicKey;
  pk.challenge = base64urlToBuffer(pk.challenge);
  if (pk.allowCredentials) {
    pk.allowCredentials = pk.allowCredentials.map((c) => ({
      ...c,
      id: base64urlToBuffer(c.id),
    }));
  }
  return pk;
}

/**
 * Enroll a new authenticator for the logged-in user.
 * Requires the CSRF token (management endpoints are auth + CSRF guarded).
 */
export async function registerCredential(name, csrfToken) {
  if (!isWebAuthnSupported()) throw new Error("WebAuthn wird von diesem Browser nicht unterstützt");

  const beginRes = await fetch(`${API_BASE}/auth/webauthn/register/begin`, {
    method: "POST",
    headers: { "Content-Type": "application/json", ...csrfHeader(csrfToken) },
    credentials: "include",
  });
  if (!beginRes.ok) throw new Error("Registrierung konnte nicht gestartet werden");

  const publicKey = toCreationOptions(await beginRes.json());
  const credential = await navigator.credentials.create({ publicKey });

  const body = {
    id: credential.id,
    rawId: bufferToBase64url(credential.rawId),
    type: credential.type,
    response: {
      attestationObject: bufferToBase64url(credential.response.attestationObject),
      clientDataJSON: bufferToBase64url(credential.response.clientDataJSON),
    },
  };

  const query = name ? `?name=${encodeURIComponent(name)}` : "";
  const finishRes = await fetch(`${API_BASE}/auth/webauthn/register/finish${query}`, {
    method: "POST",
    headers: { "Content-Type": "application/json", ...csrfHeader(csrfToken) },
    credentials: "include",
    body: JSON.stringify(body),
  });
  if (!finishRes.ok) throw new Error("Registrierung fehlgeschlagen");
  return finishRes.json();
}

/**
 * Complete the second-factor login after the password step returned an
 * mfa_token. Returns the final login response (with csrf_token + user).
 */
export async function loginWithWebAuthn(mfaToken, csrfToken) {
  if (!isWebAuthnSupported()) throw new Error("WebAuthn wird von diesem Browser nicht unterstützt");

  const beginRes = await fetch(`${API_BASE}/auth/login/webauthn/begin`, {
    method: "POST",
    headers: { "Content-Type": "application/json", ...csrfHeader(csrfToken) },
    credentials: "include",
    body: JSON.stringify({ mfa_token: mfaToken }),
  });
  if (!beginRes.ok) throw new Error("Zweiter Faktor konnte nicht gestartet werden");

  const publicKey = toRequestOptions(await beginRes.json());
  const assertion = await navigator.credentials.get({ publicKey });

  const body = {
    id: assertion.id,
    rawId: bufferToBase64url(assertion.rawId),
    type: assertion.type,
    response: {
      authenticatorData: bufferToBase64url(assertion.response.authenticatorData),
      clientDataJSON: bufferToBase64url(assertion.response.clientDataJSON),
      signature: bufferToBase64url(assertion.response.signature),
      userHandle: assertion.response.userHandle
        ? bufferToBase64url(assertion.response.userHandle)
        : null,
    },
  };

  const finishRes = await fetch(`${API_BASE}/auth/login/webauthn/finish`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "X-MFA-Token": mfaToken,
      ...csrfHeader(csrfToken),
    },
    credentials: "include",
    body: JSON.stringify(body),
  });
  if (!finishRes.ok) throw new Error("Anmeldung mit Sicherheitsschlüssel fehlgeschlagen");
  return finishRes.json();
}

/**
 * Complete the second-factor login using a recovery (backup) code instead of
 * the WebAuthn authenticator. Used as a fallback when the security key is not
 * available. The route is public + rate-limited and only needs the mfa_token
 * (plus the code) in the body — on success the server sets the auth cookies and
 * returns the same login response as the WebAuthn finish step.
 */
export async function loginWithRecoveryCode(mfaToken, code) {
  const res = await fetch(`${API_BASE}/auth/login/recovery`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: JSON.stringify({ mfa_token: mfaToken, code }),
  });
  if (!res.ok) throw new Error("Recovery-Code ungültig oder abgelaufen");
  return res.json();
}

/**
 * Get how many recovery codes the logged-in user still has available.
 * Returns the raw status object, e.g. { remaining: 7 }.
 */
export async function getRecoveryCodesStatus() {
  const res = await fetch(`${API_BASE}/auth/webauthn/recovery-codes`, {
    credentials: "include",
  });
  if (!res.ok) throw new Error("Status der Recovery-Codes konnte nicht geladen werden");
  return res.json();
}

/**
 * Generate a fresh set of recovery codes for the logged-in user. This
 * invalidates any previously issued codes. The plaintext codes are only
 * returned once (here) — the user must store them now. Requires the CSRF token.
 */
export async function generateRecoveryCodes(csrfToken) {
  const res = await fetch(`${API_BASE}/auth/webauthn/recovery-codes`, {
    method: "POST",
    headers: { "Content-Type": "application/json", ...csrfHeader(csrfToken) },
    credentials: "include",
  });
  if (!res.ok) throw new Error("Recovery-Codes konnten nicht erzeugt werden");
  return res.json();
}

/** List the user's registered authenticators. */
export async function listCredentials() {
  const res = await fetch(`${API_BASE}/auth/webauthn/credentials`, {
    credentials: "include",
  });
  if (!res.ok) throw new Error("Sicherheitsschlüssel konnten nicht geladen werden");
  const data = await res.json();
  return data.credentials || [];
}

/** Remove one of the user's authenticators. */
export async function deleteCredential(id, csrfToken) {
  const res = await fetch(`${API_BASE}/auth/webauthn/credentials/${encodeURIComponent(id)}`, {
    method: "DELETE",
    headers: { ...csrfHeader(csrfToken) },
    credentials: "include",
  });
  if (!res.ok) throw new Error("Sicherheitsschlüssel konnte nicht entfernt werden");
  return res.json();
}
