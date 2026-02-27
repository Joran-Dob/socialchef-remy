package transcription

import (
	"context"

	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGroqProvider(t *testing.T) {
	// Create a temporary audio file
	tempFile := createTempAudioFile(t)
	defer os.Remove(tempFile)

	// Create test server that simulates Groq API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request method and path
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/audio/transcriptions") {
			t.Errorf("Expected path ending with /audio/transcriptions, got %s", r.URL.Path)
		}

		// Verify content type is multipart
		contentType := r.Header.Get("Content-Type")
		if !strings.Contains(contentType, "multipart/form-data") {
			t.Errorf("Expected multipart form data, got %s", contentType)
		}

		// Verify Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test-api-key" {
			t.Errorf("Expected Authorization header 'Bearer test-api-key', got '%s'", authHeader)
		}

		// Read and verify multipart form
		err := r.ParseMultipartForm(10 << 20) // 10MB max
		if err != nil {
			t.Errorf("Failed to parse multipart form: %v", err)
			return
		}

		// Check model field
		model := r.FormValue("model")
		if model != "whisper-large-v3-turbo" {
			t.Errorf("Expected model 'whisper-large-v3-turbo', got '%s'", model)
		}

		// Check that file was uploaded
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Errorf("Failed to get file from form: %v", err)
			return
		}
		defer file.Close()

		if header.Filename != "audio.mp3" {
			t.Errorf("Expected filename 'audio.mp3', got '%s'", header.Filename)
		}

		// Read file content to verify it's not empty
		fileContent, err := io.ReadAll(file)
		if err != nil {
			t.Errorf("Failed to read file content: %v", err)
			return
		}
		if len(fileContent) == 0 {
			t.Error("File content is empty")
		}

		// Return successful response
		response := `{"text": "This is a test transcription from Groq"}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}))
	defer server.Close()

	// Create GroqProvider with test server URL
	provider := NewGroqProvider("test-api-key")
	provider.baseURL = server.URL

	// Test transcription
	result, err := provider.Transcribe(context.Background(), tempFile)
	if err != nil {
		t.Fatalf("Transcribe failed: %v", err)
	}

	expected := "This is a test transcription from Groq"
	if result != expected {
		t.Errorf("Expected transcription '%s', got '%s'", expected, result)
	}
}

func TestGroqProvider_Error(t *testing.T) {
	// Create a temporary audio file
	tempFile := createTempAudioFile(t)
	defer os.Remove(tempFile)

	// Create test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := `{"error": {"message": "Rate limit exceeded"}}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(response))
	}))
	defer server.Close()

	// Create GroqProvider with test server URL
	provider := NewGroqProvider("test-api-key")
	provider.baseURL = server.URL

	// Test transcription with error
	_, err := provider.Transcribe(context.Background(), tempFile)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Verify error type
	if !strings.Contains(err.Error(), "Groq API error") {
		t.Errorf("Expected Groq API error in response, got: %v", err)
	}
}

func TestGroqProvider_FileOpenError(t *testing.T) {
	// Create GroqProvider
	provider := NewGroqProvider("test-api-key")

	// Test with non-existent file
	_, err := provider.Transcribe(context.Background(), "/nonexistent/file.mp3")
	if err == nil {
		t.Fatal("Expected error for non-existent file, got nil")
	}

	// Verify error type
	if !strings.Contains(err.Error(), "failed to open audio file") {
		t.Errorf("Expected file open error, got: %v", err)
	}
}

func createTempAudioFile(t *testing.T) string {
	// Create a temporary directory
	tempDir := t.TempDir()

	// Create a dummy audio file
	tempFile := filepath.Join(tempDir, "audio.mp3")
	content := "dummy audio content for testing"

	err := os.WriteFile(tempFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	return tempFile
}

func TestGroqProvider_Timeout(t *testing.T) {
	// Create a temporary audio file
	tempFile := createTempAudioFile(t)
	defer os.Remove(tempFile)

	// Create a slow server that times out
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second) // This should timeout
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"text": "success"}`))
	}))
	defer server.Close()

	// Create GroqProvider with short timeout
	provider := NewGroqProvider("test-api-key")
	provider.baseURL = server.URL
	provider.httpClient = &http.Client{
		Timeout: 100 * time.Millisecond, // Very short timeout
	}

	// Test transcription - should timeout
	start := time.Now()
	_, err := provider.Transcribe(context.Background(), tempFile)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Expected timeout error, got nil")
	}

	// Should timeout quickly
	if elapsed > 2*time.Second {
		t.Errorf("Expected quick timeout, took %v", elapsed)
	}
}
