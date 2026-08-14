/**
 * Test script for verifying CryptoService logic (Node.js compatible)
 * Requires OpenSSL (built-in in Node 19+) for webcrypto polyfill behavior check.
 */

// Simple polyfill if needed or mock
if (!globalThis.crypto) {
    const nodeCrypto = require('crypto');
    globalThis.crypto = nodeCrypto.webcrypto;
}

// Emulate implementation from src/lib/crypto.js to verify math
// We copy the code because imports are ESM and environment might be CJS/mixed without transpiler
// This verifies the ALGORITHM logic.

const PBKDF2_ITERATIONS = 100000;
const SALT_LENGTH = 16;
const IV_LENGTH = 12;
const KEY_LENGTH_BITS = 256;

async function deriveKey(password, salt) {
    const textEncoder = new TextEncoder();
    const passwordBuffer = textEncoder.encode(password);

    const keyMaterial = await crypto.subtle.importKey(
        "raw",
        passwordBuffer,
        { name: "PBKDF2" },
        false,
        ["deriveBits", "deriveKey"]
    );

    return crypto.subtle.deriveKey(
        {
            name: "PBKDF2",
            salt: salt,
            iterations: PBKDF2_ITERATIONS,
            hash: "SHA-256",
        },
        keyMaterial,
        { name: "AES-GCM", length: KEY_LENGTH_BITS },
        true, // Must be extractable for test verification if we want to export
        ["encrypt", "decrypt"]
    );
}

async function encrypt(data, key, iv) {
    return crypto.subtle.encrypt(
        { name: "AES-GCM", iv: iv },
        key,
        data
    );
}

async function decrypt(data, key, iv) {
    return crypto.subtle.decrypt(
        { name: "AES-GCM", iv: iv },
        key,
        data
    );
}

// TEST RUNNER
async function runTests() {
    console.log("🔒 Starting Crypto Verification Tests...");

    // 1. Salt Gen
    const salt = crypto.getRandomValues(new Uint8Array(SALT_LENGTH));
    console.log("✅ Salt Generated:", salt.length, "bytes");

    // 2. Key Derivation
    const password = "correct-horse-battery-staple";
    const start = performance.now();
    const key = await deriveKey(password, salt);
    const time = performance.now() - start;
    console.log(`✅ Key Derived (PBKDF2, ${PBKDF2_ITERATIONS} iters): ${time.toFixed(2)}ms`);

    // 3. Encrypt / Decrypt
    const plainText = "Hello World! This is a secret message.";
    const enc = new TextEncoder();
    const data = enc.encode(plainText);
    const iv = crypto.getRandomValues(new Uint8Array(IV_LENGTH));

    // Encrypt
    const ciphertext = await encrypt(data, key, iv);
    console.log("✅ Encryption successful. Size:", ciphertext.byteLength, "bytes");

    // Decrypt
    const decryptedBuffer = await decrypt(ciphertext, key, iv);
    const dec = new TextDecoder();
    const decryptedText = dec.decode(decryptedBuffer);

    console.log("✅ Decryption Result:", decryptedText);

    if (decryptedText === plainText) {
        console.log("🎉 SUCCESS: Roundtrip match!");
    } else {
        console.error("❌ FAILURE: Mismatch!");
        process.exit(1);
    }
}

runTests().catch(err => {
    console.error("Test Error:", err);
    process.exit(1);
});
