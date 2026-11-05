package security

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"

	auth_repo "github.com/nas-ai/api/src/repository/auth"
	"github.com/sirupsen/logrus"
)

const (
	// recoveryCodeCount is how many codes are issued per (re)generation.
	recoveryCodeCount = 10
	// recoveryCodeLength is the number of characters per code before the
	// display separator is inserted.
	recoveryCodeLength = 10
	// recoveryCodeAlphabet has exactly 32 characters with the most ambiguous
	// ones (0/O and 1/I) removed. 32 divides 256 evenly, so the index selection
	// below is free of modulo bias. Each code carries recoveryCodeLength*5 = 50
	// bits of entropy.
	recoveryCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
)

// RecoveryCodeService generates and verifies one-time 2FA recovery codes.
// Codes are high-entropy and random, so they are stored as bcrypt hashes via
// the shared PasswordService rather than in plaintext.
type RecoveryCodeService struct {
	repo            *auth_repo.RecoveryCodeRepository
	passwordService *PasswordService
	logger          *logrus.Logger
}

// NewRecoveryCodeService creates a new recovery code service.
func NewRecoveryCodeService(repo *auth_repo.RecoveryCodeRepository, passwordService *PasswordService, logger *logrus.Logger) *RecoveryCodeService {
	return &RecoveryCodeService{repo: repo, passwordService: passwordService, logger: logger}
}

// GenerateForUser creates a fresh set of recovery codes for the user, replacing
// any that already exist, and returns the plaintext (display-formatted) codes.
// The plaintext is shown to the user exactly once; only hashes are persisted.
func (s *RecoveryCodeService) GenerateForUser(ctx context.Context, userID string) ([]string, error) {
	display := make([]string, 0, recoveryCodeCount)
	hashes := make([]string, 0, recoveryCodeCount)
	for i := 0; i < recoveryCodeCount; i++ {
		canonical, shown, err := generateRecoveryCode()
		if err != nil {
			return nil, err
		}
		hash, err := s.passwordService.HashPassword(canonical)
		if err != nil {
			return nil, fmt.Errorf("hash recovery code: %w", err)
		}
		display = append(display, shown)
		hashes = append(hashes, hash)
	}

	if err := s.repo.ReplaceForUser(ctx, userID, hashes); err != nil {
		return nil, err
	}
	s.logger.WithField("user_id", userID).Info("Recovery codes (re)generated")
	return display, nil
}

// Verify checks a submitted code against the user's unused codes and, on the
// first match, consumes it (single-use). It returns true only when a code was
// both matched and successfully consumed.
func (s *RecoveryCodeService) Verify(ctx context.Context, userID, code string) (bool, error) {
	normalized := normalizeRecoveryCode(code)
	if normalized == "" {
		return false, nil
	}

	records, err := s.repo.ListUnusedByUserID(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, rec := range records {
		// bcrypt comparison is constant-time for a given stored hash.
		if err := s.passwordService.ComparePassword(rec.CodeHash, normalized); err == nil {
			consumed, err := s.repo.MarkUsed(ctx, rec.ID)
			if err != nil {
				return false, err
			}
			return consumed, nil
		}
	}
	return false, nil
}

// RemainingCount reports how many unused recovery codes the user has left.
func (s *RecoveryCodeService) RemainingCount(ctx context.Context, userID string) (int, error) {
	return s.repo.CountUnusedByUserID(ctx, userID)
}

// generateRecoveryCode returns a random code in canonical form (no separator,
// used for hashing/verification) and display form (grouped with a hyphen).
func generateRecoveryCode() (canonical, display string, err error) {
	buf := make([]byte, recoveryCodeLength)
	if _, err := rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("generate recovery code: %w", err)
	}
	out := make([]byte, recoveryCodeLength)
	for i, b := range buf {
		out[i] = recoveryCodeAlphabet[int(b)%len(recoveryCodeAlphabet)]
	}
	canonical = string(out)
	half := recoveryCodeLength / 2
	display = canonical[:half] + "-" + canonical[half:]
	return canonical, display, nil
}

// normalizeRecoveryCode canonicalizes user input so that formatting differences
// (case, hyphens, surrounding spaces) do not affect verification.
func normalizeRecoveryCode(code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))
	code = strings.ReplaceAll(code, "-", "")
	code = strings.ReplaceAll(code, " ", "")
	return code
}
