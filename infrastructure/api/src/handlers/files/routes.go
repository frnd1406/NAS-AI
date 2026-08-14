package files

import (
	"github.com/gin-gonic/gin"
	"github.com/nas-ai/api/src/database"
	"github.com/nas-ai/api/src/middleware/logic"
	"github.com/nas-ai/api/src/services/content"
	"github.com/nas-ai/api/src/services/intelligence"
	"github.com/nas-ai/api/src/services/security"
	"github.com/sirupsen/logrus"
)

// Handler holds dependencies for files handlers
type Handler struct {
	storageService          *content.StorageManager
	encryptionService       *security.EncryptionService
	encryptionPolicyService *security.EncryptionPolicyService
	honeyfileService        *content.HoneyfileService
	aiAgentService          *intelligence.AIAgentService
	archiveService          *content.ArchiveService
	contentDeliveryService  *content.ContentDeliveryService
	encryptedStorageService *content.EncryptedStorageService
	secureAIFeeder          *intelligence.SecureAIFeeder
	blobStorageHandler      *BlobStorageHandler
	jwtService              security.JWTServiceInterface
	tokenService            *security.TokenService
	redis                   *database.RedisClient
	logger                  *logrus.Logger
}

// NewHandler creates a new Files Handler
func NewHandler(
	storageService *content.StorageManager,
	encryptionService *security.EncryptionService,
	encryptionPolicyService *security.EncryptionPolicyService,
	honeyfileService *content.HoneyfileService,
	aiAgentService *intelligence.AIAgentService,
	archiveService *content.ArchiveService,
	contentDeliveryService *content.ContentDeliveryService,
	encryptedStorageService *content.EncryptedStorageService,
	secureAIFeeder *intelligence.SecureAIFeeder,
	blobStorageHandler *BlobStorageHandler,
	jwtService security.JWTServiceInterface,
	tokenService *security.TokenService,
	redis *database.RedisClient,
	logger *logrus.Logger,
) *Handler {
	return &Handler{
		storageService:          storageService,
		encryptionService:       encryptionService,
		encryptionPolicyService: encryptionPolicyService,
		honeyfileService:        honeyfileService,
		aiAgentService:          aiAgentService,
		archiveService:          archiveService,
		contentDeliveryService:  contentDeliveryService,
		encryptedStorageService: encryptedStorageService,
		secureAIFeeder:          secureAIFeeder,
		blobStorageHandler:      blobStorageHandler,
		jwtService:              jwtService,
		tokenService:            tokenService,
		redis:                   redis,
		logger:                  logger,
	}
}

// RegisterV1Routes registers API v1 files routes
func (h *Handler) RegisterV1Routes(rg *gin.RouterGroup) {
	// Public (or implicitly protected?)
	rg.GET("/files/content", FileContentHandler(h.storageService, h.logger))

	// Vault Public
	rg.GET("/vault/status", VaultStatusHandler(h.encryptionService))
	rg.GET("/vault/config", VaultConfigGetHandler(h.encryptionService))

	// Vault Uploads (Protected)
	vaultUpload := rg.Group("/vault/upload")
	vaultUpload.Use(
		logic.AuthMiddleware(h.jwtService, h.tokenService, h.redis, h.logger),
		logic.CSRFMiddleware(h.redis, h.logger),
	)
	{
		vaultUpload.POST("/init", h.blobStorageHandler.InitUpload)
		vaultUpload.POST("/chunk/:id", h.blobStorageHandler.UploadChunk)
		vaultUpload.POST("/finalize/:id", h.blobStorageHandler.FinalizeUpload)
	}

	// Storage (Protected)
	storage := rg.Group("/storage")
	storage.Use(
		logic.AuthMiddleware(h.jwtService, h.tokenService, h.redis, h.logger),
		logic.CSRFMiddleware(h.redis, h.logger),
	)
	{
		storage.GET("/files", StorageListHandler(h.storageService, h.logger))
		storage.POST("/upload", StorageUploadHandler(h.storageService, h.encryptionPolicyService, h.honeyfileService, h.aiAgentService, h.logger))
		storage.GET("/download", StorageDownloadHandler(h.storageService, h.honeyfileService, h.logger))
		storage.GET("/smart-download", SmartDownloadHandler(h.storageService, h.honeyfileService, h.contentDeliveryService, h.logger))
		storage.GET("/download-zip", StorageDownloadZipHandler(h.storageService, h.logger))
		storage.POST("/batch-download", StorageBatchDownloadHandler(h.storageService, h.logger))
		storage.DELETE("/delete", StorageDeleteHandler(h.storageService, h.aiAgentService, h.logger))
		storage.GET("/trash", StorageTrashListHandler(h.storageService, h.logger))
		storage.POST("/trash/restore/:id", StorageTrashRestoreHandler(h.storageService, h.logger))
		storage.DELETE("/trash/:id", StorageTrashDeleteHandler(h.storageService, h.logger))
		storage.POST("/trash/empty", StorageTrashEmptyHandler(h.storageService, h.logger))
		storage.POST("/rename", StorageRenameHandler(h.storageService, h.logger))
		storage.POST("/move", StorageMoveHandler(h.storageService, h.logger))
		storage.POST("/mkdir", StorageMkdirHandler(h.storageService, h.logger))
		storage.POST("/upload-zip", StorageUploadZipHandler(h.storageService, h.archiveService, h.logger))
	}

	// Encrypted Storage (Protected)
	if h.encryptedStorageService != nil {
		enc := rg.Group("/encrypted")
		enc.Use(
			logic.AuthMiddleware(h.jwtService, h.tokenService, h.redis, h.logger),
			logic.CSRFMiddleware(h.redis, h.logger),
		)
		{
			enc.GET("/status", EncryptedStorageStatusHandler(h.encryptedStorageService, h.encryptionService))
			enc.GET("/files", EncryptedStorageListHandler(h.encryptedStorageService, h.logger))
			enc.POST("/upload", EncryptedStorageUploadHandler(h.encryptedStorageService, h.secureAIFeeder, h.logger))
			enc.GET("/download", EncryptedStorageDownloadHandler(h.encryptedStorageService, h.logger))
			enc.GET("/preview", EncryptedStoragePreviewHandler(h.encryptedStorageService, h.logger))
			enc.DELETE("/delete", EncryptedStorageDeleteHandler(h.encryptedStorageService, h.logger))
		}
	}
}
