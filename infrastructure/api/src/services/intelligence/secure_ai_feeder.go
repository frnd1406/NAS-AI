package intelligence

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/nas-ai/api/src/domain/files"
	"github.com/nas-ai/api/src/pathsafe"

	"github.com/nas-ai/api/src/services/security"
	"github.com/sirupsen/logrus"
)

// ==============================================================================
// THE BLIND AGENT PROTOCOL
// ==============================================================================
//
// Security Architecture: The AI agent is "blind" to encrypted content.
//
// PRINCIPLE: The AI (Python-Agent) NEVER receives encrypted file content
// during automated indexing. Encrypted files are completely invisible to the AI
// knowledge base - they don't exist in the vector store.
//
// EXCEPTION: "Ephemeral Context" - During a LIVE user session, the user can
// explicitly request to include encrypted file content in a query. The content
// is decrypted in-memory, streamed to AI, and immediately discarded. It is
// NEVER persisted in the vector index.
//
// Data Flow:
//
//   Background Indexing (Cronjob):
//     files/*.txt → GetContentForIndexing() → AI Agent → Vector Store
//     files/*.enc → GetContentForIndexing() → ErrEncryptedContentProtected → SKIP
//
//   Live Query (User Session):
//     "Search geheim.txt.enc" + password → GetEphemeralContent() → DecryptStream
//     → RAM-only pipe → AI Agent (one-shot query) → Response → WIPED
//
// ==============================================================================

// Sentinel errors for Blind Agent Protocol
var (
	// ErrEncryptedContentProtected is returned when attempting to index encrypted content.
	// This is a security feature - encrypted files must NOT be sent to AI for indexing.
	ErrEncryptedContentProtected = errors.New("content is encrypted and cannot be indexed")

	// ErrPasswordRequired is returned when ephemeral access requires a password.
	ErrPasswordRequired = errors.New("password required for encrypted content access")

	// ErrFileNotFound is returned when the file doesn't exist in storage.
	ErrFileNotFound = errors.New("file not found")
)

// FileMetadataProvider is an interface for retrieving file metadata.
// This allows the SecureAIFeeder to be decoupled from the specific repository implementation.
type FileMetadataProvider interface {
	GetFileByID(fileID string) (*files.File, error)
	GetFileByPath(storagePath string) (*files.File, error)
}

// SecureAIFeeder handles secure content feeding to the AI agent.
// SECURITY: Implements the "Blind Agent Protocol" to protect encrypted content.
//
// The AI agent is "blind" to USER-encrypted files:
//   - Automated indexing SKIPS encrypted files (ErrEncryptedContentProtected)
//   - Live queries CAN access encrypted content with explicit user password
//   - Decrypted content exists only in RAM and is immediately wiped
type SecureAIFeeder struct {
	encryption   *security.EncryptionService
	fileProvider FileMetadataProvider // For looking up file metadata
	storagePath  string               // Base path for file storage (e.g., "/mnt/data")
	// extraRoots holds additional operator-configured roots that hold files this
	// feeder may read (e.g. the encrypted storage mount). Never user input.
	extraRoots     []string
	aiAgentURL     string
	internalSecret string
	httpClient     *http.Client
	logger         *logrus.Logger
}

// IngestDirectPayload is the JSON payload for the AI agent's /ingest_direct endpoint
type IngestDirectPayload struct {
	Content  string `json:"content"`
	FileID   string `json:"file_id"`
	FilePath string `json:"file_path"`
	MimeType string `json:"mime_type"`
}

// IngestDirectResponse is the response from the AI agent
type IngestDirectResponse struct {
	Status          string `json:"status"`
	FileID          string `json:"file_id"`
	FilePath        string `json:"file_path"`
	ContentLength   int    `json:"content_length"`
	EmbeddingDim    int    `json:"embedding_dim"`
	EncryptedSource bool   `json:"encrypted_source"`
	Error           string `json:"error,omitempty"`
}

// NewSecureAIFeeder creates a new secure AI feeder service.
// DEPRECATED: Use NewSecureAIFeederV2 for Blind Agent Protocol support.
func NewSecureAIFeeder(
	encryption *security.EncryptionService,
	aiAgentURL string,
	internalSecret string,
	logger *logrus.Logger,
) *SecureAIFeeder {
	return &SecureAIFeeder{
		encryption:     encryption,
		fileProvider:   nil, // Legacy mode - no file provider
		storagePath:    absOrClean(defaultStorageRoot),
		extraRoots:     defaultExtraRoots(),
		aiAgentURL:     aiAgentURL,
		internalSecret: internalSecret,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		logger: logger,
	}
}

// NewSecureAIFeederV2 creates a hardened AI feeder with Blind Agent Protocol.
// This version requires a FileMetadataProvider for encryption-aware file access.
func NewSecureAIFeederV2(
	encryption *security.EncryptionService,
	fileProvider FileMetadataProvider,
	storagePath string,
	aiAgentURL string,
	internalSecret string,
	logger *logrus.Logger,
) *SecureAIFeeder {
	return &SecureAIFeeder{
		encryption:     encryption,
		fileProvider:   fileProvider,
		storagePath:    absOrClean(storagePath),
		extraRoots:     defaultExtraRoots(),
		aiAgentURL:     aiAgentURL,
		internalSecret: internalSecret,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		logger: logger,
	}
}

const (
	// defaultStorageRoot mirrors the storage root used elsewhere in the API.
	defaultStorageRoot = "/mnt/data"
	// defaultEncryptedRoot mirrors the encrypted storage mount configured by the
	// server; it can be overridden via ENCRYPTED_STORAGE_PATH.
	defaultEncryptedRoot = "/media/user/DEMO"
)

func absOrClean(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		return filepath.Clean(p)
	}
	return abs
}

func defaultExtraRoots() []string {
	encRoot := os.Getenv("ENCRYPTED_STORAGE_PATH")
	if encRoot == "" {
		encRoot = defaultEncryptedRoot
	}
	return []string{absOrClean(encRoot)}
}

// AllowRoot registers an additional operator-configured directory the feeder may
// read from. Provided as a setter so the constructor signatures stay stable.
func (f *SecureAIFeeder) AllowRoot(root string) {
	if root == "" {
		return
	}
	f.extraRoots = append(f.extraRoots, absOrClean(root))
}

func (f *SecureAIFeeder) allowedRoots() []string {
	return append([]string{f.storagePath}, f.extraRoots...)
}

// resolvePath confines an untrusted storage path to the feeder's roots.
//
// Relative paths (the DB-stored storage_path of a file) are joined onto the
// storage root, so "../../etc/passwd" cannot climb out. Absolute paths are
// accepted only when they already live under one of the allowed roots.
func (f *SecureAIFeeder) resolvePath(p string) (string, error) {
	if filepath.IsAbs(p) {
		clean := filepath.Clean(p)
		if err := pathsafe.WithinAnyRoot(f.allowedRoots(), clean); err != nil {
			return "", err
		}
		return clean, nil
	}
	return pathsafe.SafeJoin(f.storagePath, p)
}

// ==============================================================================
// BLIND AGENT PROTOCOL - CORE METHODS
// ==============================================================================

// GetContentForIndexing returns a reader for file content for BACKGROUND INDEXING.
// This is called by the cronjob that builds the AI knowledge base.
//
// SECURITY: This method REFUSES to return encrypted content.
// Encrypted files are completely invisible to the AI during automated indexing.
//
// Returns:
//   - io.ReadCloser: File content stream (caller must close)
//   - error: ErrEncryptedContentProtected if file is USER-encrypted
func (f *SecureAIFeeder) GetContentForIndexing(fileID string) (io.ReadCloser, error) {
	if f.fileProvider == nil {
		return nil, errors.New("file provider not configured (use NewSecureAIFeederV2)")
	}

	// Step 1: Load file metadata from database
	file, err := f.fileProvider.GetFileByID(fileID)
	if err != nil {
		f.logger.WithFields(logrus.Fields{
			"fileID": fileID,
			"error":  err.Error(),
		}).Warn("🔒 BlindAgent: File metadata lookup failed")
		return nil, ErrFileNotFound
	}

	// Step 2: CRITICAL SECURITY CHECK - Refuse encrypted content
	if file.EncryptionStatus == files.EncryptionUser {
		f.logger.WithFields(logrus.Fields{
			"fileID":   fileID,
			"filename": file.Filename,
			"status":   file.EncryptionStatus,
		}).Warn("🔒 BlindAgent: BLOCKED - Encrypted file cannot be indexed")
		return nil, ErrEncryptedContentProtected
	}

	// Step 3: For NONE/SYSTEM mode, return normal file stream
	fullPath, err := f.resolvePath(file.StoragePath)
	if err != nil {
		return nil, err
	}
	fileHandle, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}

	f.logger.WithFields(logrus.Fields{
		"fileID":   fileID,
		"filename": file.Filename,
		"path":     fullPath,
	}).Debug("🔓 BlindAgent: File approved for indexing")

	return fileHandle, nil
}

// GetEphemeralContent returns a reader for file content for LIVE QUERIES.
// This enables on-demand access to encrypted files during user sessions.
//
// SECURITY: Decrypted content exists ONLY in RAM via an io.Pipe.
// The content is streamed to the AI and immediately discarded.
// It is NEVER persisted to disk or the vector index.
//
// Parameters:
//   - fileID: File identifier
//   - userPassword: Decryption password (required for USER-encrypted files)
//
// Returns:
//   - io.ReadCloser: Decrypted content stream (caller must close)
//   - error: ErrPasswordRequired if encrypted file accessed without password
func (f *SecureAIFeeder) GetEphemeralContent(fileID string, userPassword string) (io.ReadCloser, error) {
	if f.fileProvider == nil {
		return nil, errors.New("file provider not configured (use NewSecureAIFeederV2)")
	}

	// Step 1: Load file metadata
	file, err := f.fileProvider.GetFileByID(fileID)
	if err != nil {
		return nil, ErrFileNotFound
	}

	fullPath, err := f.resolvePath(file.StoragePath)
	if err != nil {
		return nil, err
	}

	// Step 2: Route based on encryption status
	switch file.EncryptionStatus {
	case files.EncryptionNone:
		// Unencrypted - direct file access
		return os.Open(fullPath)

	case files.EncryptionUser:
		// USER-encrypted - requires password to decrypt
		if userPassword == "" {
			f.logger.WithFields(logrus.Fields{
				"fileID":   fileID,
				"filename": file.Filename,
			}).Warn("🔒 BlindAgent: Ephemeral access denied - password required")
			return nil, ErrPasswordRequired
		}

		// Step 3: Create RAM-only decryption pipe
		return f.createDecryptionPipe(fullPath, userPassword, fileID)

	case files.EncryptionSystem:
		// SYSTEM-encrypted - use vault DEK (no user password needed)
		if f.encryption == nil || !f.encryption.IsUnlocked() {
			return nil, security.ErrVaultLocked
		}
		// For SYSTEM mode, use the vault's DEK to decrypt
		// This is a future feature - fall through to error for now
		return nil, errors.New("SYSTEM encryption not yet implemented for ephemeral access")

	default:
		return nil, fmt.Errorf("unknown encryption status: %s", file.EncryptionStatus)
	}
}

// createDecryptionPipe creates an io.Pipe for streaming decryption.
// The decryption runs in a goroutine and pipes plaintext directly to the reader.
// SECURITY: Plaintext exists only in the pipe's RAM buffer.
func (f *SecureAIFeeder) createDecryptionPipe(filePath, password, fileID string) (io.ReadCloser, error) {
	// Open the encrypted file
	encFile, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open encrypted file: %w", err)
	}

	// Create pipe - pipeReader for consumer, pipeWriter for producer
	pipeReader, pipeWriter := io.Pipe()

	// Start decryption goroutine
	go func() {
		defer encFile.Close()

		err := security.DecryptStream(password, encFile, pipeWriter)
		if err != nil {
			f.logger.WithFields(logrus.Fields{
				"fileID": fileID,
				"error":  err.Error(),
			}).Warn("🔒 BlindAgent: Ephemeral decryption failed")
			pipeWriter.CloseWithError(fmt.Errorf("decryption failed: %w", err))
			return
		}

		pipeWriter.Close()
		f.logger.WithField("fileID", fileID).Debug("🔓 BlindAgent: Ephemeral content streamed successfully")
	}()

	return pipeReader, nil
}

// GetContentForIndexingByPath is a convenience method that looks up file by path.
// Used when the file ID is not known but the storage path is.
func (f *SecureAIFeeder) GetContentForIndexingByPath(storagePath string) (io.ReadCloser, error) {
	if f.fileProvider == nil {
		// Fallback: Legacy mode - assume unencrypted
		fullPath, err := f.resolvePath(storagePath)
		if err != nil {
			return nil, err
		}
		return os.Open(fullPath)
	}

	file, err := f.fileProvider.GetFileByPath(storagePath)
	if err != nil {
		return nil, ErrFileNotFound
	}

	return f.GetContentForIndexing(file.ID)
}

// ==============================================================================
// LEGACY METHODS (kept for backward compatibility)
// ==============================================================================

// FeedEncryptedFile decrypts a file and pushes the content to the AI agent.
// DEPRECATED: This method uses SYSTEM encryption only. For USER encryption,
// use GetEphemeralContent instead.
func (f *SecureAIFeeder) FeedEncryptedFile(
	encryptedPath string,
	originalPath string,
	fileID string,
	mimeType string,
) error {
	// Check if encryption is available
	if f.encryption == nil || !f.encryption.IsUnlocked() {
		return security.ErrVaultLocked
	}

	f.logger.WithFields(logrus.Fields{
		"encryptedPath": encryptedPath,
		"originalPath":  originalPath,
		"fileID":        fileID,
	}).Info("SecureAIFeeder: Starting encrypted file ingestion")

	// Step 1: Read encrypted file from disk, confined to the allowed roots
	safePath, err := f.resolvePath(encryptedPath)
	if err != nil {
		return err
	}

	encryptedData, err := os.ReadFile(safePath)
	if err != nil {
		return fmt.Errorf("read encrypted file: %w", err)
	}

	// Step 2: Decrypt in-memory (uses SYSTEM key from vault)
	plaintext, err := f.encryption.DecryptData(encryptedData)
	if err != nil {
		return fmt.Errorf("decrypt file: %w", err)
	}

	// Ensure plaintext is wiped after we're done
	defer func() {
		for i := range plaintext {
			plaintext[i] = 0
		}
		f.logger.Debug("SecureAIFeeder: Plaintext securely wiped from memory")
	}()

	// Step 3: Send to AI agent
	return f.sendToAIAgent(string(plaintext), fileID, originalPath, mimeType)
}

// FeedContent pushes raw content (already decrypted) to the AI agent.
func (f *SecureAIFeeder) FeedContent(
	content string,
	fileID string,
	filePath string,
	mimeType string,
) error {
	if len(content) == 0 {
		return nil
	}
	return f.sendToAIAgent(content, fileID, filePath, mimeType)
}

// sendToAIAgent sends content to the AI agent for indexing
func (f *SecureAIFeeder) sendToAIAgent(content, fileID, filePath, mimeType string) error {
	payload := IngestDirectPayload{
		Content:  content,
		FileID:   fileID,
		FilePath: filePath,
		MimeType: mimeType,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	url := f.aiAgentURL + "/process"

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if f.internalSecret != "" {
		req.Header.Set("X-Internal-Secret", f.internalSecret)
	}

	resp, err := f.httpClient.Do(req)
	if err != nil {
		f.logger.WithFields(logrus.Fields{
			"url":    url,
			"fileID": fileID,
			"error":  err.Error(),
		}).Warn("SecureAIFeeder: Failed to contact AI agent")
		return fmt.Errorf("AI agent request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var result IngestDirectResponse
		if err := json.Unmarshal(body, &result); err == nil {
			f.logger.WithFields(logrus.Fields{
				"fileID":        fileID,
				"contentLength": result.ContentLength,
				"embeddingDim":  result.EmbeddingDim,
			}).Info("SecureAIFeeder: Content indexed successfully")
		}
		return nil
	}

	f.logger.WithFields(logrus.Fields{
		"fileID":     fileID,
		"statusCode": resp.StatusCode,
		"response":   string(body),
	}).Warn("SecureAIFeeder: AI agent returned error")

	return fmt.Errorf("AI agent error (status %d): %s", resp.StatusCode, string(body))
}

// RemoveDocument removes a document from the AI agent's vector index.
func (f *SecureAIFeeder) RemoveDocument(fileID string, filePath string) error {
	payload := map[string]string{
		"file_id":   fileID,
		"file_path": filePath,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	url := f.aiAgentURL + "/delete"

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if f.internalSecret != "" {
		req.Header.Set("X-Internal-Secret", f.internalSecret)
	}

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("AI agent unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		f.logger.WithFields(logrus.Fields{
			"fileID":   fileID,
			"filePath": filePath,
		}).Info("SecureAIFeeder: Document removed from index")
		return nil
	}

	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("AI agent error (status %d): %s", resp.StatusCode, string(body))
}

// ListVectorsResponse is the response from the AI agent's /list_vectors endpoint
type ListVectorsResponse struct {
	FileIDs []string `json:"file_ids"`
	Count   int      `json:"count"`
}

// ReconcileIndex identifies and removes orphaned embeddings.
func (f *SecureAIFeeder) ReconcileIndex(existingFileIDs map[string]bool) (int, error) {
	f.logger.Info("SecureAIFeeder: Starting index reconciliation (garbage collection)")

	// Step 1: Get all file IDs from vector store
	url := f.aiAgentURL + "/list_vectors"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("create request: %w", err)
	}
	if f.internalSecret != "" {
		req.Header.Set("X-Internal-Secret", f.internalSecret)
	}
	resp, err := f.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to contact AI agent: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("AI agent error (status %d): %s", resp.StatusCode, string(body))
	}

	var vectorResponse ListVectorsResponse
	if err := json.NewDecoder(resp.Body).Decode(&vectorResponse); err != nil {
		return 0, fmt.Errorf("decode response: %w", err)
	}

	f.logger.WithField("vectorCount", vectorResponse.Count).Info("SecureAIFeeder: Found vectors in index")

	// Step 2: Find zombies (vectors without corresponding files)
	var zombies []string
	for _, fileID := range vectorResponse.FileIDs {
		if !existingFileIDs[fileID] {
			zombies = append(zombies, fileID)
		}
	}

	if len(zombies) == 0 {
		f.logger.Info("SecureAIFeeder: No orphaned embeddings found, index is clean")
		return 0, nil
	}

	f.logger.WithField("zombieCount", len(zombies)).Warn("SecureAIFeeder: Found orphaned embeddings")

	// Step 3: Delete zombies
	deleted := 0
	for _, fileID := range zombies {
		if err := f.RemoveDocument(fileID, ""); err != nil {
			f.logger.WithFields(logrus.Fields{
				"fileID": fileID,
				"error":  err.Error(),
			}).Warn("SecureAIFeeder: Failed to delete orphaned embedding")
			continue
		}
		deleted++
	}

	f.logger.WithFields(logrus.Fields{
		"deleted": deleted,
		"total":   len(zombies),
	}).Info("SecureAIFeeder: Index reconciliation complete")

	return deleted, nil
}
