package security

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/nas-ai/api/src/config"
	"github.com/sirupsen/logrus"
)

// TokenType represents the type of JWT token
type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

// TokenClaims represents the JWT claims structure
type TokenClaims struct {
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	TokenType TokenType `json:"token_type"`
	jwt.RegisteredClaims
}

// JWTService handles JWT token operations
type JWTService struct {
	secret []byte
	logger *logrus.Logger
}

// NewJWTService creates a new JWT service
func NewJWTService(cfg *config.Config, logger *logrus.Logger) (*JWTService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	if err := config.ValidateJWTSecret(cfg.JWTSecret); err != nil {
		return nil, err
	}

	secret := strings.TrimSpace(cfg.JWTSecret)

	return &JWTService{
		secret: []byte(secret),
		logger: logger,
	}, nil
}

// GenerateAccessToken generates a new access token (15 minute expiry)
func (s *JWTService) GenerateAccessToken(userID, email string) (string, error) {
	now := time.Now()
	// SECURITY FIX [BUG-GO-021]: Generate unique JTI (JWT ID) for token tracking and replay prevention
	jti := uuid.New().String()

	claims := TokenClaims{
		UserID:    userID,
		Email:     email,
		TokenType: AccessToken,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti, // Unique token identifier for tracking and revocation
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "nas-api",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.secret)
	if err != nil {
		s.logger.WithError(err).Error("Failed to generate access token")
		return "", fmt.Errorf("failed to generate access token: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"user_id": userID,
		"email":   email,
		"type":    AccessToken,
		"jti":     jti,
	}).Debug("Access token generated")

	return tokenString, nil
}

// GenerateRefreshToken generates a new refresh token (7 days expiry)
func (s *JWTService) GenerateRefreshToken(userID, email string) (string, error) {
	now := time.Now()
	// SECURITY FIX [BUG-GO-021]: Generate unique JTI (JWT ID) for token tracking and replay prevention
	jti := uuid.New().String()

	claims := TokenClaims{
		UserID:    userID,
		Email:     email,
		TokenType: RefreshToken,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti, // Unique token identifier for tracking and revocation
			ExpiresAt: jwt.NewNumericDate(now.Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "nas-api",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.secret)
	if err != nil {
		s.logger.WithError(err).Error("Failed to generate refresh token")
		return "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"user_id": userID,
		"email":   email,
		"type":    RefreshToken,
		"jti":     jti,
	}).Debug("Refresh token generated")

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns the claims
func (s *JWTService) ValidateToken(tokenString string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secret, nil
	})

	if err != nil {
		s.logger.WithError(err).Debug("Token validation failed")
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*TokenClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	s.logger.WithFields(logrus.Fields{
		"user_id": claims.UserID,
		"type":    claims.TokenType,
	}).Debug("Token validated successfully")

	return claims, nil
}

// ExtractClaims returns the claims of a token whose signature is valid but whose
// expiry is ignored, so callers (e.g. logout) can still read an expired token.
// The signature is always verified — an unsigned or forged token is rejected.
//
// Only expiry is tolerated; every other claim is still validated. Disabling
// claims validation wholesale would also drop the "not before" and "issued at"
// checks, which nothing here needs to skip.
//
// Tolerating ErrTokenExpired cannot let a forged token through: the parser
// verifies the signature first and returns immediately on mismatch, so a
// signature failure never surfaces as an expiry error.
func (s *JWTService) ExtractClaims(tokenString string) (*TokenClaims, error) {
	parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))

	token, err := parser.ParseWithClaims(tokenString, &TokenClaims{}, func(*jwt.Token) (interface{}, error) {
		return s.secret, nil
	})
	if err != nil && !errors.Is(err, jwt.ErrTokenExpired) {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}
	if token == nil {
		return nil, fmt.Errorf("failed to parse token")
	}

	claims, ok := token.Claims.(*TokenClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}
