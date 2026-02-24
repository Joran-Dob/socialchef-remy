package utils

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestIsRetryableError(t *testing.T) {
	patterns := []string{"timeout", "rate limit"}

	tests := []struct {
		err      error
		expected bool
	}{
		{errors.New("request timeout"), true},
		{errors.New("Rate Limit exceeded"), true},
		{errors.New("not found"), false},
		{nil, false},
	}

	for _, tt := range tests {
		if got := IsRetryableError(tt.err, patterns); got != tt.expected {
			t.Errorf("IsRetryableError(%v) = %v; want %v", tt.err, got, tt.expected)
		}
	}
}

func TestWithRetry_Success(t *testing.T) {
	ctx := context.Background()
	config := DefaultRetryConfig()
	config.InitialDelay = 1 * time.Millisecond // fast tests

	attempts := 0
	operation := func(ctx context.Context) (string, error) {
		attempts++
		return "success", nil
	}

	result, err := WithRetry(ctx, operation, config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if result != "success" {
		t.Errorf("Expected result 'success', got %v", result)
	}
	if attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", attempts)
	}
}

func TestWithRetry_RetrySuccess(t *testing.T) {
	ctx := context.Background()
	config := DefaultRetryConfig()
	config.InitialDelay = 1 * time.Millisecond
	config.MaxAttempts = 3

	attempts := 0
	operation := func(ctx context.Context) (string, error) {
		attempts++
		if attempts < 3 {
			return "", errors.New("timeout error")
		}
		return "success", nil
	}

	result, err := WithRetry(ctx, operation, config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if result != "success" {
		t.Errorf("Expected result 'success', got %v", result)
	}
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestWithRetry_MaxAttemptsExhausted(t *testing.T) {
	ctx := context.Background()
	config := DefaultRetryConfig()
	config.InitialDelay = 1 * time.Millisecond
	config.MaxAttempts = 3

	attempts := 0
	expectedErr := errors.New("persistent timeout")
	operation := func(ctx context.Context) (string, error) {
		attempts++
		return "", expectedErr
	}

	_, err := WithRetry(ctx, operation, config)
	if err != expectedErr {
		t.Fatalf("Expected error %v, got %v", expectedErr, err)
	}
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestWithRetry_NonRetryableError(t *testing.T) {
	ctx := context.Background()
	config := DefaultRetryConfig()
	config.InitialDelay = 1 * time.Millisecond

	attempts := 0
	expectedErr := errors.New("fatal error")
	operation := func(ctx context.Context) (string, error) {
		attempts++
		return "", expectedErr
	}

	_, err := WithRetry(ctx, operation, config)
	if err != expectedErr {
		t.Fatalf("Expected error %v, got %v", expectedErr, err)
	}
	if attempts != 1 {
		t.Errorf("Expected 1 attempt for non-retryable error, got %d", attempts)
	}
}

func TestWithRetry_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	config := DefaultRetryConfig()
	config.InitialDelay = 50 * time.Millisecond

	attempts := 0
	operation := func(ctx context.Context) (string, error) {
		attempts++
		return "", errors.New("timeout")
	}

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	_, err := WithRetry(ctx, operation, config)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Expected context.Canceled, got %v", err)
	}
}

func TestWithRetry_TimeoutPerAttempt(t *testing.T) {
	ctx := context.Background()
	config := DefaultRetryConfig()
	config.Timeout = 10 * time.Millisecond
	config.InitialDelay = 1 * time.Millisecond

	operation := func(ctx context.Context) (string, error) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(50 * time.Millisecond):
			return "done", nil
		}
	}

	_, err := WithRetry(ctx, operation, config)
	if err == nil || !strings.Contains(err.Error(), "deadline exceeded") {
		t.Fatalf("Expected deadline exceeded error, got %v", err)
	}
}

func TestWithRetry_BackoffTiming(t *testing.T) {
	ctx := context.Background()
	config := DefaultRetryConfig()
	config.InitialDelay = 10 * time.Millisecond
	config.BackoffFactor = 2.0
	config.MaxAttempts = 3

	start := time.Now()
	_, _ = WithRetry(ctx, func(ctx context.Context) (string, error) {
		return "", errors.New("timeout")
	}, config)
	elapsed := time.Since(start)

	// Attempt 1: fail, wait 10ms
	// Attempt 2: fail, wait 20ms
	// Attempt 3: fail, return
	// Total wait should be at least 30ms
	if elapsed < 30*time.Millisecond {
		t.Errorf("Expected at least 30ms elapsed for backoff, got %v", elapsed)
	}
}
