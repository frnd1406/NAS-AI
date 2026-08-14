package common

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// =============================================================================
// SENTINEL ERRORS (for error type checking)
// =============================================================================

var (
	// ErrServiceUnavailable indicates the remote service is not reachable
	ErrServiceUnavailable = errors.New("service unavailable")

	// ErrServiceTimeout indicates the request exceeded timeout
	ErrServiceTimeout = errors.New("service timeout")

	// ErrServiceOverloaded indicates the service returned 503/504/429
	ErrServiceOverloaded = errors.New("service overloaded")

	// ErrMaxRetriesExceeded indicates all retry attempts failed
	ErrMaxRetriesExceeded = errors.New("max retries exceeded")
)

// =============================================================================
// CONFIGURATION
// =============================================================================

// RetryConfig holds retry/backoff configuration
type RetryConfig struct {
	MaxRetries     int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	BackoffFactor  float64
}

// DefaultRetryConfig returns sensible defaults for resilient HTTP requests
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:     3,
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     30 * time.Second,
		BackoffFactor:  2.0,
	}
}

// =============================================================================
// RESILIENT HTTP CLIENT
// =============================================================================

// ResilientHTTPClient wraps http.Client with retry/backoff capabilities
// Single Responsibility: Resilient HTTP request execution
type ResilientHTTPClient struct {
	client *http.Client
	config RetryConfig
	logger *logrus.Logger
}

// NewResilientHTTPClient creates a new resilient HTTP client
func NewResilientHTTPClient(timeout time.Duration, config RetryConfig, logger *logrus.Logger) *ResilientHTTPClient {
	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 5,
			IdleConnTimeout:     90 * time.Second,
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
		},
	}

	return &ResilientHTTPClient{
		client: client,
		config: config,
		logger: logger,
	}
}

// Do executes an HTTP request with exponential backoff retry
func (c *ResilientHTTPClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	return c.DoWithRetry(ctx, req.Method+" "+req.URL.Path, func() (*http.Response, error) {
		return c.client.Do(req)
	})
}

// DoWithRetry executes a function with exponential backoff retry
// Use this when you need to rebuild the request body for each attempt
func (c *ResilientHTTPClient) DoWithRetry(ctx context.Context, operation string, fn func() (*http.Response, error)) (*http.Response, error) {
	var lastErr error
	backoff := c.config.InitialBackoff

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		// Check context before each attempt
		if ctx.Err() != nil {
			return nil, fmt.Errorf("%w: %v", ErrServiceTimeout, ctx.Err())
		}

		if attempt > 0 {
			c.logger.WithFields(logrus.Fields{
				"operation": operation,
				"attempt":   attempt,
				"backoff":   backoff.String(),
			}).Debug("Retrying request after backoff")

			// Wait with context awareness
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("%w: %v", ErrServiceTimeout, ctx.Err())
			case <-time.After(backoff):
			}

			// Exponential backoff with cap
			backoff = time.Duration(float64(backoff) * c.config.BackoffFactor)
			if backoff > c.config.MaxBackoff {
				backoff = c.config.MaxBackoff
			}
		}

		resp, err := fn()
		if err != nil {
			lastErr = c.classifyError(err)

			if !c.isRetryable(lastErr) {
				return nil, lastErr
			}

			c.logger.WithFields(logrus.Fields{
				"operation": operation,
				"attempt":   attempt + 1,
				"error":     err.Error(),
			}).Warn("Request failed, will retry")
			continue
		}

		// Check for retryable HTTP status codes
		if c.isRetryableStatusCode(resp.StatusCode) {
			lastErr = fmt.Errorf("%w: status %d", ErrServiceOverloaded, resp.StatusCode)
			resp.Body.Close()

			c.logger.WithFields(logrus.Fields{
				"operation":   operation,
				"attempt":     attempt + 1,
				"status_code": resp.StatusCode,
			}).Warn("Retryable status code, will retry")
			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("%w after %d attempts: %v", ErrMaxRetriesExceeded, c.config.MaxRetries+1, lastErr)
}

// classifyError converts a network error to a sentinel error
func (c *ResilientHTTPClient) classifyError(err error) error {
	if err == nil {
		return nil
	}

	// Context cancellation/timeout
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("%w: %v", ErrServiceTimeout, err)
	}

	// Connection refused
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		if netErr.Op == "dial" {
			return fmt.Errorf("%w: %v", ErrServiceUnavailable, err)
		}
	}

	// DNS errors
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return fmt.Errorf("%w: DNS lookup failed: %v", ErrServiceUnavailable, err)
	}

	return fmt.Errorf("%w: %v", ErrServiceUnavailable, err)
}

// isRetryable determines if an error warrants a retry
func (c *ResilientHTTPClient) isRetryable(err error) bool {
	return errors.Is(err, ErrServiceUnavailable) || errors.Is(err, ErrServiceOverloaded)
}

// isRetryableStatusCode returns true for HTTP status codes that warrant retry
func (c *ResilientHTTPClient) isRetryableStatusCode(statusCode int) bool {
	switch statusCode {
	case http.StatusServiceUnavailable, // 503
		http.StatusGatewayTimeout,  // 504
		http.StatusTooManyRequests: // 429
		return true
	default:
		return false
	}
}

// =============================================================================
// UTILITY FUNCTIONS
// =============================================================================

// CalculateBackoff computes the backoff duration for a given attempt
func CalculateBackoff(attempt int, initial, max time.Duration, factor float64) time.Duration {
	if attempt <= 0 {
		return 0
	}
	backoff := float64(initial) * math.Pow(factor, float64(attempt-1))
	if time.Duration(backoff) > max {
		return max
	}
	return time.Duration(backoff)
}
