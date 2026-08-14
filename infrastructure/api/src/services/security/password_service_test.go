package security

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPasswordService_HashPassword(t *testing.T) {
	ps := NewPasswordService()

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "valid password",
			password: "SecurePassword123",
			wantErr:  false,
		},
		{
			name:     "empty password",
			password: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := ps.HashPassword(tt.password)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, hash)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, hash)
				// Hash should be different from password
				assert.NotEqual(t, tt.password, hash)
				// Hash should be bcrypt format (starts with $2a$)
				assert.Contains(t, hash, "$2a$")
			}
		})
	}
}

func TestPasswordService_ComparePassword(t *testing.T) {
	ps := NewPasswordService()
	password := "SecurePassword123"
	hash, err := ps.HashPassword(password)
	require.NoError(t, err)

	tests := []struct {
		name     string
		password string
		hash     string
		wantErr  bool
	}{
		{
			name:     "correct password",
			password: password,
			hash:     hash,
			wantErr:  false,
		},
		{
			name:     "incorrect password",
			password: "WrongPassword123",
			hash:     hash,
			wantErr:  true,
		},
		{
			name:     "empty password",
			password: "",
			hash:     hash,
			wantErr:  true,
		},
		{
			name:     "invalid hash",
			password: password,
			hash:     "invalid-hash",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ps.ComparePassword(tt.hash, tt.password)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPasswordService_ValidatePasswordStrength(t *testing.T) {
	ps := NewPasswordService()

	tests := []struct {
		name     string
		password string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid strong password",
			password: "SecurePassword123",
			wantErr:  false,
		},
		{
			name:     "too short",
			password: "Short1",
			wantErr:  true,
			errMsg:   "at least 8 characters",
		},
		{
			name:     "no uppercase",
			password: "weakpassword123",
			wantErr:  true,
			errMsg:   "uppercase letter",
		},
		{
			name:     "no lowercase",
			password: "WEAKPASSWORD123",
			wantErr:  true,
			errMsg:   "lowercase letter",
		},
		{
			name:     "no number",
			password: "WeakPassword",
			wantErr:  true,
			errMsg:   "number",
		},
		{
			name:     "empty password",
			password: "",
			wantErr:  true,
			errMsg:   "at least 8 characters",
		},
		{
			name:     "with special characters",
			password: "SecureP@ssw0rd!",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ps.ValidatePasswordStrength(tt.password)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
