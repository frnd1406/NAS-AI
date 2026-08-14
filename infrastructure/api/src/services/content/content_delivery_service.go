package content

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/nas-ai/api/src/domain/auth"
	"github.com/nas-ai/api/src/domain/files"
	"github.com/nas-ai/api/src/services/security"

	"github.com/sirupsen/logrus"
)

type FileStreamResult struct {
	Stream        io.ReadCloser
	ContentLength int64
	ContentRange  string
	ContentType   string
	StatusCode    int
	ETag          string

	// XAccel headers for Nginx offloading
	XAccelRedirect  string
	XAccelBuffering string
}

type EncryptionStatus struct {
	Mode files.EncryptionMode
}

type ContentDeliveryService struct {
	storage       *StorageManager
	encryptionSvc *security.EncryptionService
	logger        *logrus.Logger
}

func NewContentDeliveryService(storage *StorageManager, encryptionSvc *security.EncryptionService, logger *logrus.Logger) *ContentDeliveryService {
	return &ContentDeliveryService{
		storage:       storage,
		encryptionSvc: encryptionSvc,
		logger:        logger,
	}
}

// GetStream prepares the file stream, handling encryption and range requests.
func (s *ContentDeliveryService) GetStream(ctx context.Context, path string, rangeHeader string, password string, mode string, user *auth.User) (*FileStreamResult, error) {
	// Get full filesystem path
	fullPath, err := s.storage.GetFullPath(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	// Check if file exists
	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, os.ErrNotExist
		}
		return nil, err
	}

	if fileInfo.IsDir() {
		return nil, fmt.Errorf("cannot download a directory")
	}

	// Detect encryption status
	encryptionStatus := s.detectEncryptionStatus(fullPath)

	// Route based on encryption status
	switch encryptionStatus {
	case files.EncryptionNone:
		return s.prepareUnencryptedStream(fullPath, fileInfo, rangeHeader)
	case files.EncryptionUser:
		if mode == "raw" {
			return s.prepareRawStream(fullPath, fileInfo)
		}
		return s.prepareEncryptedStream(fullPath, fileInfo, rangeHeader, password)
	case files.EncryptionSystem:
		return nil, fmt.Errorf("SYSTEM encryption not yet supported")
	default:
		return nil, fmt.Errorf("unknown encryption status")
	}
}

func (s *ContentDeliveryService) detectEncryptionStatus(fullPath string) files.EncryptionMode {
	// Check file extension first (fast path)
	if strings.HasSuffix(strings.ToLower(fullPath), ".enc") {
		// Verify with magic bytes
		isEnc, err := security.IsEncryptedFile(fullPath)
		if err != nil {
			s.logger.WithError(err).Warn("Failed to check encryption magic bytes")
			return files.EncryptionNone // Fail open for availability
		}
		if isEnc {
			return files.EncryptionUser
		}
	}
	return files.EncryptionNone
}

func (s *ContentDeliveryService) prepareUnencryptedStream(fullPath string, fileInfo os.FileInfo, rangeHeader string) (*FileStreamResult, error) {
	filename := fileInfo.Name()
	contentType := s.detectContentType(filename)

	// Check for X-Accel-Redirect (Nginx)
	useXAccel := os.Getenv("USE_NGINX_XACCEL") == "true"
	if useXAccel {
		// /mnt/data/folder/file.txt -> /protected-files/folder/file.txt
		xAccelPath := strings.Replace(fullPath, "/mnt/data", "/protected-files", 1)
		return &FileStreamResult{
			StatusCode:      200,
			ContentType:     contentType,
			XAccelRedirect:  xAccelPath,
			XAccelBuffering: "no",
		}, nil
	}

	// Direct File Serve with Range support logic (though http.ServeContent usually handles this,
	// for a unified interface we might need to manually handle range if we want to return a stream,
	// OR we can just return the file and let the handler use http.ServeContent if StatusCode is 200 and Stream is a *os.File.
	// However, the interface contract implies WE handle the stream.
	// BUT http.ServeContent is very good at handling Range requests for os.File.
	// To strictly follow the interface "GetStream", we should probably parse the range ourselves if we want consistent behavior
	// across encrypted/unencrypted, OR we return the file and let the handler decide.
	// Given the prompt "Berechnung der korrekten Offsets ... in diesen Service", let's handle it manually to be consistent.

	file, err := os.Open(fullPath)
	if err != nil {
		return nil, err
	}

	if rangeHeader != "" {
		start, end, err := s.parseRangeHeader(rangeHeader, fileInfo.Size())
		if err != nil {
			file.Close()
			return &FileStreamResult{
				StatusCode:   416, // Range Not Satisfiable
				ContentRange: fmt.Sprintf("bytes */%d", fileInfo.Size()),
			}, nil // Return valid result with error status
		}

		if _, err := file.Seek(start, io.SeekStart); err != nil {
			file.Close()
			return nil, err
		}

		length := end - start + 1
		return &FileStreamResult{
			Stream:        &limitReadCloser{Reader: io.LimitReader(file, length), Closer: file},
			ContentLength: length,
			ContentRange:  fmt.Sprintf("bytes %d-%d/%d", start, end, fileInfo.Size()),
			ContentType:   contentType,
			StatusCode:    206,
		}, nil
	}

	return &FileStreamResult{
		Stream:        file,
		ContentLength: fileInfo.Size(),
		ContentType:   contentType,
		StatusCode:    200,
	}, nil
}

func (s *ContentDeliveryService) prepareRawStream(fullPath string, fileInfo os.FileInfo) (*FileStreamResult, error) {
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, err
	}

	// Raw ciphertext is always octet-stream
	return &FileStreamResult{
		Stream:        file,
		ContentLength: fileInfo.Size(),
		ContentType:   "application/octet-stream",
		StatusCode:    200,
	}, nil
}

func (s *ContentDeliveryService) prepareEncryptedStream(fullPath string, fileInfo os.FileInfo, rangeHeader string, password string) (*FileStreamResult, error) {
	if password == "" {
		if s.encryptionSvc != nil && !s.encryptionSvc.IsUnlocked() {
			return nil, fmt.Errorf("VAULT_LOCKED") // Caller handles 423
		}
		// Require password for basic USER encryption if not using some other key mechanism (simplified)
		return nil, fmt.Errorf("PASSWORD_REQUIRED")
	}

	file, err := os.Open(fullPath)
	if err != nil {
		return nil, err
	}

	filename := fileInfo.Name()
	if strings.HasSuffix(strings.ToLower(filename), ".enc") {
		filename = filename[:len(filename)-4]
	}
	contentType := s.detectContentType(filename)

	// Get encrypted file info
	encInfo, err := security.GetEncryptedFileInfo(file)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to read file metadata: %w", err)
	}

	if !encInfo.IsValid {
		file.Close()
		return nil, fmt.Errorf("file is not properly encrypted")
	}

	plaintextSize := encInfo.EstimatedPlainSize

	if rangeHeader != "" {
		start, end, err := s.parseRangeHeader(rangeHeader, plaintextSize)
		if err != nil {
			file.Close()
			return &FileStreamResult{
				StatusCode:   416,
				ContentRange: fmt.Sprintf("bytes */%d", plaintextSize),
			}, nil
		}

		length := end - start + 1

		// Create a pipe to stream the decrypted content
		r, w := io.Pipe()

		go func() {
			defer w.Close()
			// Need to reopen file or seek reset? DecryptStreamWithSeek expects file to be open.
			// However `file` here is shared.
			// Wait, DecryptStreamWithSeek takes `io.ReadSeeker`.
			_, err := security.DecryptStreamWithSeek(password, file, w, start, length)
			if err != nil {
				s.logger.WithError(err).Error("Async decryption failed")
				w.CloseWithError(err)
			}
			// We cannot close `file` here easily if it's used by the reader,
			// but `DecryptStreamWithSeek` uses it.
			// The closer responsibility is tricky with pipes.
			// The `FileStreamResult.Stream` should close the underlying resources.
		}()

		// Wrapper to close both pipe reader and the underlying file
		stream := &pipeFileCloser{PipeReader: r, File: file}

		return &FileStreamResult{
			Stream:        stream,
			ContentLength: length,
			ContentRange:  fmt.Sprintf("bytes %d-%d/%d", start, end, plaintextSize),
			ContentType:   contentType,
			StatusCode:    206,
		}, nil

	} else {
		// Full stream
		r, w := io.Pipe()
		go func() {
			defer w.Close()
			// Reset seek just in case
			file.Seek(0, 0)
			err := security.DecryptStream(password, file, w)
			if err != nil {
				s.logger.WithError(err).Error("Async decryption failed")
				w.CloseWithError(err)
			}
		}()

		stream := &pipeFileCloser{PipeReader: r, File: file}

		return &FileStreamResult{
			Stream: stream,
			// No Content-Length for full encrypted stream (chunked)
			ContentType: contentType,
			StatusCode:  200,
		}, nil
	}
}

func (s *ContentDeliveryService) detectContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	case ".mkv":
		return "video/x-matroska"
	case ".avi":
		return "video/x-msvideo"
	case ".mov":
		return "video/quicktime"
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	case ".ogg":
		return "audio/ogg"
	case ".pdf":
		return "application/pdf"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".txt":
		return "text/plain; charset=utf-8"
	case ".json":
		return "application/json; charset=utf-8"
	case ".html":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".js":
		return "application/javascript; charset=utf-8"
	case ".md":
		return "text/markdown; charset=utf-8"
	case ".zip":
		return "application/zip"
	case ".tar":
		return "application/x-tar"
	case ".gz":
		return "application/gzip"
	}
	mimeType := mime.TypeByExtension(ext)
	if mimeType != "" {
		return mimeType
	}
	return "application/octet-stream"
}

func (s *ContentDeliveryService) parseRangeHeader(rangeHeader string, fileSize int64) (start, end int64, err error) {
	if !strings.HasPrefix(rangeHeader, "bytes=") {
		return 0, 0, errors.New("invalid range format")
	}
	rangeSpec := strings.TrimPrefix(rangeHeader, "bytes=")
	parts := strings.Split(rangeSpec, "-")
	if len(parts) != 2 {
		return 0, 0, errors.New("invalid range format")
	}

	if parts[0] == "" {
		suffixLen, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return 0, 0, err
		}
		start = fileSize - suffixLen
		end = fileSize - 1
	} else if parts[1] == "" {
		start, err = strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return 0, 0, err
		}
		end = fileSize - 1
	} else {
		start, err = strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return 0, 0, err
		}
		end, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return 0, 0, err
		}
	}

	if start < 0 || start > end || start >= fileSize {
		return 0, 0, errors.New("range not satisfiable")
	}
	if end >= fileSize {
		end = fileSize - 1
	}
	return start, end, nil
}

// Helpers for stream closing

type limitReadCloser struct {
	io.Reader
	Closer io.Closer
}

func (l *limitReadCloser) Close() error {
	return l.Closer.Close()
}

type pipeFileCloser struct {
	*io.PipeReader
	File *os.File
}

func (p *pipeFileCloser) Close() error {
	// Close pipe first to stop writer
	err1 := p.PipeReader.Close()
	// Then close file
	err2 := p.File.Close()

	if err1 != nil {
		return err1
	}
	return err2
}
