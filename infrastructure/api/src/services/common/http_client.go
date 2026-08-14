package common

import (
	"crypto/tls"
	"net/http"
	"os"
	"strings"
	"time"
)

// NewSecureHTTPClient creates an HTTP client that automatically injects
// the X-Internal-Secret header into all requests.
//
// The client also enforces a TLS floor (min TLS 1.2) for any https:// target,
// so the internal shared secret and AI payloads are protected once the AI /
// Ollama services are reachable over TLS. Plain http:// targets keep working.
// Set INTERNAL_TLS_SKIP_VERIFY=true to accept self-signed certs on trusted
// internal networks.
func NewSecureHTTPClient(secret string, timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &secureTransport{
			secret: secret,
			base:   newTLSHardenedTransport(),
		},
	}
}

// newTLSHardenedTransport clones the default transport and pins a minimum TLS
// version, so outbound internal calls never negotiate a sniffable legacy
// protocol version.
func newTLSHardenedTransport() *http.Transport {
	transport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		transport = &http.Transport{}
	}
	cloned := transport.Clone()
	cloned.TLSClientConfig = &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: internalTLSSkipVerify(), //nolint:gosec // opt-in for internal self-signed certs
	}
	return cloned
}

// internalTLSSkipVerify reports whether certificate verification for internal
// service calls should be skipped (self-signed internal certs).
func internalTLSSkipVerify() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("INTERNAL_TLS_SKIP_VERIFY"))) {
	case "1", "t", "true", "yes", "on":
		return true
	default:
		return false
	}
}

type secureTransport struct {
	secret string
	base   http.RoundTripper
}

func (t *secureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.secret != "" {
		req.Header.Set("X-Internal-Secret", t.secret)
	}
	if t.base == nil {
		return http.DefaultTransport.RoundTrip(req)
	}
	return t.base.RoundTrip(req)
}
