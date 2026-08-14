package operations

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestBenchmarkService_RunStartupBenchmark(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Use smaller test size for faster unit tests
	svc := NewBenchmarkServiceWithSize(logger, 1*1024*1024) // 1 MB

	err := svc.RunStartupBenchmark()
	if err != nil {
		t.Fatalf("RunStartupBenchmark failed: %v", err)
	}

	// Verify result is valid
	result := svc.GetResult()
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if !result.IsValid {
		t.Error("Expected IsValid to be true")
	}

	if result.SpeedMBps <= 0 {
		t.Errorf("Expected positive speed, got %f", result.SpeedMBps)
	}

	if result.CPUCores <= 0 {
		t.Errorf("Expected positive CPU cores, got %d", result.CPUCores)
	}

	if result.Algorithm != "XChaCha20-Poly1305" {
		t.Errorf("Expected XChaCha20-Poly1305, got %s", result.Algorithm)
	}

	t.Logf("Benchmark result: %.2f MB/s", result.SpeedMBps)
}

func TestBenchmarkService_EstimateDuration(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	svc := NewBenchmarkServiceWithSize(logger, 1*1024*1024) // 1 MB

	// Before benchmark, should return 0
	duration := svc.EstimateDuration(100 * 1024 * 1024) // 100 MB
	if duration != 0 {
		t.Errorf("Expected 0 before benchmark, got %v", duration)
	}

	// Run benchmark
	if err := svc.RunStartupBenchmark(); err != nil {
		t.Fatalf("RunStartupBenchmark failed: %v", err)
	}

	// After benchmark, should return positive duration
	duration = svc.EstimateDuration(100 * 1024 * 1024) // 100 MB
	if duration <= 0 {
		t.Errorf("Expected positive duration after benchmark, got %v", duration)
	}

	// Test with 0 bytes should return ~0
	duration = svc.EstimateDuration(0)
	if duration < 0 {
		t.Errorf("Expected non-negative duration for 0 bytes, got %v", duration)
	}

	t.Logf("Estimated duration for 100 MB: %v", duration)
}

func TestBenchmarkService_GetRecommendation(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	svc := NewBenchmarkServiceWithSize(logger, 1*1024*1024) // 1 MB

	// Before benchmark - should return warning
	rec := svc.GetRecommendation(100 * 1024 * 1024)
	if !rec.Warning {
		t.Error("Expected warning before benchmark")
	}
	if rec.EstimatedTimeSeconds != -1 {
		t.Errorf("Expected -1 for unknown time, got %f", rec.EstimatedTimeSeconds)
	}

	// Run benchmark
	if err := svc.RunStartupBenchmark(); err != nil {
		t.Fatalf("RunStartupBenchmark failed: %v", err)
	}

	// Small file - should have no warning
	rec = svc.GetRecommendation(1 * 1024 * 1024) // 1 MB
	if rec.Warning {
		t.Error("Expected no warning for small file")
	}

	// Very large file - might have warning depending on speed
	rec = svc.GetRecommendation(100 * 1024 * 1024 * 1024) // 100 GB
	// Just verify it doesn't panic and returns valid data
	if rec.EstimatedTimeSeconds < 0 {
		t.Errorf("Expected positive estimated time, got %f", rec.EstimatedTimeSeconds)
	}

	t.Logf("Recommendation for 100 GB: Warning=%v, Time=%.2fs", rec.Warning, rec.EstimatedTimeSeconds)
}

func TestBenchmarkService_ThreadSafety(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	svc := NewBenchmarkServiceWithSize(logger, 512*1024) // 512 KB for speed

	// Run benchmark
	if err := svc.RunStartupBenchmark(); err != nil {
		t.Fatalf("RunStartupBenchmark failed: %v", err)
	}

	// Concurrent reads should not panic
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_ = svc.GetResult()
				_ = svc.GetSpeedMBps()
				_ = svc.EstimateDuration(10 * 1024 * 1024)
				_ = svc.IsReady()
				_ = svc.GetRecommendation(5 * 1024 * 1024)
			}
			done <- true
		}()
	}

	// Wait for all goroutines with timeout
	timeout := time.After(5 * time.Second)
	for i := 0; i < 10; i++ {
		select {
		case <-done:
			// OK
		case <-timeout:
			t.Fatal("Timeout waiting for goroutines")
		}
	}
}
