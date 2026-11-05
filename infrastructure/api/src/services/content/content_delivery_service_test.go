package content

import (
	"bytes"
	"context"
	"crypto/rand"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/nas-ai/api/src/drivers/storage"
	"github.com/nas-ai/api/src/services/security"
	"github.com/sirupsen/logrus"
)

// Helper function to create a StorageManager for testing
func newTestStorageManager(t *testing.T, basePath string) *StorageManager {
	store, err := storage.NewLocalStore(basePath)
	if err != nil {
		t.Fatalf("Failed to create local store: %v", err)
	}
	logger := logrus.New()
	return NewStorageManager(store, nil, nil, logger)
}

// TestContentDeliveryService_ParseRangeHeader tests range header parsing
func TestContentDeliveryService_ParseRangeHeader(t *testing.T) {
	logger := logrus.New()
	service := &ContentDeliveryService{logger: logger}

	testCases := []struct {
		name        string
		rangeHeader string
		fileSize    int64
		wantStart   int64
		wantEnd     int64
		wantErr     bool
	}{
		// Valid ranges
		{"StartToEnd", "bytes=0-100", 1000, 0, 100, false},
		{"MidRange", "bytes=100-200", 1000, 100, 200, false},
		{"FromStart", "bytes=0-999", 1000, 0, 999, false},
		{"OpenEnd", "bytes=500-", 1000, 500, 999, false},
		{"SuffixRange", "bytes=-100", 1000, 900, 999, false},

		// Edge cases
		{"SingleByte", "bytes=0-0", 1000, 0, 0, false},
		{"LastByte", "bytes=999-999", 1000, 999, 999, false},
		{"EndBeyondFile", "bytes=500-2000", 1000, 500, 999, false}, // End should be capped

		// Invalid ranges
		{"InvalidFormat", "invalid", 1000, 0, 0, true},
		{"NoBytes", "chars=0-100", 1000, 0, 0, true},
		{"StartBeyondEnd", "bytes=200-100", 1000, 0, 0, true},
		{"StartBeyondFile", "bytes=2000-3000", 1000, 0, 0, true},
		{"NegativeStart", "bytes=-0", 1000, 0, 0, true}, // This is suffix range with 0 bytes
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			start, end, err := service.parseRangeHeader(tc.rangeHeader, tc.fileSize)
			if tc.wantErr {
				if err == nil {
					t.Errorf("Expected error for %q, got none", tc.rangeHeader)
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error for %q: %v", tc.rangeHeader, err)
				return
			}
			if start != tc.wantStart || end != tc.wantEnd {
				t.Errorf("parseRangeHeader(%q, %d) = (%d, %d), want (%d, %d)",
					tc.rangeHeader, tc.fileSize, start, end, tc.wantStart, tc.wantEnd)
			}
		})
	}
}

// TestContentDeliveryService_DetectContentType tests MIME type detection
func TestContentDeliveryService_DetectContentType(t *testing.T) {
	logger := logrus.New()
	service := &ContentDeliveryService{logger: logger}

	testCases := []struct {
		filename string
		expected string
	}{
		// Video
		{"video.mp4", "video/mp4"},
		{"movie.mkv", "video/x-matroska"},
		{"clip.webm", "video/webm"},
		{"film.avi", "video/x-msvideo"},
		{"recording.mov", "video/quicktime"},

		// Audio
		{"song.mp3", "audio/mpeg"},
		{"audio.wav", "audio/wav"},
		{"podcast.ogg", "audio/ogg"},

		// Images
		{"photo.jpg", "image/jpeg"},
		{"photo.jpeg", "image/jpeg"},
		{"image.png", "image/png"},
		{"animation.gif", "image/gif"},
		{"photo.webp", "image/webp"},

		// Documents
		{"document.pdf", "application/pdf"},
		{"readme.txt", "text/plain; charset=utf-8"},
		{"data.json", "application/json; charset=utf-8"},
		{"page.html", "text/html; charset=utf-8"},
		{"style.css", "text/css; charset=utf-8"},
		{"script.js", "application/javascript; charset=utf-8"},
		{"notes.md", "text/markdown; charset=utf-8"},

		// Archives
		{"archive.zip", "application/zip"},
		{"backup.tar", "application/x-tar"},
		{"compressed.gz", "application/gzip"},

		// Unknown
		{"unknown.xyz", "application/octet-stream"},
		{"noextension", "application/octet-stream"},
	}

	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			result := service.detectContentType(tc.filename)
			if result != tc.expected {
				t.Errorf("detectContentType(%q) = %q, want %q", tc.filename, result, tc.expected)
			}
		})
	}
}

// TestContentDeliveryService_GetStream_UnencryptedFile tests downloading an unencrypted file
func TestContentDeliveryService_GetStream_UnencryptedFile(t *testing.T) {
	logger := logrus.New()

	// Create temp directory as storage root
	tempDir, err := os.MkdirTemp("", "content-delivery-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test file
	testContent := []byte("Hello, World! This is test content for streaming.")
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Create storage manager pointing to temp dir
	storage := newTestStorageManager(t, tempDir)
	service := NewContentDeliveryService(storage, nil, logger)

	// Test full download
	result, err := service.GetStream(context.Background(), "test.txt", "", "", "auto", nil)
	if err != nil {
		t.Fatalf("GetStream failed: %v", err)
	}

	if result.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", result.StatusCode)
	}

	if result.ContentLength != int64(len(testContent)) {
		t.Errorf("Expected ContentLength %d, got %d", len(testContent), result.ContentLength)
	}

	if result.ContentType != "text/plain; charset=utf-8" {
		t.Errorf("Expected ContentType 'text/plain; charset=utf-8', got %q", result.ContentType)
	}

	// Read and verify content
	if result.Stream != nil {
		defer result.Stream.Close()
		buf := new(bytes.Buffer)
		_, err := io.Copy(buf, result.Stream)
		if err != nil {
			t.Fatalf("Failed to read stream: %v", err)
		}
		if !bytes.Equal(buf.Bytes(), testContent) {
			t.Errorf("Content mismatch: got %q, want %q", buf.String(), string(testContent))
		}
	}
}

// TestContentDeliveryService_GetStream_RangeRequest tests partial content delivery
func TestContentDeliveryService_GetStream_RangeRequest(t *testing.T) {
	logger := logrus.New()

	tempDir, err := os.MkdirTemp("", "content-delivery-range-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create larger test file
	testContent := make([]byte, 1000)
	for i := range testContent {
		testContent[i] = byte(i % 256)
	}
	testFile := filepath.Join(tempDir, "data.bin")
	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	storage := newTestStorageManager(t, tempDir)
	service := NewContentDeliveryService(storage, nil, logger)

	testCases := []struct {
		name               string
		rangeHeader        string
		expectedStatus     int
		expectedLength     int64
		expectedStart      int
		expectedEnd        int
		expectContentRange bool
	}{
		{
			name:               "First100Bytes",
			rangeHeader:        "bytes=0-99",
			expectedStatus:     206,
			expectedLength:     100,
			expectedStart:      0,
			expectedEnd:        99,
			expectContentRange: true,
		},
		{
			name:               "MidRange",
			rangeHeader:        "bytes=100-199",
			expectedStatus:     206,
			expectedLength:     100,
			expectedStart:      100,
			expectedEnd:        199,
			expectContentRange: true,
		},
		{
			name:               "LastBytes",
			rangeHeader:        "bytes=-50",
			expectedStatus:     206,
			expectedLength:     50,
			expectedStart:      950,
			expectedEnd:        999,
			expectContentRange: true,
		},
		{
			name:               "OpenEnd",
			rangeHeader:        "bytes=900-",
			expectedStatus:     206,
			expectedLength:     100,
			expectedStart:      900,
			expectedEnd:        999,
			expectContentRange: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := service.GetStream(context.Background(), "data.bin", tc.rangeHeader, "", "auto", nil)
			if err != nil {
				t.Fatalf("GetStream failed: %v", err)
			}

			if result.StatusCode != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, result.StatusCode)
			}

			if result.ContentLength != tc.expectedLength {
				t.Errorf("Expected ContentLength %d, got %d", tc.expectedLength, result.ContentLength)
			}

			if tc.expectContentRange && result.ContentRange == "" {
				t.Error("Expected Content-Range header, got empty")
			}

			// Verify actual content
			if result.Stream != nil {
				defer result.Stream.Close()
				buf := new(bytes.Buffer)
				n, _ := io.Copy(buf, result.Stream)
				if n != tc.expectedLength {
					t.Errorf("Read %d bytes, expected %d", n, tc.expectedLength)
				}

				expectedData := testContent[tc.expectedStart : tc.expectedEnd+1]
				if !bytes.Equal(buf.Bytes(), expectedData) {
					t.Errorf("Content mismatch for range %s", tc.rangeHeader)
				}
			}
		})
	}
}

// TestContentDeliveryService_GetStream_InvalidPath tests error handling for invalid paths
func TestContentDeliveryService_GetStream_InvalidPath(t *testing.T) {
	logger := logrus.New()

	tempDir, err := os.MkdirTemp("", "content-delivery-invalid-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage := newTestStorageManager(t, tempDir)
	service := NewContentDeliveryService(storage, nil, logger)

	// Test non-existent file
	_, err = service.GetStream(context.Background(), "nonexistent.txt", "", "", "auto", nil)
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
	if !os.IsNotExist(err) {
		t.Logf("Got error (expected): %v", err)
	}

	// Test directory instead of file
	if err := os.Mkdir(filepath.Join(tempDir, "subdir"), 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}
	_, err = service.GetStream(context.Background(), "subdir", "", "", "auto", nil)
	if err == nil {
		t.Error("Expected error for directory, got nil")
	}
}

// TestContentDeliveryService_GetStream_EncryptedFile tests encrypted file handling
func TestContentDeliveryService_GetStream_EncryptedFile(t *testing.T) {
	logger := logrus.New()

	tempDir, err := os.MkdirTemp("", "content-delivery-encrypted-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create encrypted test file
	password := "test-password-123"
	testContent := []byte("Secret data that should be encrypted")

	var encryptedBuf bytes.Buffer
	if err := security.EncryptStream(password, bytes.NewReader(testContent), &encryptedBuf); err != nil {
		t.Fatalf("EncryptStream failed: %v", err)
	}

	encryptedFile := filepath.Join(tempDir, "secret.txt.enc")
	if err := os.WriteFile(encryptedFile, encryptedBuf.Bytes(), 0644); err != nil {
		t.Fatalf("Failed to write encrypted file: %v", err)
	}

	storage := newTestStorageManager(t, tempDir)
	service := NewContentDeliveryService(storage, nil, logger)

	// Test without password (should fail)
	_, err = service.GetStream(context.Background(), "secret.txt.enc", "", "", "auto", nil)
	if err == nil {
		t.Error("Expected error when accessing encrypted file without password")
	}

	// Test with password (should succeed)
	result, err := service.GetStream(context.Background(), "secret.txt.enc", "", password, "auto", nil)
	if err != nil {
		t.Fatalf("GetStream with password failed: %v", err)
	}

	if result.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", result.StatusCode)
	}

	// Read and verify decrypted content
	if result.Stream != nil {
		defer result.Stream.Close()
		buf := new(bytes.Buffer)
		_, err := io.Copy(buf, result.Stream)
		if err != nil {
			t.Fatalf("Failed to read stream: %v", err)
		}
		if !bytes.Equal(buf.Bytes(), testContent) {
			t.Errorf("Decrypted content mismatch: got %q, want %q", buf.String(), string(testContent))
		}
	}

	// Test raw mode (should return ciphertext)
	result, err = service.GetStream(context.Background(), "secret.txt.enc", "", "", "raw", nil)
	if err != nil {
		t.Fatalf("GetStream raw mode failed: %v", err)
	}

	if result.ContentType != "application/octet-stream" {
		t.Errorf("Expected octet-stream for raw mode, got %q", result.ContentType)
	}
}

// TestContentDeliveryService_GetStream_EncryptedRangeRequest tests range requests on encrypted files
func TestContentDeliveryService_GetStream_EncryptedRangeRequest(t *testing.T) {
	logger := logrus.New()

	tempDir, err := os.MkdirTemp("", "content-delivery-encrypted-range-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create larger encrypted test file (multiple chunks)
	password := "range-test-password"
	testContent := make([]byte, security.ChunkSize*2+1000) // ~130KB
	if _, err := rand.Read(testContent); err != nil {
		t.Fatalf("Failed to generate random data: %v", err)
	}

	var encryptedBuf bytes.Buffer
	if err := security.EncryptStream(password, bytes.NewReader(testContent), &encryptedBuf); err != nil {
		t.Fatalf("EncryptStream failed: %v", err)
	}

	encryptedFile := filepath.Join(tempDir, "video.mp4.enc")
	if err := os.WriteFile(encryptedFile, encryptedBuf.Bytes(), 0644); err != nil {
		t.Fatalf("Failed to write encrypted file: %v", err)
	}

	storage := newTestStorageManager(t, tempDir)
	service := NewContentDeliveryService(storage, nil, logger)

	testCases := []struct {
		name        string
		rangeHeader string
		wantStatus  int
		wantLength  int64
		startByte   int // Expected start byte in plaintext
	}{
		{"FirstBytes", "bytes=0-100", 206, 101, 0},
		{"CrossChunkBoundary", "bytes=65000-66000", 206, 1001, 65000},
		{"SecondChunk", "bytes=65536-66000", 206, 465, 65536},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := service.GetStream(context.Background(), "video.mp4.enc", tc.rangeHeader, password, "auto", nil)
			if err != nil {
				t.Fatalf("GetStream failed: %v", err)
			}

			if result.StatusCode != tc.wantStatus {
				t.Errorf("Expected status %d, got %d", tc.wantStatus, result.StatusCode)
			}

			if result.ContentLength != tc.wantLength {
				t.Errorf("Expected ContentLength %d, got %d", tc.wantLength, result.ContentLength)
			}

			// Verify decrypted range matches expected plaintext
			if result.Stream != nil {
				defer result.Stream.Close()
				buf := new(bytes.Buffer)
				n, _ := io.Copy(buf, result.Stream)
				if n != tc.wantLength {
					t.Errorf("Read %d bytes, expected %d", n, tc.wantLength)
				}

				expectedData := testContent[tc.startByte : tc.startByte+int(tc.wantLength)]
				if !bytes.Equal(buf.Bytes(), expectedData) {
					t.Errorf("Decrypted range content mismatch for %s", tc.rangeHeader)
					t.Logf("Got first 32 bytes: %x", buf.Bytes()[:minInt(32, buf.Len())])
					t.Logf("Want first 32 bytes: %x", expectedData[:minInt(32, len(expectedData))])
				}
			}
		})
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
