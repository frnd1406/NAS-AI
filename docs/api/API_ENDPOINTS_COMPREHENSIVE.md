# NAS.AI Comprehensive API Endpoints Documentation

**Generated:** 2025-12-01
**System:** Agent Orchestration Platform with AI Knowledge Integration

---

## Table of Contents
1. [Public Endpoints](#public-endpoints)
2. [Authentication Endpoints](#authentication-endpoints)
3. [Protected API Endpoints](#protected-api-endpoints)
4. [AI/ML Endpoints](#aiml-endpoints)
5. [System Monitoring & Metrics](#system-monitoring--metrics)
6. [Storage & File Management](#storage--file-management)
7. [Backup Management](#backup-management)
8. [Orchestrator Endpoints](#orchestrator-endpoints)
9. [Service Specifications](#service-specifications)

---

## Public Endpoints

### Health Check
- **Method:** `GET`
- **Path:** `/health`
- **Description:** Health status check for API and connected services
- **Authentication:** None
- **Response:** 200 OK
  ```json
  {
    "status": "healthy",
    "postgres": "connected",
    "redis": "connected"
  }
  ```
- **Service:** API (Gin)
- **Port:** 8080

### Swagger Documentation
- **Method:** `GET`
- **Path:** `/swagger/*any`
- **Description:** Interactive API documentation (development only)
- **Authentication:** None
- **Available in:** Development environment only
- **Service:** API (Gin)

---

## Authentication Endpoints

### User Registration
- **Method:** `POST`
- **Path:** `/auth/register`
- **Description:** Register a new user account
- **Authentication:** None
- **Rate Limit:** 5 requests/minute
- **Request Body:**
  ```json
  {
    "email": "user@example.com",
    "password": "secure-password"
  }
  ```
- **Response:** 201 Created
  ```json
  {
    "message": "User registered successfully",
    "user_id": "uuid"
  }
  ```
- **Service:** API (Gin)

### User Login
- **Method:** `POST`
- **Path:** `/auth/login`
- **Description:** Authenticate user and receive JWT token
- **Authentication:** None
- **Rate Limit:** 5 requests/minute
- **Request Body:**
  ```json
  {
    "email": "user@example.com",
    "password": "password"
  }
  ```
- **Response:** 200 OK
  ```json
  {
    "token": "jwt-token-here",
    "user": {
      "id": "uuid",
      "email": "user@example.com"
    }
  }
  ```
- **Service:** API (Gin)

### Token Refresh
- **Method:** `POST`
- **Path:** `/auth/refresh`
- **Description:** Refresh JWT token
- **Authentication:** Bearer JWT Token
- **Response:** 200 OK
  ```json
  {
    "token": "new-jwt-token-here"
  }
  ```
- **Service:** API (Gin)

### User Logout
- **Method:** `POST`
- **Path:** `/auth/logout`
- **Description:** Invalidate current session
- **Authentication:** Bearer JWT Token (Required)
- **Response:** 200 OK
  ```json
  {
    "message": "Logged out successfully"
  }
  ```
- **Service:** API (Gin)

### Email Verification
- **Method:** `POST`
- **Path:** `/auth/verify-email`
- **Description:** Verify user email address
- **Authentication:** None
- **Request Body:**
  ```json
  {
    "email": "user@example.com",
    "token": "verification-token"
  }
  ```
- **Response:** 200 OK
- **Service:** API (Gin)

### Resend Email Verification
- **Method:** `POST`
- **Path:** `/auth/resend-verification`
- **Description:** Resend email verification link
- **Authentication:** Bearer JWT Token (Required)
- **Response:** 200 OK
  ```json
  {
    "message": "Verification email sent"
  }
  ```
- **Service:** API (Gin)

### Forgot Password
- **Method:** `POST`
- **Path:** `/auth/forgot-password`
- **Description:** Request password reset token
- **Authentication:** None
- **Request Body:**
  ```json
  {
    "email": "user@example.com"
  }
  ```
- **Response:** 200 OK
  ```json
  {
    "message": "Password reset email sent"
  }
  ```
- **Service:** API (Gin)

### Reset Password
- **Method:** `POST`
- **Path:** `/auth/reset-password`
- **Description:** Reset user password with token
- **Authentication:** None
- **Request Body:**
  ```json
  {
    "token": "reset-token",
    "new_password": "new-secure-password"
  }
  ```
- **Response:** 200 OK
  ```json
  {
    "message": "Password reset successfully"
  }
  ```
- **Service:** API (Gin)

### Get CSRF Token
- **Method:** `GET`
- **Path:** `/api/v1/auth/csrf`
- **Description:** Retrieve CSRF token for state-changing operations
- **Authentication:** Bearer JWT Token
- **Response:** 200 OK
  ```json
  {
    "token": "csrf-token-here"
  }
  ```
- **Service:** API (Gin)

---

## Protected API Endpoints

### Get User Profile
- **Method:** `GET`
- **Path:** `/api/profile`
- **Description:** Retrieve authenticated user profile
- **Authentication:** Bearer JWT Token + CSRF Token (Required)
- **Response:** 200 OK
  ```json
  {
    "id": "uuid",
    "email": "user@example.com",
    "created_at": "2025-11-27T12:00:00Z"
  }
  ```
- **Service:** API (Gin)

### Get Monitoring Data
- **Method:** `GET`
- **Path:** `/api/monitoring`
- **Description:** Retrieve monitoring information
- **Authentication:** Bearer JWT Token + CSRF Token (Required)
- **Response:** 200 OK
  ```json
  {
    "items": [
      {
        "id": "uuid",
        "status": "healthy",
        "timestamp": "2025-11-27T12:00:00Z"
      }
    ]
  }
  ```
- **Service:** API (Gin)

---

## AI/ML Endpoints

### AI Knowledge Agent - Health Check
- **Method:** `GET`
- **Path:** `/health`
- **Description:** Health status of AI knowledge agent and model loading
- **Authentication:** None
- **Response:** 200 OK (when ready) or 503 Service Unavailable (loading)
  ```json
  {
    "status": "ok",
    "model_loaded": true,
    "db_ok": true
  }
  ```
- **Service:** ai-knowledge-agent (Flask/Python)
- **Port:** 5000
- **Model:** sentence-transformers/all-MiniLM-L6-v2

### Process File for Embeddings
- **Method:** `POST`
- **Path:** `/process`
- **Description:** Generate vector embeddings for uploaded file
- **Authentication:** Internal (AI Service)
- **Request Body:**
  ```json
  {
    "file_path": "/mnt/data/document.txt",
    "file_id": "document.txt",
    "mime_type": "text/plain"
  }
  ```
- **Response:** 200 OK
  ```json
  {
    "status": "success",
    "file_id": "document.txt",
    "content_length": 5000,
    "embedding_dim": 384
  }
  ```
- **Response:** 503 Service Unavailable (model not loaded)
- **Service:** ai-knowledge-agent (Flask/Python)
- **Port:** 5000
- **Database:** PostgreSQL with pgvector extension
- **Embedding Storage:** `file_embeddings` table

### Generate Query Embedding
- **Method:** `POST`
- **Path:** `/embed_query`
- **Description:** Generate embedding for search query
- **Authentication:** Internal (API Service)
- **Request Body:**
  ```json
  {
    "text": "Search query text"
  }
  ```
- **Response:** 200 OK
  ```json
  {
    "embedding": [0.123, 0.456, ...]
  }
  ```
- **Response:** 503 Service Unavailable (model not loaded)
- **Service:** ai-knowledge-agent (Flask/Python)
- **Port:** 5000

### Semantic Search
- **Method:** `GET`
- **Path:** `/api/v1/search`
- **Description:** Perform semantic search using AI embeddings
- **Authentication:** None (but rate-limited)
- **Query Parameters:**
  - `q` (required): Search query string
- **Response:** 200 OK
  ```json
  {
    "query": "search term",
    "results": [
      {
        "file_path": "/path/to/file.txt",
        "content": "file content excerpt",
        "similarity": 0.95
      }
    ]
  }
  ```
- **Service:** API (Gin) - calls AI Knowledge Agent internally
- **Integration:** Calls `/embed_query` on ai-knowledge-agent, then performs vector similarity search in PostgreSQL

---

## System Monitoring & Metrics

### Submit System Metrics
- **Method:** `POST`
- **Path:** `/api/v1/system/metrics`
- **Description:** Submit system metrics (CPU, RAM, Disk)
- **Authentication:** X-Monitoring-Token header
- **Request Body:**
  ```json
  {
    "agent_id": "monitoring-agent",
    "cpu_usage": 45.2,
    "ram_usage": 62.8,
    "disk_usage": 38.5
  }
  ```
- **Response:** 200 OK
- **Service:** API (Gin)
- **Source:** Monitoring Agent sends metrics periodically

### List System Metrics
- **Method:** `GET`
- **Path:** `/api/v1/system/metrics`
- **Description:** Retrieve historical system metrics
- **Authentication:** None
- **Query Parameters:**
  - `limit` (optional): Number of entries to return
- **Response:** 200 OK
  ```json
  {
    "items": [
      {
        "id": "uuid",
        "cpu_usage": 45.2,
        "ram_usage": 62.8,
        "disk_usage": 38.5,
        "timestamp": "2025-11-27T16:00:00Z"
      }
    ]
  }
  ```
- **Service:** API (Gin)
- **Storage:** PostgreSQL `system_metrics` table

### List System Alerts
- **Method:** `GET`
- **Path:** `/api/v1/system/alerts`
- **Description:** Get system alerts and warnings
- **Authentication:** None
- **Response:** 200 OK
  ```json
  {
    "items": [
      {
        "id": "uuid",
        "severity": "warning",
        "message": "High CPU usage detected",
        "resolved": false,
        "created_at": "2025-11-27T16:00:00Z"
      }
    ]
  }
  ```
- **Service:** API (Gin)
- **Storage:** PostgreSQL `system_alerts` table

### Create System Alert
- **Method:** `POST`
- **Path:** `/api/v1/system/alerts`
- **Description:** Create a new system alert
- **Authentication:** None
- **Request Body:**
  ```json
  {
    "severity": "warning|critical",
    "message": "Alert message"
  }
  ```
- **Response:** 201 Created
- **Service:** API (Gin)

### Resolve System Alert
- **Method:** `POST`
- **Path:** `/api/v1/system/alerts/:id/resolve`
- **Description:** Mark alert as resolved
- **Authentication:** None
- **Response:** 200 OK
- **Service:** API (Gin)

### Ingest Monitoring Data
- **Method:** `POST`
- **Path:** `/monitoring/ingest`
- **Description:** Public endpoint for monitoring agents to submit data
- **Authentication:** Monitoring Token in header
- **Service:** API (Gin)

---

## Storage & File Management

### List Files
- **Method:** `GET`
- **Path:** `/api/v1/storage/files`
- **Description:** List files in a directory
- **Authentication:** Bearer JWT Token + CSRF Token (Required)
- **Query Parameters:**
  - `path` (required): Directory path
- **Response:** 200 OK
  ```json
  {
    "items": [
      {
        "name": "documents",
        "isDir": true,
        "size": 0,
        "modTime": "2025-11-27T12:00:00Z"
      }
    ]
  }
  ```
- **Service:** API (Gin)
- **Storage:** `/mnt/data`

### Upload File
- **Method:** `POST`
- **Path:** `/api/v1/storage/upload`
- **Description:** Upload a file to storage
- **Authentication:** Bearer JWT Token + CSRF Token (Required)
- **Content-Type:** multipart/form-data
- **Form Parameters:**
  - `file`: File to upload
  - `path`: Target directory path
- **Response:** 200 OK
  ```json
  {
    "message": "File uploaded successfully",
    "path": "/file.txt"
  }
  ```
- **Service:** API (Gin)
- **Post-Processing:** Notifies AI Knowledge Agent for embedding generation
- **Storage:** `/mnt/data`

### Download File
- **Method:** `GET`
- **Path:** `/api/v1/storage/download`
- **Description:** Download a file from storage
- **Authentication:** Bearer JWT Token + CSRF Token (Required)
- **Query Parameters:**
  - `path` (required): File path
- **Response:** 200 OK (file content)
- **Service:** API (Gin)
- **Storage:** `/mnt/data`

### Rename File
- **Method:** `POST`
- **Path:** `/api/v1/storage/rename`
- **Description:** Rename a file or directory
- **Authentication:** Bearer JWT Token + CSRF Token (Required)
- **Request Body:**
  ```json
  {
    "oldPath": "/photo.jpg",
    "newPath": "/vacation.jpg"
  }
  ```
- **Response:** 200 OK
- **Service:** API (Gin)

### Delete File (to Trash)
- **Method:** `DELETE`
- **Path:** `/api/v1/storage/delete`
- **Description:** Move file to trash
- **Authentication:** Bearer JWT Token + CSRF Token (Required)
- **Query Parameters:**
  - `path` (required): File path
- **Response:** 200 OK
  ```json
  {
    "message": "File moved to trash"
  }
  ```
- **Service:** API (Gin)

### List Trash
- **Method:** `GET`
- **Path:** `/api/v1/storage/trash`
- **Description:** List deleted files in trash
- **Authentication:** Bearer JWT Token + CSRF Token (Required)
- **Response:** 200 OK
  ```json
  {
    "items": [
      {
        "id": "uuid",
        "name": "old-file.txt",
        "originalPath": "/old-file.txt",
        "deletedAt": "2025-11-27T15:00:00Z"
      }
    ]
  }
  ```
- **Service:** API (Gin)

### Restore from Trash
- **Method:** `POST`
- **Path:** `/api/v1/storage/trash/restore/:id`
- **Description:** Restore file from trash
- **Authentication:** Bearer JWT Token + CSRF Token (Required)
- **Response:** 200 OK
  ```json
  {
    "message": "File restored successfully"
  }
  ```
- **Service:** API (Gin)

### Delete from Trash (Permanent)
- **Method:** `DELETE`
- **Path:** `/api/v1/storage/trash/:id`
- **Description:** Permanently delete file from trash
- **Authentication:** Bearer JWT Token + CSRF Token (Required)
- **Response:** 200 OK
  ```json
  {
    "message": "File permanently deleted"
  }
  ```
- **Service:** API (Gin)

---

## Backup Management

### List Backups
- **Method:** `GET`
- **Path:** `/api/v1/backups`
- **Description:** List all available backups
- **Authentication:** Bearer JWT Token + CSRF Token (Required)
- **Response:** 200 OK
  ```json
  {
    "items": [
      {
        "id": "backup-20251127T030000Z.tar.gz",
        "name": "backup-20251127T030000Z.tar.gz",
        "size": 1024000,
        "created_at": "2025-11-27T03:00:00Z"
      }
    ]
  }
  ```
- **Service:** API (Gin)
- **Storage:** `/mnt/backups`

### Create Backup
- **Method:** `POST`
- **Path:** `/api/v1/backups`
- **Description:** Create a new backup
- **Authentication:** Bearer JWT Token + CSRF Token (Required)
- **Response:** 201 Created
  ```json
  {
    "message": "Backup created successfully",
    "backup_id": "backup-20251127T160000Z.tar.gz"
  }
  ```
- **Service:** API (Gin)

### Restore Backup
- **Method:** `POST`
- **Path:** `/api/v1/backups/:id/restore`
- **Description:** Restore a backup
- **Authentication:** Bearer JWT Token + CSRF Token (Required)
- **Authorization:** Admin only (destructive operation)
- **Response:** 200 OK
  ```json
  {
    "message": "Backup restored successfully"
  }
  ```
- **Service:** API (Gin)

### Delete Backup
- **Method:** `DELETE`
- **Path:** `/api/v1/backups/:id`
- **Description:** Delete a backup
- **Authentication:** Bearer JWT Token + CSRF Token (Required)
- **Authorization:** Admin only
- **Response:** 200 OK
  ```json
  {
    "message": "Backup deleted successfully"
  }
  ```
- **Service:** API (Gin)

---

## System Settings & Configuration

### Get System Settings
- **Method:** `GET`
- **Path:** `/api/v1/system/settings`
- **Description:** Get system configuration and backup settings
- **Authentication:** Bearer JWT Token + CSRF Token (Required)
- **Response:** 200 OK
  ```json
  {
    "backup": {
      "schedule": "0 3 * * *",
      "retention": 7,
      "path": "/mnt/backups"
    }
  }
  ```
- **Service:** API (Gin)

### Update Backup Settings
- **Method:** `PUT`
- **Path:** `/api/v1/system/settings/backup`
- **Description:** Update backup configuration
- **Authentication:** Bearer JWT Token + CSRF Token (Required)
- **Request Body:**
  ```json
  {
    "schedule": "0 2 * * *",
    "retention": 14,
    "path": "/mnt/backups"
  }
  ```
- **Response:** 200 OK
- **Service:** API (Gin)

### Validate Path
- **Method:** `POST`
- **Path:** `/api/v1/system/validate-path`
- **Description:** Validate backup path accessibility
- **Authentication:** Bearer JWT Token + CSRF Token (Required)
- **Request Body:**
  ```json
  {
    "path": "/mnt/backups"
  }
  ```
- **Response:** 200 OK
- **Service:** API (Gin)

---

## Orchestrator Endpoints

### Orchestrator Health
- **Method:** `GET`
- **Path:** `/health`
- **Description:** Orchestrator service health status
- **Authentication:** None
- **Response:** 200 OK
  ```json
  {
    "status": "ok",
    "timestamp": "2025-11-27T16:00:00Z",
    "version": "1.0.0"
  }
  ```
- **Service:** Orchestrator (Go)
- **Port:** 9000

### List All Services
- **Method:** `GET`
- **Path:** `/api/services`
- **Description:** Get status of all registered services
- **Authentication:** None
- **Response:** 200 OK
  ```json
  [
    {
      "name": "nas-api",
      "url": "http://api:8080/health",
      "healthy": true,
      "last_check": "2025-11-27T16:00:00Z",
      "last_healthy": "2025-11-27T16:00:00Z",
      "consecutive_fails": 0,
      "total_checks": 100,
      "total_failures": 5,
      "uptime_percent": 95.0
    }
  ]
  ```
- **Service:** Orchestrator (Go)
- **Port:** 9000

### Service Registry
- **Method:** `GET`
- **Path:** `/api/registry`
- **Description:** Get service registry information
- **Authentication:** None
- **Response:** 200 OK
  ```json
  {
    "services": [
      {
        "name": "nas-api",
        "url": "http://api:8080/health",
        "tags": ["core", "api"],
        "metadata": {
          "type": "backend",
          "language": "go"
        }
      }
    ]
  }
  ```
- **Service:** Orchestrator (Go)
- **Port:** 9000

### Prometheus Metrics
- **Method:** `GET`
- **Path:** `/metrics`
- **Description:** Prometheus metrics endpoint
- **Authentication:** None
- **Response:** 200 OK (Prometheus text format)
- **Service:** Orchestrator (Go)
- **Port:** 9000
- **Format:** OpenMetrics/Prometheus text format

---

## Service Specifications

### Services Table

| Service | Language | Port | Container | Status | Key Endpoints |
|---------|----------|------|-----------|--------|---------------|
| **API** | Go (Gin) | 8080 | nas-api | Running | `/health`, `/auth/*`, `/api/v1/*` |
| **AI Knowledge Agent** | Python (Flask) | 5000 | nas-ai-knowledge-agent | Running | `/health`, `/process`, `/embed_query` |
| **Orchestrator** | Go | 9000 | orchestrator | Running | `/health`, `/api/services`, `/api/registry` |
| **Monitoring Agent** | Go | - | nas-monitoring | Running | Pushes to API `/api/v1/system/metrics` |
| **Analysis Agent** | Go | - | nas-analysis-agent | Running | Reads metrics, creates alerts |
| **Pentester Agent** | - | - | nas-pentester-agent | Running | Security testing |
| **PostgreSQL** | - | 5432 | nas-api-postgres | Running | Database backend |
| **Redis** | - | 6379 | nas-api-redis | Running | Token/cache store |
| **WebUI** | Node.js/Vite | 80/3000 | nas-webui | Running | React frontend |

### Database Schema

**Key Tables:**
- `system_metrics` - CPU, RAM, Disk metrics over time
- `system_alerts` - System warnings and critical alerts
- `system_settings` - Configuration storage
- `users` - User accounts
- `file_embeddings` - AI embeddings for files (pgvector)
- `monitoring` - Monitoring data from agents

### Authentication

- **JWT Tokens:** Issued by `/auth/login`, validated by Bearer header
- **CSRF Tokens:** Issued by `/api/v1/auth/csrf`, required for state-changing operations
- **Monitoring Token:** X-Monitoring-Token header for metrics submission
- **Token Storage:** Redis (TTL-based expiration)

### Security Headers

- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `Strict-Transport-Security: max-age=31536000`
- `Content-Security-Policy: default-src 'self'`

### Rate Limiting

- **Standard Endpoints:** 100 requests/minute
- **Auth Endpoints:** 5 requests/minute
- **Search Endpoint:** Standard rate limit

### CORS Configuration

- Configured via `CORS_ORIGINS` environment variable
- Preflight requests (OPTIONS) allowed globally
- Authorization headers required for protected endpoints

---

## Future Endpoints (Planned)

Based on NAS_AI_SYSTEM.md documentation:

### User Management
- `GET/POST/PUT/DELETE /api/v1/users` - User CRUD operations
- `POST /api/v1/users/:id/reset-password` - Admin password reset
- `GET /auth/sessions` - User session management
- `POST /auth/sessions/:id/revoke` - Session revocation

### Files & Favorites
- `GET /api/v1/files/:id/thumbnail` - File thumbnails
- `POST /api/v1/files/zip` - Create zip archive
- `POST /api/v1/files/unzip` - Extract archive
- `GET/POST/DELETE /api/v1/favorites` - Favorite files
- `GET /api/v1/storage/overview` - Storage overview
- `GET /api/v1/storage/usage` - Detailed usage stats
- `GET /api/v1/storage/directory-size` - Directory size calculation
- `GET /api/v1/storage/alerts` - Storage alerts
- `GET /api/v1/storage/trends` - Storage usage trends

### Shares (SMB/NFS/FTP)
- `GET/POST/PUT/DELETE /api/v1/shares` - Share management
- Share configuration with type enum (smb/nfs/ftp)
- Access control lists

### AI Services
- `GET /api/v1/ai/search` - Enhanced AI search with facets
- `POST /api/v1/ai/documents/:id/analyze` - Document analysis
- `GET /api/v1/ai/suggestions` - AI recommendations

### Documentation Terminal
- `GET/POST /api/v1/docs-terminal/*` - Command execution
- `GET/PUT /api/v1/docs-settings` - Terminal settings

### Settings & Security
- `GET /api/v1/settings/security` - Security configuration
- `PUT /api/v1/settings/security` - Update security settings
- `GET /api/v1/auth/audit` - Audit logs
- Support for passkey authentication

---

## Error Handling

All endpoints follow standard HTTP status codes:

| Code | Meaning |
|------|---------|
| 200 | OK - Request successful |
| 201 | Created - Resource created |
| 204 | No Content - Success, no response body |
| 400 | Bad Request - Invalid input |
| 401 | Unauthorized - Authentication required |
| 403 | Forbidden - Insufficient permissions |
| 404 | Not Found - Resource not found |
| 429 | Too Many Requests - Rate limit exceeded |
| 500 | Internal Server Error - Server error |
| 503 | Service Unavailable - Service down (e.g., model loading) |

### Error Response Format

```json
{
  "error": "Description of the error"
}
```

---

## WebSocket Topics (Real-time Events)

| Topic | Publisher | Payload | Consumer |
|-------|-----------|---------|----------|
| `files:progress` | FileService | `{path, op, percent, user_id}` | File widgets |
| `files:favorites` | FavoritesService | `{favorite_id, action}` | Favorites page |
| `backups:jobs` | BackupAgent | `{job_id, status, eta}` | Backup timeline |
| `storage:alerts` | Orchestrator | `{level, message, action}` | Storage alerts |
| `security:sessions` | AuthService | `{user_id, device_id, action}` | Settings page |
| `ai:search` | AIKnowledgeAgent | `{query_id, status, facets}` | AI Lens |
| `docs:terminal` | DocumentationAgent | `stream` lines | Terminal UI |

---

## Development & Testing

### Available in Development
- Swagger documentation at `/swagger/index.html`
- Enhanced logging with JSON format
- CORS relaxed for local testing

### Testing Endpoints
- Contract tests in: `/infrastructure/api/test/integration/`
- Example tests: `auth_test.go`, `health_test.go`

---

## Configuration

### Environment Variables (from docker-compose.prod.yml)

```
PORT=8080
JWT_SECRET=<secret-key>
DATABASE_URL=postgres://nas_user:password@postgres:5432/nas_db
REDIS_URL=redis:6379
CORS_ORIGINS=<allowed-origins>
MONITORING_TOKEN=<token>
AI_SERVICE_URL=http://ai-knowledge-agent:5000
RATE_LIMIT_PER_MIN=100
```

---

**Document Version:** 1.0
**Last Updated:** 2025-12-01
