package security

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/nas-ai/api/src/database"
	"github.com/nas-ai/api/src/domain/auth"
	auth_repo "github.com/nas-ai/api/src/repository/auth"
	"github.com/sirupsen/logrus"
)

const (
	// mfaSessionTTL bounds how long a begin/finish ceremony (and the pending
	// second-factor step) stays valid.
	mfaSessionTTL = 5 * time.Minute

	redisKeyRegSession   = "mfa:reg:"     // + userID
	redisKeyLoginSession = "mfa:login:"   // + mfaToken
	redisKeyMFAPending   = "mfa:pending:" // + mfaToken
)

// WebAuthnService encapsulates the WebAuthn relying party, the transient
// ceremony/session storage (Redis) and credential persistence. It is nil when
// WebAuthn is not configured, in which case the login stays password-only.
type WebAuthnService struct {
	web    *webauthn.WebAuthn
	redis  *database.RedisClient
	repo   *auth_repo.WebAuthnCredentialRepository
	logger *logrus.Logger
}

// NewWebAuthnService builds the service from relying-party configuration. It
// returns (nil, nil) when rpID is empty so callers can treat WebAuthn as an
// optional, disabled feature.
func NewWebAuthnService(
	rpID, rpDisplayName string,
	rpOrigins []string,
	redis *database.RedisClient,
	repo *auth_repo.WebAuthnCredentialRepository,
	logger *logrus.Logger,
) (*WebAuthnService, error) {
	if rpID == "" {
		return nil, nil
	}

	web, err := webauthn.New(&webauthn.Config{
		RPID:          rpID,
		RPDisplayName: rpDisplayName,
		RPOrigins:     rpOrigins,
	})
	if err != nil {
		return nil, fmt.Errorf("init webauthn: %w", err)
	}

	return &WebAuthnService{web: web, redis: redis, repo: repo, logger: logger}, nil
}

// webAuthnUser adapts a domain user + its stored credentials to the library's
// webauthn.User interface.
type webAuthnUser struct {
	id          []byte
	name        string
	displayName string
	credentials []webauthn.Credential
}

func (u *webAuthnUser) WebAuthnID() []byte                         { return u.id }
func (u *webAuthnUser) WebAuthnName() string                       { return u.name }
func (u *webAuthnUser) WebAuthnDisplayName() string                { return u.displayName }
func (u *webAuthnUser) WebAuthnCredentials() []webauthn.Credential { return u.credentials }

// buildUser loads the user's stored credentials and wraps them for the library.
func (s *WebAuthnService) buildUser(ctx context.Context, user *auth.User) (*webAuthnUser, error) {
	records, err := s.repo.ListByUserID(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	creds := make([]webauthn.Credential, 0, len(records))
	for _, rec := range records {
		var c webauthn.Credential
		if err := json.Unmarshal(rec.CredentialJSON, &c); err != nil {
			s.logger.WithError(err).Warn("skipping unparseable webauthn credential")
			continue
		}
		creds = append(creds, c)
	}
	return &webAuthnUser{
		id:          []byte(user.ID),
		name:        user.Email,
		displayName: user.Username,
		credentials: creds,
	}, nil
}

// --- Registration ---------------------------------------------------------

// BeginRegistration starts enrolling a new authenticator for a logged-in user.
func (s *WebAuthnService) BeginRegistration(ctx context.Context, user *auth.User) (*protocol.CredentialCreation, error) {
	waUser, err := s.buildUser(ctx, user)
	if err != nil {
		return nil, err
	}

	creation, session, err := s.web.BeginRegistration(waUser)
	if err != nil {
		return nil, fmt.Errorf("begin registration: %w", err)
	}

	if err := s.storeSession(ctx, redisKeyRegSession+user.ID, session); err != nil {
		return nil, err
	}
	return creation, nil
}

// FinishRegistration verifies the attestation and persists the new credential.
func (s *WebAuthnService) FinishRegistration(ctx context.Context, user *auth.User, name string, body io.Reader) error {
	session, err := s.loadSession(ctx, redisKeyRegSession+user.ID, true)
	if err != nil {
		return err
	}

	waUser, err := s.buildUser(ctx, user)
	if err != nil {
		return err
	}

	parsed, err := protocol.ParseCredentialCreationResponseBody(body)
	if err != nil {
		return fmt.Errorf("parse registration response: %w", err)
	}

	credential, err := s.web.CreateCredential(waUser, *session, parsed)
	if err != nil {
		return fmt.Errorf("create credential: %w", err)
	}

	credJSON, err := json.Marshal(credential)
	if err != nil {
		return fmt.Errorf("marshal credential: %w", err)
	}

	credID := base64.RawURLEncoding.EncodeToString(credential.ID)
	return s.repo.Create(ctx, user.ID, name, credID, credJSON, int64(credential.Authenticator.SignCount))
}

// --- Login (second factor) ------------------------------------------------

// BeginLogin starts the assertion ceremony for a user who already passed the
// password check. The ceremony session is keyed by the one-time mfaToken.
func (s *WebAuthnService) BeginLogin(ctx context.Context, mfaToken string, user *auth.User) (*protocol.CredentialAssertion, error) {
	waUser, err := s.buildUser(ctx, user)
	if err != nil {
		return nil, err
	}

	assertion, session, err := s.web.BeginLogin(waUser)
	if err != nil {
		return nil, fmt.Errorf("begin login: %w", err)
	}

	if err := s.storeSession(ctx, redisKeyLoginSession+mfaToken, session); err != nil {
		return nil, err
	}
	return assertion, nil
}

// FinishLogin verifies the assertion and updates the credential's sign count.
func (s *WebAuthnService) FinishLogin(ctx context.Context, mfaToken string, user *auth.User, body io.Reader) error {
	session, err := s.loadSession(ctx, redisKeyLoginSession+mfaToken, true)
	if err != nil {
		return err
	}

	waUser, err := s.buildUser(ctx, user)
	if err != nil {
		return err
	}

	parsed, err := protocol.ParseCredentialRequestResponseBody(body)
	if err != nil {
		return fmt.Errorf("parse assertion response: %w", err)
	}

	credential, err := s.web.ValidateLogin(waUser, *session, parsed)
	if err != nil {
		return fmt.Errorf("validate login: %w", err)
	}

	credJSON, err := json.Marshal(credential)
	if err != nil {
		return fmt.Errorf("marshal credential: %w", err)
	}
	credID := base64.RawURLEncoding.EncodeToString(credential.ID)
	if err := s.repo.UpdateAfterLogin(ctx, credID, credJSON, int64(credential.Authenticator.SignCount)); err != nil {
		s.logger.WithError(err).Warn("failed to update webauthn credential after login")
	}
	return nil
}

// --- MFA pending token (password ok, awaiting second factor) --------------

// GenerateMFAPendingToken issues a short-lived, single-use token that proves the
// password step succeeded and binds the pending second factor to a user.
func (s *WebAuthnService) GenerateMFAPendingToken(ctx context.Context, userID string) (string, error) {
	token, err := randomToken()
	if err != nil {
		return "", err
	}
	rctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := s.redis.Set(rctx, redisKeyMFAPending+token, userID, mfaSessionTTL).Err(); err != nil {
		return "", fmt.Errorf("store mfa pending token: %w", err)
	}
	return token, nil
}

// ValidateMFAPendingToken resolves the token to its user ID. When consume is
// true the token is deleted (use at the final finish step); when false it is
// only peeked (use at the begin step).
func (s *WebAuthnService) ValidateMFAPendingToken(ctx context.Context, token string, consume bool) (string, error) {
	rctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	userID, err := s.redis.Get(rctx, redisKeyMFAPending+token).Result()
	if err != nil {
		return "", fmt.Errorf("invalid or expired mfa token")
	}
	if consume {
		if err := s.redis.Del(rctx, redisKeyMFAPending+token).Err(); err != nil {
			s.logger.WithError(err).Warn("failed to delete mfa pending token")
		}
	}
	return userID, nil
}

// --- Redis session helpers ------------------------------------------------

func (s *WebAuthnService) storeSession(ctx context.Context, key string, session *webauthn.SessionData) error {
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	rctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := s.redis.Set(rctx, key, data, mfaSessionTTL).Err(); err != nil {
		return fmt.Errorf("store session: %w", err)
	}
	return nil
}

func (s *WebAuthnService) loadSession(ctx context.Context, key string, consume bool) (*webauthn.SessionData, error) {
	rctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	data, err := s.redis.Get(rctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("session not found or expired")
	}
	if consume {
		if err := s.redis.Del(rctx, key).Err(); err != nil {
			s.logger.WithError(err).Warn("failed to delete webauthn session")
		}
	}

	var session webauthn.SessionData
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, fmt.Errorf("unmarshal session: %w", err)
	}
	return &session, nil
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate random token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
