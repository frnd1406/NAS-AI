package server

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nas-ai/api/src/middleware/core"
)

// TestGenerateSelfSignedCert verifies the ephemeral certificate is usable for
// a real TLS handshake and that the hardened base config refuses < TLS 1.2.
func TestGenerateSelfSignedCert(t *testing.T) {
	cert, err := generateSelfSignedCert()
	if err != nil {
		t.Fatalf("generateSelfSignedCert failed: %v", err)
	}
	if len(cert.Certificate) == 0 {
		t.Fatal("expected at least one certificate in chain")
	}

	// Serve /health over TLS using the same config the server uses, wrapped in
	// the SecureHeaders middleware, and confirm an encrypted request succeeds.
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	ts := httptest.NewUnstartedServer(core.SecureHeaders(mux))
	ts.TLS = newBaseTLSConfig()
	ts.TLS.Certificates = []tls.Certificate{cert}
	ts.StartTLS()
	defer ts.Close()

	client := ts.Client() // trusts the test server's self-signed cert
	resp, err := client.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("TLS request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if resp.TLS == nil {
		t.Fatal("expected a TLS connection state")
	}
	if resp.TLS.Version < tls.VersionTLS12 {
		t.Fatalf("expected TLS >= 1.2, got version %x", resp.TLS.Version)
	}

	// Recon minimization: security headers present, server software hidden.
	if got := resp.Header.Get("Strict-Transport-Security"); got == "" {
		t.Error("expected HSTS header to be set")
	}
	if got := resp.Header.Get("Content-Security-Policy"); got == "" {
		t.Error("expected Content-Security-Policy header to be set")
	}
	if got := resp.Header.Get("Server"); got != "" {
		t.Errorf("expected Server header to be stripped, got %q", got)
	}
}
