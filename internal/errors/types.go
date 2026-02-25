package errors

import (
	"fmt"
	"net/http"
)

// ErrorType defines the category of the error
type ErrorType string

const (
	ErrorTypeValidation       ErrorType = "VALIDATION_ERROR"
	ErrorTypeTranscription    ErrorType = "TRANSCRIPTION_ERROR"
	ErrorTypeScraper          ErrorType = "SCRAPER_ERROR"
	ErrorTypeRecipeGeneration ErrorType = "RECIPE_GENERATION_ERROR"
	ErrorTypeRateLimit        ErrorType = "RATE_LIMIT_ERROR"
	ErrorTypeNotFound         ErrorType = "NOT_FOUND_ERROR"
	ErrorTypeInternal         ErrorType = "INTERNAL_ERROR"
)

// AppError represents a structured error for the application
type AppError struct {
	Type          ErrorType `json:"type"`
	Message       string    `json:"message"`
	StatusCode    int       `json:"statusCode"`
	ErrorCode     string    `json:"errorCode"`
	IsOperational bool      `json:"isOperational"`
	Recovery      string    `json:"recoverySuggestion,omitempty"`
	Err           error     `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Code returns the application-specific error code
func (e *AppError) Code() string {
	return e.ErrorCode
}

// RecoverySuggestion returns the suggestion on how to recover from the error
func (e *AppError) RecoverySuggestion() string {
	return e.Recovery
}

// IsRetryable determines if the operation that caused the error should be retried
func (e *AppError) IsRetryable() bool {
	switch e.Type {
	case ErrorTypeRateLimit:
		return true
	case ErrorTypeScraper, ErrorTypeTranscription, ErrorTypeRecipeGeneration:
		// These might be retryable depending on the underlying cause,
		// but usually 5xx errors are worth retrying
		return e.StatusCode >= 500
	default:
		return false
	}
}

// NewValidationError creates a new validation error (400)
func NewValidationError(message string, errorCode string, suggestion string) *AppError {
	return &AppError{
		Type:          ErrorTypeValidation,
		Message:       message,
		StatusCode:    http.StatusBadRequest,
		ErrorCode:     errorCode,
		IsOperational: true,
		Recovery:      suggestion,
	}
}

// NewNotFoundError creates a new not found error (404)
func NewNotFoundError(message string, errorCode string, suggestion string) *AppError {
	return &AppError{
		Type:          ErrorTypeNotFound,
		Message:       message,
		StatusCode:    http.StatusNotFound,
		ErrorCode:     errorCode,
		IsOperational: true,
		Recovery:      suggestion,
	}
}

// NewRateLimitError creates a new rate limit error (429)
func NewRateLimitError(message string, errorCode string, suggestion string) *AppError {
	return &AppError{
		Type:          ErrorTypeRateLimit,
		Message:       message,
		StatusCode:    http.StatusTooManyRequests,
		ErrorCode:     errorCode,
		IsOperational: true,
		Recovery:      suggestion,
	}
}

// NewTranscriptionError creates a new transcription error (500)
func NewTranscriptionError(message string, errorCode string, err error) *AppError {
	return &AppError{
		Type:          ErrorTypeTranscription,
		Message:       message,
		StatusCode:    http.StatusInternalServerError,
		ErrorCode:     errorCode,
		IsOperational: true,
		Recovery:      "Try providing a clearer video or audio source.",
		Err:           err,
	}
}

// NewScraperError creates a new scraper error (500)
func NewScraperError(message string, errorCode string, err error) *AppError {
	return &AppError{
		Type:          ErrorTypeScraper,
		Message:       message,
		StatusCode:    http.StatusInternalServerError,
		ErrorCode:     errorCode,
		IsOperational: true,
		Recovery:      "Verify the URL is accessible and try again later.",
		Err:           err,
	}
}

// NewRecipeGenerationError creates a new recipe generation error (500)
func NewRecipeGenerationError(message string, errorCode string, err error) *AppError {
	return &AppError{
		Type:          ErrorTypeRecipeGeneration,
		Message:       message,
		StatusCode:    http.StatusInternalServerError,
		ErrorCode:     errorCode,
		IsOperational: true,
		Recovery:      "Try adjusting the input parameters or wait for the service to be available.",
		Err:           err,
	}
}
