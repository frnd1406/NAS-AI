package server

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// defaultDevTLSDir is where the persistent dev certificate is cached when
// DEV_TLS_DIR is not set. Mount a volume here to keep the pin stable across
// container recreation (otherwise it survives restarts within the same
// container, which is enough for TOFU testing).
const defaultDevTLSDir = "/app/dev-certs"

// resolveTLSMaterial decides how the server obtains its TLS certificate.
//
//   - If TLS_CERT_FILE and TLS_KEY_FILE are configured, those files are used
//     (returned as paths for ListenAndServeTLS).
//   - Otherwise, in production this is a fatal misconfiguration (fail-fast).
//   - In non-production environments an ephemeral self-signed certificate is
//     generated in memory and injected into srv.TLSConfig; empty paths are
//     returned so ListenAndServeTLS uses that certificate.
func (s *Server) resolveTLSMaterial(srv *http.Server) (certFile, keyFile string, err error) {
	if s.cfg.TLSCertFile != "" && s.cfg.TLSKeyFile != "" {
		s.logger.WithField("cert", s.cfg.TLSCertFile).Info("TLS: using configured certificate")
		return s.cfg.TLSCertFile, s.cfg.TLSKeyFile, nil
	}

	if s.cfg.Environment == "production" {
		return "", "", fmt.Errorf(
			"CRITICAL: TLS_CERT_FILE and TLS_KEY_FILE are required in production (no plaintext HTTP allowed)")
	}

	dir := os.Getenv("DEV_TLS_DIR")
	if dir == "" {
		dir = defaultDevTLSDir
	}
	cert, genErr := loadOrCreatePersistentDevCert(dir)
	if genErr != nil {
		return "", "", fmt.Errorf("failed to obtain self-signed certificate: %w", genErr)
	}
	if srv.TLSConfig == nil {
		srv.TLSConfig = newBaseTLSConfig()
	}
	srv.TLSConfig.Certificates = []tls.Certificate{cert}
	s.logger.WithField("dir", dir).Warn(
		"⚠️  TLS: using persistent self-signed dev certificate (dev/lab only, not for production)")
	return "", "", nil
}

// newBaseTLSConfig returns a hardened TLS configuration used by the API server.
// TLS 1.2 is the minimum; older, sniffable protocol versions are refused.
func newBaseTLSConfig() *tls.Config {
	return &tls.Config{
		MinVersion:               tls.VersionTLS12,
		PreferServerCipherSuites: true,
	}
}

// generateSelfSignedCert creates an ephemeral, in-memory self-signed
// certificate valid for localhost. It is used in non-production environments
// so the HTTPS lab proof works without provisioning cert files on disk.
//
// The private key never touches the filesystem; it only lives in RAM for the
// lifetime of the process, matching the project's zero-persistence posture.
func generateSelfSignedCert() (tls.Certificate, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generate ECDSA key: %w", err)
	}

	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serial, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generate serial number: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "nas-ai-local"},
		NotBefore:    time.Now().Add(-1 * time.Hour),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
	}

	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("create certificate: %w", err)
	}

	return tls.Certificate{
		Certificate: [][]byte{der},
		PrivateKey:  priv,
		Leaf:        &template,
	}, nil
}

// loadOrCreatePersistentDevCert returns a self-signed dev certificate that is
// STABLE across process restarts: it is cached as PEM files under dir. If a
// valid cached cert exists it is reused; otherwise a fresh one is generated and
// written. A stable cert is required so the desktop client's TOFU pinning does
// not flag every server restart as a certificate mismatch.
//
// Dev/lab only. Writing the private key to disk here is a deliberate trade-off
// for the stable-fingerprint requirement; production uses TLS_CERT_FILE/KEY.
func loadOrCreatePersistentDevCert(dir string) (tls.Certificate, error) {
	certPath := filepath.Join(dir, "dev-cert.pem")
	keyPath := filepath.Join(dir, "dev-key.pem")

	if fileExists(certPath) && fileExists(keyPath) {
		if cert, err := tls.LoadX509KeyPair(certPath, keyPath); err == nil && devCertStillValid(cert) {
			return cert, nil
		}
		// Fall through to regenerate on load error or expiry.
	}

	cert, err := generateSelfSignedCert()
	if err != nil {
		return tls.Certificate{}, err
	}
	if err := writeDevCertPEM(dir, certPath, keyPath, cert); err != nil {
		return tls.Certificate{}, fmt.Errorf("persist dev certificate: %w", err)
	}
	return cert, nil
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

// devCertStillValid parses the leaf and checks it has not expired.
func devCertStillValid(cert tls.Certificate) bool {
	if len(cert.Certificate) == 0 {
		return false
	}
	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return false
	}
	return time.Now().Before(leaf.NotAfter)
}

// writeDevCertPEM writes the certificate and its EC private key as PEM files.
// The directory is created 0700 and the key file 0600 (private key protection).
func writeDevCertPEM(dir, certPath, keyPath string, cert tls.Certificate) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]})
	if certPEM == nil {
		return fmt.Errorf("encode certificate PEM")
	}
	if err := os.WriteFile(certPath, certPEM, 0o644); err != nil {
		return err
	}

	ecKey, ok := cert.PrivateKey.(*ecdsa.PrivateKey)
	if !ok {
		return fmt.Errorf("unexpected private key type %T", cert.PrivateKey)
	}
	keyDER, err := x509.MarshalECPrivateKey(ecKey)
	if err != nil {
		return err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	if keyPEM == nil {
		return fmt.Errorf("encode private key PEM")
	}
	return os.WriteFile(keyPath, keyPEM, 0o600)
}
