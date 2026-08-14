package security

import (
	"crypto/rand"
	"encoding/base64"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/nas-ai/api/src/config"
	"github.com/sirupsen/logrus"
)

// randomSecret produces a per-run signing secret, so no key material is
// checked into the repository.
func randomSecret(t *testing.T) string {
	t.Helper()

	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		t.Fatalf("generate secret: %v", err)
	}
	return base64.StdEncoding.EncodeToString(buf)
}

func newExtractTestService(t *testing.T, secret string) *JWTService {
	t.Helper()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	svc, err := NewJWTService(&config.Config{JWTSecret: secret}, logger)
	if err != nil {
		t.Fatalf("NewJWTService failed: %v", err)
	}
	return svc
}

// signWith builds a token signed with an arbitrary secret, so the test can
// forge one the service must reject.
func signWith(t *testing.T, secret string, expiresAt time.Time) string {
	t.Helper()

	claims := &TokenClaims{
		UserID:    "user-1",
		Email:     "user@example.com",
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

// The whole point of ExtractClaims: an expired but correctly signed token is
// still readable (logout needs its expiry to size the blacklist TTL).
func TestExtractClaims_AcceptsExpiredButValidlySigned(t *testing.T) {
	secret := randomSecret(t)
	svc := newExtractTestService(t, secret)
	token := signWith(t, secret, time.Now().Add(-1*time.Hour))

	claims, err := svc.ExtractClaims(token)
	if err != nil {
		t.Fatalf("expected an expired but validly signed token to be readable: %v", err)
	}
	if claims.UserID != "user-1" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

// A token signed with the wrong key must be rejected — this is the actual
// hardening: ExtractClaims previously used ParseUnverified.
func TestExtractClaims_RejectsForgedSignature(t *testing.T) {
	svc := newExtractTestService(t, randomSecret(t))
	// Signed with a different key than the service holds.
	token := signWith(t, randomSecret(t), time.Now().Add(1*time.Hour))

	if _, err := svc.ExtractClaims(token); err == nil {
		t.Fatal("expected a token signed with a foreign key to be rejected")
	}
}

// Only expiry is tolerated: a correctly signed token that is not yet valid must
// still be rejected, proving claims validation was narrowed rather than disabled.
func TestExtractClaims_RejectsNotYetValidToken(t *testing.T) {
	secret := randomSecret(t)
	svc := newExtractTestService(t, secret)

	claims := &TokenClaims{
		UserID:    "user-1",
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			NotBefore: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(2 * time.Hour)),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	if _, err := svc.ExtractClaims(signed); err == nil {
		t.Fatal("expected a not-yet-valid token to be rejected")
	}
}

// An unsigned ("alg: none") token must not be accepted either.
func TestExtractClaims_RejectsUnsignedToken(t *testing.T) {
	svc := newExtractTestService(t, randomSecret(t))

	claims := &TokenClaims{UserID: "user-1", TokenType: "access"}
	unsigned, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("build unsigned token: %v", err)
	}

	if _, err := svc.ExtractClaims(unsigned); err == nil {
		t.Fatal("expected an unsigned token to be rejected")
	}
}
