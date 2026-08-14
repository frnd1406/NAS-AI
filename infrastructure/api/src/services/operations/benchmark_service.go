package operations

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"runtime"
	"sync"
	"time"

	"github.com/nas-ai/api/src/services/security"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/chacha20poly1305"
)

// ==============================================================================
// Benchmark Service - "Performance Guard"
// ==============================================================================
//
// Purpose: Measures system crypto throughput at startup to enable intelligent
// warnings for users about expected encryption times.
//
// Use Case (Raspberry Pi 5):
// - User uploads 1GB file with encryption enabled
// - System knows it can encrypt at ~150 MB/s
// - Frontend shows: "Estimated time: ~7 seconds"
//
// ==============================================================================

// BenchmarkResult holds the results of a system benchmark
type BenchmarkResult struct {
	SpeedMBps     float64   `json:"speed_mbps"`      // Throughput in MB/s
	TestSizeBytes int64     `json:"test_size_bytes"` // Size of test data
	DurationMs    int64     `json:"duration_ms"`     // Time taken in milliseconds
	Timestamp     time.Time `json:"timestamp"`       // When benchmark was run
	CPUCores      int       `json:"cpu_cores"`       // Available CPU cores
	Algorithm     string    `json:"algorithm"`       // Encryption algorithm used
	ChunkSize     int       `json:"chunk_size"`      // Chunk size used
	IsValid       bool      `json:"is_valid"`        // Whether benchmark completed successfully
}

// BenchmarkService measures and stores system crypto performance
type BenchmarkService struct {
	mu     sync.RWMutex
	result *BenchmarkResult
	logger *logrus.Logger

	// Configuration
	testSizeBytes int64 // Default: 10 MB
	warmupRounds  int   // Warmup iterations before measurement
}

// NewBenchmarkService creates a new benchmark service
func NewBenchmarkService(logger *logrus.Logger) *BenchmarkService {
	return &BenchmarkService{
		logger:        logger,
		testSizeBytes: 10 * 1024 * 1024, // 10 MB default
		warmupRounds:  2,
	}
}

// NewBenchmarkServiceWithSize creates a benchmark service with custom test size
func NewBenchmarkServiceWithSize(logger *logrus.Logger, testSizeBytes int64) *BenchmarkService {
	return &BenchmarkService{
		logger:        logger,
		testSizeBytes: testSizeBytes,
		warmupRounds:  2,
	}
}

// RunStartupBenchmark performs a crypto speed test and caches the result.
// This should be called once at application startup.
//
// The test:
// 1. Generates random data in RAM
// 2. Encrypts it using XChaCha20-Poly1305 (same as NasCrypt V2)
// 3. Measures throughput
// 4. Stores result for future EstimateDuration() calls
func (b *BenchmarkService) RunStartupBenchmark() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.logger.Info("🚀 Starting crypto benchmark (Performance Guard)...")

	// Generate random test data
	testData := make([]byte, b.testSizeBytes)
	if _, err := io.ReadFull(rand.Reader, testData); err != nil {
		b.logger.WithError(err).Error("Failed to generate test data")
		return fmt.Errorf("failed to generate test data: %w", err)
	}

	// Generate random key and nonce
	key := make([]byte, security.KeySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return fmt.Errorf("failed to generate key: %w", err)
	}

	// Create XChaCha20-Poly1305 cipher
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}

	nonce := make([]byte, security.NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Warmup runs to get CPU caches hot
	warmupBuf := make([]byte, 0, len(testData)+security.TagSize)
	for i := 0; i < b.warmupRounds; i++ {
		warmupBuf = aead.Seal(warmupBuf[:0], nonce, testData[:security.ChunkSize], nil)
	}

	// Pre-allocate output buffer
	outputBuf := make([]byte, 0, len(testData)+security.TagSize)

	// Force GC before benchmark for consistent results
	runtime.GC()

	// === BENCHMARK START ===
	startTime := time.Now()

	// Encrypt the entire test data using chunked approach (like NasCrypt V2)
	reader := bytes.NewReader(testData)
	writer := &bytes.Buffer{}
	writer.Grow(int(b.testSizeBytes) + security.HeaderSize + (int(b.testSizeBytes)/security.ChunkSize+1)*security.TagSize)

	chunkBuf := make([]byte, security.ChunkSize)
	chunkNonce := make([]byte, security.NonceSize)
	copy(chunkNonce, nonce) // Use base nonce

	var chunkIndex uint64 = 0
	for {
		n, err := io.ReadFull(reader, chunkBuf)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return fmt.Errorf("failed to read chunk: %w", err)
		}
		if n == 0 {
			break
		}

		// Derive chunk nonce (same algorithm as NasCrypt V2)
		security.DeriveChunkNonceInPlace(nonce, chunkIndex, chunkNonce)

		// Encrypt chunk
		outputBuf = aead.Seal(outputBuf[:0], chunkNonce, chunkBuf[:n], nil)
		writer.Write(outputBuf)

		chunkIndex++

		if n < security.ChunkSize {
			break
		}
	}
	duration := time.Since(startTime)
	// === BENCHMARK END ===

	// Calculate throughput
	durationSeconds := duration.Seconds()
	if durationSeconds == 0 {
		durationSeconds = 0.001 // Avoid division by zero
	}

	sizeMB := float64(b.testSizeBytes) / (1024 * 1024)
	speedMBps := sizeMB / durationSeconds

	// Store result
	b.result = &BenchmarkResult{
		SpeedMBps:     speedMBps,
		TestSizeBytes: b.testSizeBytes,
		DurationMs:    duration.Milliseconds(),
		Timestamp:     time.Now(),
		CPUCores:      runtime.NumCPU(),
		Algorithm:     "XChaCha20-Poly1305",
		ChunkSize:     security.ChunkSize,
		IsValid:       true,
	}

	// Log the result
	b.logger.WithFields(logrus.Fields{
		"speed_mbps":    fmt.Sprintf("%.1f", speedMBps),
		"test_size_mb":  fmt.Sprintf("%.1f", sizeMB),
		"duration_ms":   duration.Milliseconds(),
		"cpu_cores":     runtime.NumCPU(),
		"chunk_size_kb": security.ChunkSize / 1024,
	}).Info("✅ Crypto benchmark complete - Performance Guard ready")

	return nil
}

// GetResult returns the current benchmark result (thread-safe)
func (b *BenchmarkService) GetResult() *BenchmarkResult {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.result == nil {
		return &BenchmarkResult{
			IsValid:   false,
			CPUCores:  runtime.NumCPU(),
			Algorithm: "XChaCha20-Poly1305",
			ChunkSize: security.ChunkSize,
		}
	}

	// Return a copy to prevent external mutation
	result := *b.result
	return &result
}

// EstimateDuration calculates how long encryption would take for a given file size.
// Includes a 10% safety buffer for I/O overhead.
//
// Returns 0 if benchmark hasn't run yet.
func (b *BenchmarkService) EstimateDuration(sizeBytes int64) time.Duration {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.result == nil || !b.result.IsValid || b.result.SpeedMBps <= 0 {
		return 0
	}

	// Calculate base time
	sizeMB := float64(sizeBytes) / (1024 * 1024)
	baseSeconds := sizeMB / b.result.SpeedMBps

	// Add 10% safety buffer for I/O overhead
	bufferedSeconds := baseSeconds * 1.10

	return time.Duration(bufferedSeconds * float64(time.Second))
}

// EstimateDurationSeconds is a convenience method returning seconds as float64
func (b *BenchmarkService) EstimateDurationSeconds(sizeBytes int64) float64 {
	duration := b.EstimateDuration(sizeBytes)
	return duration.Seconds()
}

// GetSpeedMBps returns the current measured speed in MB/s
func (b *BenchmarkService) GetSpeedMBps() float64 {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.result == nil || !b.result.IsValid {
		return 0
	}
	return b.result.SpeedMBps
}

// IsReady returns true if a valid benchmark result is available
func (b *BenchmarkService) IsReady() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.result != nil && b.result.IsValid
}

// ShouldWarn returns true if encryption time exceeds the threshold (default: 60 seconds)
func (b *BenchmarkService) ShouldWarn(sizeBytes int64, thresholdSeconds float64) bool {
	if thresholdSeconds <= 0 {
		thresholdSeconds = 60 // Default: warn if > 60 seconds
	}
	return b.EstimateDurationSeconds(sizeBytes) > thresholdSeconds
}

// GetRecommendation returns a recommendation for encrypting a file of given size
type EncryptionRecommendation struct {
	EncryptSupported     bool    `json:"encrypt_supported"`
	EstimatedTimeSeconds float64 `json:"estimated_time_seconds"`
	Warning              bool    `json:"warning"`
	WarningMessage       string  `json:"warning_message,omitempty"`
}

// GetRecommendation generates an encryption recommendation for a file size
func (b *BenchmarkService) GetRecommendation(sizeBytes int64) EncryptionRecommendation {
	b.mu.RLock()
	defer b.mu.RUnlock()

	rec := EncryptionRecommendation{
		EncryptSupported: true, // Always supported, just might be slow
	}

	if b.result == nil || !b.result.IsValid || b.result.SpeedMBps <= 0 {
		// No benchmark data - provide conservative estimate
		rec.EstimatedTimeSeconds = -1 // Unknown
		rec.Warning = true
		rec.WarningMessage = "System benchmark not available. Encryption times unknown."
		return rec
	}

	// Calculate estimated time with 10% buffer
	sizeMB := float64(sizeBytes) / (1024 * 1024)
	baseSeconds := sizeMB / b.result.SpeedMBps
	rec.EstimatedTimeSeconds = baseSeconds * 1.10

	// Set warning if > 60 seconds
	if rec.EstimatedTimeSeconds > 60 {
		rec.Warning = true
		rec.WarningMessage = fmt.Sprintf(
			"Large file detected. Encryption will take approximately %.0f seconds.",
			rec.EstimatedTimeSeconds,
		)
	}

	return rec
}
