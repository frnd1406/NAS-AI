# CSRF Endpoint - Dokumentation

**Status:** ✅ Implementiert und deployed
**URL:** `GET /auth/csrf`
**Auth Required:** ✅ Yes (Bearer Token)

---

## Endpoint Details

### Request

```http
GET /auth/csrf HTTP/1.1
Host: your-domain.com
Authorization: Bearer <JWT_ACCESS_TOKEN>
```

### Response (Success - 200 OK)

```json
{
  "csrf_token": "xK9mF2nP8qR5sT7vW3yZ1bC4dE6gH8jL0mN..."
}
```

### Response (Unauthorized - 401)

```json
{
  "error": {
    "code": "unauthorized",
    "message": "Missing authorization token",
    "request_id": "87583e93-ffb3-414e-8762-7a5cf931a50f"
  }
}
```

### Response (Server Error - 500)

```json
{
  "error": {
    "code": "csrf_generation_failed",
    "message": "Failed to generate CSRF token",
    "request_id": "..."
  }
}
```

---

## Implementation

### Handler Location
**File:** `src/handlers/csrf.go`

```go
func GetCSRFToken(redis *database.RedisClient, logger *logrus.Logger) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Get user_id from auth middleware
        userID, exists := c.Get("user_id")
        if !exists {
            c.JSON(401, gin.H{"error": ...})
            return
        }

        // Generate CSRF token (32 bytes, base64)
        token, err := middleware.GenerateCSRFToken(redis, userID.(string))
        if err != nil {
            c.JSON(500, gin.H{"error": ...})
            return
        }

        c.JSON(200, gin.H{"csrf_token": token})
    }
}
```

### Route Configuration
**File:** `src/main.go:159-162`

```go
// CSRF token endpoint (requires auth)
authGroup.GET("/csrf",
    middleware.AuthMiddleware(jwtService, redis, logger), // Require auth
    handlers.GetCSRFToken(redis, logger),
)
```

---

## Token Storage

### Redis Key Format
```
Key: csrf:<user_id>
Value: <base64_csrf_token>
TTL: 24 hours
```

### Example
```
Key: csrf:user_123abc
Value: xK9mF2nP8qR5sT7vW3yZ1bC4dE6gH8jL0mN2pQ4rS6tU8vX0yA1bD3eF5gI7jK9lM
TTL: 86400 seconds (24h)
```

---

## Usage Flow

### 1. User Login
```bash
curl -X POST https://your-domain.com/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"SecurePass123"}'
```

**Response:**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {...}
}
```

### 2. Get CSRF Token
```bash
curl -X GET https://your-domain.com/auth/csrf \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..."
```

**Response:**
```json
{
  "csrf_token": "xK9mF2nP8qR5sT7vW3yZ1bC4dE6gH8jL0mN..."
}
```

### 3. Use CSRF Token for Protected Requests
```bash
curl -X GET https://your-domain.com/api/profile \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..." \
  -H "X-CSRF-Token: xK9mF2nP8qR5sT7vW3yZ1bC4dE6gH8jL0mN..."
```

---

## CSRF Middleware

### Validation Logic
**File:** `src/middleware/csrf.go:16-84`

**Rules:**
1. Only validates for: POST, PUT, DELETE, PATCH
2. GET, HEAD, OPTIONS requests bypass CSRF check
3. Requires `X-CSRF-Token` header
4. Requires authenticated user (user_id in context)
5. Token must match stored token in Redis

### Protected Routes
All routes under `/api/*` require CSRF token:
```go
apiGroup := r.Group("/api")
apiGroup.Use(middleware.AuthMiddleware(...))  // JWT validation
apiGroup.Use(middleware.CSRFMiddleware(...))  // CSRF validation
```

---

## Security Features

### Token Generation
- **Length:** 32 bytes (256 bits)
- **Encoding:** Base64 URL-safe
- **Randomness:** crypto/rand (cryptographically secure)
- **Uniqueness:** Per-user token (not per-session)

### Token Validation
- ✅ Stored in Redis (server-side)
- ✅ TTL: 24 hours auto-expiration
- ✅ User-specific (tied to user_id)
- ✅ Single token per user (old tokens overwritten)

### Attack Prevention
- ✅ **CSRF Attacks:** Token required for state-changing operations
- ✅ **Token Theft:** Requires both JWT + CSRF token
- ✅ **Replay Attacks:** TTL limits token lifetime
- ✅ **Brute Force:** 32-byte random token (2^256 possibilities)

---

## Testing

### Local Test (without auth - should fail)
```bash
curl -s http://localhost:8080/auth/csrf
# Response: {"error":{"code":"unauthorized",...}}
```

### Production Test (without auth - should fail)
```bash
curl -s https://your-domain.com/auth/csrf
# Response: {"error":{"code":"unauthorized",...}}
```

### Full Flow Test
```bash
# 1. Login
LOGIN_RESPONSE=$(curl -s -X POST https://your-domain.com/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"SecurePass123"}')

# 2. Extract access token
ACCESS_TOKEN=$(echo $LOGIN_RESPONSE | jq -r '.access_token')

# 3. Get CSRF token
CSRF_RESPONSE=$(curl -s -X GET https://your-domain.com/auth/csrf \
  -H "Authorization: Bearer $ACCESS_TOKEN")

# 4. Extract CSRF token
CSRF_TOKEN=$(echo $CSRF_RESPONSE | jq -r '.csrf_token')

# 5. Use both tokens
curl -X GET https://your-domain.com/api/profile \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "X-CSRF-Token: $CSRF_TOKEN"
```

---

## Response Formats

### Expected Response Format
```typescript
interface CSRFResponse {
  csrf_token: string;  // Base64 encoded token
}
```

### Error Response Format
```typescript
interface ErrorResponse {
  error: {
    code: string;           // Error code
    message: string;        // Human-readable message
    request_id: string;     // For debugging/logging
  }
}
```

---

## Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `unauthorized` | 401 | Missing or invalid JWT token |
| `csrf_generation_failed` | 500 | Failed to generate token (Redis error) |
| `csrf_token_missing` | 403 | CSRF header missing (on protected routes) |
| `csrf_validation_failed` | 403 | CSRF token invalid or expired |

---

## Deployment Status

- ✅ Handler implemented: `src/handlers/csrf.go`
- ✅ Route registered: `GET /auth/csrf`
- ✅ Auth middleware applied
- ✅ Tested locally
- ✅ Deployed to production: https://your-domain.com/auth/csrf
- ✅ Redis integration working
- ✅ Documentation complete

---

## Notes for Frontend

### When to Request CSRF Token
```javascript
// After successful login
const { access_token } = await login(email, password);

// Get CSRF token
const { csrf_token } = await fetch('/auth/csrf', {
  headers: {
    'Authorization': `Bearer ${access_token}`
  }
}).then(r => r.json());

// Store both tokens
localStorage.setItem('access_token', access_token);
localStorage.setItem('csrf_token', csrf_token);
```

### When to Include CSRF Token
```javascript
// All state-changing requests to /api/* endpoints
fetch('/api/profile', {
  method: 'PUT',
  headers: {
    'Authorization': `Bearer ${access_token}`,
    'X-CSRF-Token': csrf_token,
    'Content-Type': 'application/json'
  },
  body: JSON.stringify(data)
})
```

### When CSRF is NOT Required
- ✅ `/auth/login` - No CSRF needed
- ✅ `/auth/register` - No CSRF needed
- ✅ `/health` - No CSRF needed
- ✅ Any GET request - CSRF middleware skips GET

---

**Implemented:** 2025-11-22
**Status:** Production Ready ✅
