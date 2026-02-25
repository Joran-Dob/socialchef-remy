package errors

import (
	"errors"
	"net/http"
	"testing"
)

func TestAppError_Error(t *testing.T) {
	err := &AppError{
		Message: "something went wrong",
	}
	if err.Error() != "something went wrong" {
		t.Errorf("expected 'something went wrong', got %v", err.Error())
	}

	wrappedErr := errors.New("underlying error")
	errWithWrap := &AppError{
		Message: "failed operation",
		Err:     wrappedErr,
	}
	expected := "failed operation: underlying error"
	if errWithWrap.Error() != expected {
		t.Errorf("expected %q, got %q", expected, errWithWrap.Error())
	}
}

func TestAppError_Code(t *testing.T) {
	err := &AppError{
		ErrorCode: "ERR_CODE_123",
	}
	if err.Code() != "ERR_CODE_123" {
		t.Errorf("expected ERR_CODE_123, got %v", err.Code())
	}
}

func TestAppError_IsRetryable(t *testing.T) {
	tests := []struct {
		name string
		err  *AppError
		want bool
	}{
		{
			name: "rate limit is retryable",
			err: &AppError{
				Type:       ErrorTypeRateLimit,
				StatusCode: http.StatusTooManyRequests,
			},
			want: true,
		},
		{
			name: "validation error is not retryable",
			err: &AppError{
				Type:       ErrorTypeValidation,
				StatusCode: http.StatusBadRequest,
			},
			want: false,
		},
		{
			name: "500 scraper error is retryable",
			err: &AppError{
				Type:       ErrorTypeScraper,
				StatusCode: http.StatusInternalServerError,
			},
			want: true,
		},
		{
			name: "404 not found is not retryable",
			err: &AppError{
				Type:       ErrorTypeNotFound,
				StatusCode: http.StatusNotFound,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.IsRetryable(); got != tt.want {
				t.Errorf("AppError.IsRetryable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewValidationError(t *testing.T) {
	err := NewValidationError("invalid input", "VALIDATION_FAILED", "Check your fields")
	if err.Type != ErrorTypeValidation {
		t.Errorf("expected TypeValidation, got %v", err.Type)
	}
	if err.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %v", err.StatusCode)
	}
	if err.RecoverySuggestion() != "Check your fields" {
		t.Errorf("expected 'Check your fields', got %v", err.RecoverySuggestion())
	}
}

func TestNewTranscriptionError(t *testing.T) {
	underlying := errors.New("ai failed")
	err := NewTranscriptionError("could not transcribe", "TRANSCRIPTION_FAILED", underlying)
	if err.Type != ErrorTypeTranscription {
		t.Errorf("expected TypeTranscription, got %v", err.Type)
	}
	if err.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %v", err.StatusCode)
	}
	if err.Err != underlying {
		t.Error("underlying error not correctly wrapped")
	}
}
