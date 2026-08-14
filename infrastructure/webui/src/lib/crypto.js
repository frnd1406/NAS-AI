/**
 * WebCrypto API Wrapper for Client-Side Encryption (Zero-Knowledge)
 * Uses AES-GCM for file encryption and PBKDF2 for key derivation.
 */

// Configuration
const PBKDF2_ITERATIONS = 100000;
const SALT_LENGTH = 16;
const IV_LENGTH = 12; // Standard for AES-GCM
const KEY_LENGTH_BITS = 256;

/**
 * Generates a cryptographically strong random salt
 * @returns {Uint8Array} 16 bytes salt
 */
export function generateSalt() {
    return window.crypto.getRandomValues(new Uint8Array(SALT_LENGTH));
}

/**
 * Generates a random Initialization Vector (IV)
 * @returns {Uint8Array} 12 bytes IV
 */
export function generateIV() {
    return window.crypto.getRandomValues(new Uint8Array(IV_LENGTH));
}

/**
 * Derives a cryptographic key from a password using PBKDF2
 * @param {string} password - User's vault password
 * @param {Uint8Array} salt - Salt used for derivation
 * @returns {Promise<CryptoKey>} AES-GCM key
 */
export async function deriveKey(password, salt) {
    const textEncoder = new TextEncoder();
    const passwordBuffer = textEncoder.encode(password);

    // 1. Import password as key material
    const keyMaterial = await window.crypto.subtle.importKey(
        "raw",
        passwordBuffer,
        { name: "PBKDF2" },
        false,
        ["deriveBits", "deriveKey"]
    );

    // 2. Derive AES-GCM key
    return window.crypto.subtle.deriveKey(
        {
            name: "PBKDF2",
            salt: salt,
            iterations: PBKDF2_ITERATIONS,
            hash: "SHA-256",
        },
        keyMaterial,
        { name: "AES-GCM", length: KEY_LENGTH_BITS },
        false, // Key cannot be exported (Security!)
        ["encrypt", "decrypt"]
    );
}

/**
 * Encrypts a data chunk using AES-GCM
 * @param {ArrayBuffer} data - Data to encrypt
 * @param {CryptoKey} key - Derived key
 * @param {Uint8Array} iv - Initialization Vector
 * @returns {Promise<ArrayBuffer>} Encrypted data (Ciphertext + Auth Tag)
 */
export async function encryptChunk(data, key, iv) {
    return window.crypto.subtle.encrypt(
        {
            name: "AES-GCM",
            iv: iv,
        },
        key,
        data
    );
}

/**
 * Decrypts a data chunk using AES-GCM
 * @param {ArrayBuffer} encryptedData - Data to decrypt
 * @param {CryptoKey} key - Derived key
 * @param {Uint8Array} iv - Initialization Vector
 * @returns {Promise<ArrayBuffer>} Decrypted data
 */
export async function decryptChunk(encryptedData, key, iv) {
    return window.crypto.subtle.decrypt(
        {
            name: "AES-GCM",
            iv: iv,
        },
        key,
        encryptedData
    );
}

/**
 * Generates a random Recovery Key (formatted 32-char string)
 * Used if user forgets password.
 */
export function generateRecoveryKey() {
    const array = new Uint8Array(24); // 24 bytes => ~32 Base64 chars
    window.crypto.getRandomValues(array);

    // Convert to simplified Base64 (URL safe)
    let str = btoa(String.fromCharCode.apply(null, array))
        .replace(/\+/g, '-')
        .replace(/\//g, '_')
        .replace(/=/g, '');

    // Format as groups: XXXX-XXXX-XXXX-...
    return str.match(/.{1,4}/g).join("-");
}

/**
 * Converts ArrayBuffer to Base64 string (for storage/transport)
 */
export function arrayBufferToBase64(buffer) {
    let binary = '';
    const bytes = new Uint8Array(buffer);
    const len = bytes.byteLength;
    for (let i = 0; i < len; i++) {
        binary += String.fromCharCode(bytes[i]);
    }
    return window.btoa(binary);
}

/**
 * Converts Base64 string to Uint8Array
 */
export function base64ToUint8Array(base64) {
    const binary_string = window.atob(base64);
    const len = binary_string.length;
    const bytes = new Uint8Array(len);
    for (let i = 0; i < len; i++) {
        bytes[i] = binary_string.charCodeAt(i);
    }
    return bytes;
}
