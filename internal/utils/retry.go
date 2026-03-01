package utils

import (
	"context"
	"math"
	"math/rand"
	"strings"
	"time"
)

// RetryConfig holds the configuration for the retry mechanism.
type RetryConfig struct {
	MaxAttempts     int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	BackoffFactor   float64
	Timeout         time.Duration
	RetryableErrors []string
}

// RetryableFunc defines the signature for operations that can be retried.
type RetryableFunc[T any] func(ctx context.Context) (T, error)

// DefaultRetryConfig returns a RetryConfig with sensible default values.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  1 * time.Second,
		MaxDelay:      5 * time.Second,
		BackoffFactor: 2.0,
		Timeout:       30 * time.Second,
		RetryableErrors: []string{
			"timeout",
			"connection reset",
			"rate limit",
			"connection refused",
			"socket hang up",
			"5", // covers 5xx status codes usually mentioned in error messages
		},
	}
}

// FastRetryConfig returns a RetryConfig optimized for fast retries (5-30s).
// Use this for Instagram fetch failures where quick retry is preferred.
func FastRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:   5,
		InitialDelay:  5 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		Timeout:       30 * time.Second,
		RetryableErrors: []string{
			"timeout",
			"connection reset",
			"rate limit",
			"connection refused",
			"socket hang up",
			"5",                 // covers 5xx status codes
			"invalid character", // Instagram returns HTML instead of JSON (captcha/error page)
			"<",                 // HTML response indicator
		},
	}
}

// IsRetryableError checks if the given error is retryable based on defined patterns.
func IsRetryableError(err error, patterns []string) bool {
	if err == nil {
		return false
	}
	errMsg := strings.ToLower(err.Error())
	for _, pattern := range patterns {
		if strings.Contains(errMsg, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

// WithRetry executes the given operation with retries based on the provided config.
func WithRetry[T any](ctx context.Context, operation RetryableFunc[T], config RetryConfig) (T, error) {
	var lastErr error
	var zero T

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		// Create a context with timeout for this specific attempt
		attemptCtx, cancel := context.WithTimeout(ctx, config.Timeout)

		result, err := operation(attemptCtx)
		cancel() // Release resources as soon as operation is done

		if err == nil {
			return result, nil
		}

		lastErr = err

		// If this was the last attempt, don't wait or check retryability
		if attempt == config.MaxAttempts {
			break
		}

		// Check if the error is retryable
		if !IsRetryableError(err, config.RetryableErrors) {
			break
		}

		// Calculate backoff delay: InitialDelay * (BackoffFactor ^ (attempt - 1))
		backoff := float64(config.InitialDelay) * math.Pow(config.BackoffFactor, float64(attempt-1))
		delay := time.Duration(backoff)

		if delay > config.MaxDelay {
			delay = config.MaxDelay
		}

		// Add jitter (up to 10% of the delay)
		jitterRange := int64(delay) / 10
		if jitterRange > 0 {
			jitter := time.Duration(rand.Int63n(jitterRange))
			delay += jitter
		}

		// Wait for the delay or context cancellation
		select {
		case <-time.After(delay):
			// Continue to next attempt
		case <-ctx.Done():
			return zero, ctx.Err()
		}
	}

	return zero, lastErr
}
