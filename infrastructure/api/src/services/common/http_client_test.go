package common

import (
	"crypto/tls"
	"net/http"
	"testing"
	"time"
)

// TestNewSecureHTTPClient_TLSFloor verifies the internal HTTP client enforces a
// minimum TLS version, so outbound internal calls cannot negotiate a sniffable
// legacy protocol version.
func TestNewSecureHTTPClient_TLSFloor(t *testing.T) {
	client := NewSecureHTTPClient("secret", 5*time.Second)

	st, ok := client.Transport.(*secureTransport)
	if !ok {
		t.Fatalf("expected *secureTransport, got %T", client.Transport)
	}

	tr, ok := st.base.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport base, got %T", st.base)
	}
	if tr.TLSClientConfig == nil {
		t.Fatal("expected TLSClientConfig to be set")
	}
	if tr.TLSClientConfig.MinVersion != tls.VersionTLS12 {
		t.Fatalf("expected MinVersion TLS 1.2 (%x), got %x", tls.VersionTLS12, tr.TLSClientConfig.MinVersion)
	}
}

// TestSecureTransport_InjectsSecret verifies the shared secret header is added.
func TestSecureTransport_InjectsSecret(t *testing.T) {
	tr := &secureTransport{secret: "s3cr3t", base: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if got := req.Header.Get("X-Internal-Secret"); got != "s3cr3t" {
			t.Errorf("expected secret header, got %q", got)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
	})}

	req, _ := http.NewRequest("GET", "https://internal.example/health", nil)
	if _, err := tr.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip failed: %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }
