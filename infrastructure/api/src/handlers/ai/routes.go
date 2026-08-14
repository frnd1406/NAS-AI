package ai

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nas-ai/api/src/config"
	"github.com/nas-ai/api/src/database"
	auth_repo "github.com/nas-ai/api/src/repository/auth"
	"github.com/nas-ai/api/src/services/intelligence"
	"github.com/nas-ai/api/src/services/operations"
	"github.com/nas-ai/api/src/services/security"
	"github.com/sirupsen/logrus"
)

// Handler holds dependencies for AI handlers
type Handler struct {
	db             *database.DB
	cfg            *config.Config
	aiHTTPClient   *http.Client
	jobService     *operations.JobService
	secureAIFeeder *intelligence.SecureAIFeeder
	userRepo       auth_repo.UserRepositoryInterface
	jwtService     *security.JWTService
	tokenService   *security.TokenService
	redis          *database.RedisClient
	logger         *logrus.Logger
}

// NewHandler creates a new AI Handler
func NewHandler(
	db *database.DB,
	cfg *config.Config,
	aiHTTPClient *http.Client,
	jobService *operations.JobService,
	secureAIFeeder *intelligence.SecureAIFeeder,
	userRepo auth_repo.UserRepositoryInterface,
	jwtService *security.JWTService,
	tokenService *security.TokenService,
	redis *database.RedisClient,
	logger *logrus.Logger,
) *Handler {
	return &Handler{
		db:             db,
		cfg:            cfg,
		aiHTTPClient:   aiHTTPClient,
		jobService:     jobService,
		secureAIFeeder: secureAIFeeder,
		userRepo:       userRepo,
		jwtService:     jwtService,
		tokenService:   tokenService,
		redis:          redis,
		logger:         logger,
	}
}

// RegisterV1Routes registers API v1 AI routes.
// Public search/query/ask and /ai/* UI endpoints are DISABLED (2026-08-14):
// handler files and SecureAIFeeder remain. To re-enable, restore the previous
// ag.GET/POST registrations from git history (commit before this change).
func (h *Handler) RegisterV1Routes(rg *gin.RouterGroup) {
	_ = rg
	h.logger.Info("AI public routes disabled (search/query/ask/ai/*); handlers retained")
}

// RegisterAdminRoutes registers admin AI routes.
// Reconcile-knowledge is disabled while AI UI is off; handler file kept.
func (h *Handler) RegisterAdminRoutes(rg *gin.RouterGroup) {
	_ = rg
	h.logger.Info("AI admin routes disabled (reconcile-knowledge); handler retained")
}
