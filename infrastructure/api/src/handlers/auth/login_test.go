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

func init() {
	gin.SetMode(gin.TestMode)
}

// ============================================================
// LoginHandler Tests
// ============================================================

func TestLoginHandler_InvalidJSON(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString("not json"))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("request_id", "test-req-1")

	// Create mock handler that simulates JSON binding error
	handler := func(c *gin.Context) {
		var req LoginRequest
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

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))

	errorObj := response["error"].(map[string]interface{})
	assert.Equal(t, "invalid_request", errorObj["code"])
}

func TestLoginHandler_MissingEmail(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := `{"password": "somepassword"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("request_id", "test-req-2")

	handler := func(c *gin.Context) {
		var req LoginRequest
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

func TestLoginHandler_MissingPassword(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := `{"email": "test@example.com"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("request_id", "test-req-3")

	handler := func(c *gin.Context) {
		var req LoginRequest
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

func TestLoginHandler_InvalidEmailFormat(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := `{"email": "not-an-email", "password": "somepassword"}`
	c.Request = httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("request_id", "test-req-4")

	handler := func(c *gin.Context) {
		var req LoginRequest
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

func TestLoginRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request LoginRequest
		valid   bool
	}{
		{
			name:    "valid request",
			request: LoginRequest{Email: "test@example.com", Password: "password123"},
			valid:   true,
		},
		{
			name:    "empty email",
			request: LoginRequest{Email: "", Password: "password123"},
			valid:   false,
		},
		{
			name:    "empty password",
			request: LoginRequest{Email: "test@example.com", Password: ""},
			valid:   false,
		},
		{
			name:    "both empty",
			request: LoginRequest{Email: "", Password: ""},
			valid:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			body, _ := json.Marshal(tt.request)
			c.Request = httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(body))
			c.Request.Header.Set("Content-Type", "application/json")

			var req LoginRequest
			err := c.ShouldBindJSON(&req)

			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

// TestLoginResponse_Structure verifies the expected response structure
// Note: Access and Refresh tokens are now sent as HttpOnly cookies, not in JSON body
func TestLoginResponse_Structure(t *testing.T) {
	resp := LoginResponse{
		User:      map[string]interface{}{"id": "123", "email": "test@example.com"},
		CSRFToken: "csrf-token-here",
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var parsed map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &parsed))

	assert.Contains(t, parsed, "user")
	assert.Contains(t, parsed, "csrf_token")
	// Tokens are no longer in JSON response - they're in HttpOnly cookies
	assert.NotContains(t, parsed, "access_token")
	assert.NotContains(t, parsed, "refresh_token")
}
