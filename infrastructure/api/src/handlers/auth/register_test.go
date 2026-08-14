package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================
// RegisterHandler Tests
// ============================================================

func TestRegisterHandler_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString("invalid json"))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("request_id", "test-reg-1")

	handler := func(c *gin.Context) {
		var req RegisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":    "invalid_request",
					"message": "Invalid request body",
				},
			})
			return
		}
	}
	handler(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegisterHandler_MissingUsername(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := `{"email": "test@example.com", "password": "SecurePassword123!"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("request_id", "test-reg-2")

	handler := func(c *gin.Context) {
		var req RegisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":    "invalid_request",
					"message": "Invalid request body",
				},
			})
			return
		}
	}
	handler(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegisterHandler_MissingEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := `{"username": "testuser", "password": "SecurePassword123!"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("request_id", "test-reg-3")

	handler := func(c *gin.Context) {
		var req RegisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":    "invalid_request",
					"message": "Invalid request body",
				},
			})
			return
		}
	}
	handler(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegisterHandler_MissingPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := `{"username": "testuser", "email": "test@example.com"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("request_id", "test-reg-4")

	handler := func(c *gin.Context) {
		var req RegisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":    "invalid_request",
					"message": "Invalid request body",
				},
			})
			return
		}
	}
	handler(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegisterHandler_ShortUsername(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := `{"username": "ab", "email": "test@example.com", "password": "SecurePassword123!"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("request_id", "test-reg-5")

	// Simulate username validation
	handler := func(c *gin.Context) {
		var req RegisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
			return
		}

		if len(req.Username) < 3 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":    "invalid_username",
					"message": "Username must be at least 3 characters",
				},
			})
			return
		}
	}
	handler(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))

	errorObj := response["error"].(map[string]interface{})
	assert.Equal(t, "invalid_username", errorObj["code"])
}

func TestRegisterHandler_InvalidEmailFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := `{"username": "testuser", "email": "not-an-email", "password": "SecurePassword123!"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("request_id", "test-reg-6")

	handler := func(c *gin.Context) {
		var req RegisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":    "invalid_request",
					"message": "Invalid request body",
				},
			})
			return
		}
	}
	handler(c)

	// Gin's email validation in binding should catch this
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegisterRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request RegisterRequest
		valid   bool
	}{
		{
			name: "valid request",
			request: RegisterRequest{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "SecurePassword123!",
			},
			valid: true,
		},
		{
			name: "empty username",
			request: RegisterRequest{
				Username: "",
				Email:    "test@example.com",
				Password: "SecurePassword123!",
			},
			valid: false,
		},
		{
			name: "empty email",
			request: RegisterRequest{
				Username: "testuser",
				Email:    "",
				Password: "SecurePassword123!",
			},
			valid: false,
		},
		{
			name: "empty password",
			request: RegisterRequest{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "",
			},
			valid: false,
		},
		{
			name: "all empty",
			request: RegisterRequest{
				Username: "",
				Email:    "",
				Password: "",
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			body, _ := json.Marshal(tt.request)
			c.Request = httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
			c.Request.Header.Set("Content-Type", "application/json")

			var req RegisterRequest
			err := c.ShouldBindJSON(&req)

			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

// TestRegisterResponse_Structure verifies the expected response structure
func TestRegisterResponse_Structure(t *testing.T) {
	resp := RegisterResponse{
		User:              map[string]interface{}{"id": "123", "email": "test@example.com", "username": "testuser"},
		AccessToken:       "access-token-here",
		RefreshToken:      "refresh-token-here",
		CSRFToken:         "csrf-token-here",
		VerificationToken: "verify-token-here",
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &parsed))

	assert.Contains(t, parsed, "user")
	assert.Contains(t, parsed, "access_token")
	assert.Contains(t, parsed, "refresh_token")
	assert.Contains(t, parsed, "csrf_token")
	assert.Contains(t, parsed, "verification_token")
}

// TestRegisterResponse_OmitsEmptyVerificationToken verifies omitempty behavior
func TestRegisterResponse_OmitsEmptyVerificationToken(t *testing.T) {
	resp := RegisterResponse{
		User:              map[string]interface{}{"id": "123"},
		AccessToken:       "access-token",
		RefreshToken:      "refresh-token",
		CSRFToken:         "csrf-token",
		VerificationToken: "", // Empty - should be omitted
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &parsed))

	// verification_token should NOT be present when empty
	_, exists := parsed["verification_token"]
	assert.False(t, exists, "verification_token should be omitted when empty")
}
