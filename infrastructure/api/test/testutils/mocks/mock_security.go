package mocks

import (
	"context"

	"github.com/nas-ai/api/src/domain/files"
	"github.com/nas-ai/api/src/services/security"
	"github.com/stretchr/testify/mock"
)

// ============================================================
// MockJWTService
// ============================================================

type MockJWTService struct {
	mock.Mock
}

func (m *MockJWTService) GenerateAccessToken(userID, email string) (string, error) {
	args := m.Called(userID, email)
	return args.String(0), args.Error(1)
}

func (m *MockJWTService) GenerateRefreshToken(userID, email string) (string, error) {
	args := m.Called(userID, email)
	return args.String(0), args.Error(1)
}

func (m *MockJWTService) ValidateToken(tokenString string) (*security.TokenClaims, error) {
	args := m.Called(tokenString)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*security.TokenClaims), args.Error(1)
}

func (m *MockJWTService) ExtractClaims(tokenString string) (*security.TokenClaims, error) {
	args := m.Called(tokenString)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*security.TokenClaims), args.Error(1)
}

// ============================================================
// MockPasswordService
// ============================================================

type MockPasswordService struct {
	mock.Mock
}

func (m *MockPasswordService) HashPassword(password string) (string, error) {
	args := m.Called(password)
	return args.String(0), args.Error(1)
}

func (m *MockPasswordService) ComparePassword(hashedPassword, password string) error {
	args := m.Called(hashedPassword, password)
	return args.Error(0)
}

func (m *MockPasswordService) ValidatePasswordStrength(password string) error {
	args := m.Called(password)
	return args.Error(0)
}

// ============================================================
// MockTokenService
// ============================================================

type MockTokenService struct {
	mock.Mock
}

func (m *MockTokenService) GenerateVerificationToken(ctx context.Context, userID string) (string, error) {
	args := m.Called(ctx, userID)
	return args.String(0), args.Error(1)
}

func (m *MockTokenService) ValidateVerificationToken(ctx context.Context, token string) (string, error) {
	args := m.Called(ctx, token)
	return args.String(0), args.Error(1)
}

func (m *MockTokenService) GeneratePasswordResetToken(ctx context.Context, userID string) (string, error) {
	args := m.Called(ctx, userID)
	return args.String(0), args.Error(1)
}

func (m *MockTokenService) ValidatePasswordResetToken(ctx context.Context, token string) (string, error) {
	args := m.Called(ctx, token)
	return args.String(0), args.Error(1)
}

func (m *MockTokenService) InvalidateUserTokens(ctx context.Context, userID string) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockTokenService) IsTokenRevoked(ctx context.Context, userID string, iat int64) bool {
	args := m.Called(ctx, userID, iat)
	return args.Bool(0)
}

// ============================================================
// MockEncryptionPolicyService
// ============================================================

type MockEncryptionPolicyService struct {
	mock.Mock
}

func (m *MockEncryptionPolicyService) DetermineMode(filename string, size int64, override string) files.EncryptionMode {
	args := m.Called(filename, size, override)
	return args.Get(0).(files.EncryptionMode)
}

// ============================================================
// MockEncryptionService
// ============================================================

type MockEncryptionService struct {
	mock.Mock
}

func (m *MockEncryptionService) EncryptData(plaintext []byte) ([]byte, error) {
	args := m.Called(plaintext)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockEncryptionService) DecryptData(ciphertext []byte) ([]byte, error) {
	args := m.Called(ciphertext)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockEncryptionService) IsUnlocked() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockEncryptionService) Unlock(masterPassword string) error {
	args := m.Called(masterPassword)
	return args.Error(0)
}

func (m *MockEncryptionService) Lock() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockEncryptionService) Setup(masterPassword string) error {
	args := m.Called(masterPassword)
	return args.Error(0)
}

func (m *MockEncryptionService) IsConfigured() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockEncryptionService) GetVaultPath() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockEncryptionService) SetVaultPath(path string) error {
	args := m.Called(path)
	return args.Error(0)
}
