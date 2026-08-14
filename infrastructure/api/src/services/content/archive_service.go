package content

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// Security constants for ZIP extraction
const (
	// ZipMagicBytes - First 4 bytes of a valid ZIP file
	ZipMagicBytes = "PK\x03\x04"

	// MaxDecompressedSize - Maximum total decompressed size (1 GB)
	MaxDecompressedSize int64 = 1024 * 1024 * 1024

	// MaxFileCount - Maximum number of files allowed in a ZIP
	MaxFileCount = 10000

	// MaxSingleFileSize - Maximum size for a single extracted file (500MB)
	MaxSingleFileSize int64 = 500 * 1024 * 1024

	// MaxCompressionRatio - Maximum allowed ratio of uncompressed/compressed size
	MaxCompressionRatio = 100
)

// UnzipResult contains information about the extraction result
type UnzipResult struct {
	ExtractedFiles []string
	TotalBytes     int64
	FileCount      int
}

type ArchiveService struct {
	logger *logrus.Logger
}

func NewArchiveService(logger *logrus.Logger) *ArchiveService {
	return &ArchiveService{
		logger: logger,
	}
}

// UnzipSecure extracts a ZIP archive securely with multiple protection layers
func (s *ArchiveService) UnzipSecure(ctx context.Context, src io.Reader, size int64, destPath string) (*UnzipResult, error) {
	// Read full content into memory to support zip.NewReader (ReaderAt)
	// In a more advanced version, we might check if src implements ReaderAt or use a temp file.
	// For now, we read into buffer to match previous logic and support IO limits.

	// Create a buffer with pre-allocated size if possible
	buf := bytes.NewBuffer(make([]byte, 0, size))
	_, err := io.Copy(buf, src)
	if err != nil {
		return nil, fmt.Errorf("failed to read zip content: %w", err)
	}
	zipData := buf.Bytes()

	// 1. Check Magic Bytes strictly
	if len(zipData) < 4 || string(zipData[:4]) != ZipMagicBytes {
		return nil, fmt.Errorf("SECURITY: Invalid file signature (not a zip)")
	}

	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, err
	}

	// 2. Pre-Check Limits (DoS Prevention)
	if len(reader.File) > MaxFileCount {
		return nil, fmt.Errorf("SECURITY: Too many files in archive (max %d)", MaxFileCount)
	}

	// Prepare result tracking
	result := &UnzipResult{
		ExtractedFiles: make([]string, 0),
	}

	destination := filepath.Clean(destPath)
	var totalSize int64 = 0

	for _, f := range reader.File {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// 3. Path Traversal Prevention (Zip Slip)
		// Clean the path to resolve ".." and "."
		fpath := filepath.Join(destination, f.Name)
		if !strings.HasPrefix(fpath, filepath.Clean(destination)+string(os.PathSeparator)) {
			s.logger.Warnf("Zip Slip attempt detected: %s tries to write outside %s", f.Name, destination)
			return nil, fmt.Errorf("SECURITY: Illegal file path in zip: %s", f.Name)
		}

		// 4. Block Symlinks (Risk of accessing system files)
		if f.Mode()&os.ModeSymlink != 0 {
			s.logger.Warnf("Symlink detected and blocked: %s", f.Name)
			continue // Skip symlinks silently or return error based on strictness. Here we skip.
		}

		// 5. Individual File Size Limit
		if f.UncompressedSize64 > uint64(MaxSingleFileSize) {
			return nil, fmt.Errorf("SECURITY: File %s exceeds max size limit", f.Name)
		}

		// 6. Check for Zip Bomb (Compression Ratio)
		if f.UncompressedSize64 > 0 && f.CompressedSize64 > 0 {
			ratio := float64(f.UncompressedSize64) / float64(f.CompressedSize64)
			if ratio > float64(MaxCompressionRatio) {
				return nil, fmt.Errorf("SECURITY: Compression ratio too high (Zip Bomb detected) in %s", f.Name)
			}
		}

		totalSize += int64(f.UncompressedSize64)
		if totalSize > MaxDecompressedSize {
			return nil, fmt.Errorf("SECURITY: Total decompressed size exceeds limit")
		}

		// Check if file content is actually extractable
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		// Create directory for file if needed
		if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
			return nil, err
		}

		// 7. Sanitize Permissions
		// Never trust file modes from the zip. Use safe defaults.
		// 0644 = rw-r--r-- (User readable/writable, others readable, NO EXECUTE)
		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return nil, err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return nil, err
		}

		// Copy securely preventing massive memory allocation
		_, err = io.Copy(outFile, rc)

		outFile.Close()
		rc.Close()

		if err != nil {
			return nil, err
		}

		result.ExtractedFiles = append(result.ExtractedFiles, fpath)
		result.FileCount++
	}

	result.TotalBytes = totalSize
	return result, nil
}
