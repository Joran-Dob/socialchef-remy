package transcription

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTranscribeVideo(t *testing.T) {
	// 1. Mock video server
	videoContent := "dummy-video-data"
	videoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(videoContent))
	}))
	defer videoServer.Close()

	// 2. Mock OpenAI server
	expectedTranscript := "This is a test transcript."
	openAIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			t.Errorf("Expected multipart/form-data content type, got %s", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			t.Errorf("Expected test-api-key, got %s", r.Header.Get("Authorization"))
		}

		// Verify multipart form
		err := r.ParseMultipartForm(10 << 20)
		if err != nil {
			t.Errorf("Failed to parse multipart form: %v", err)
		}
		if r.FormValue("model") != "gpt-4o-mini-transcribe" {
			t.Errorf("Expected model gpt-4o-mini-transcribe, got %s", r.FormValue("model"))
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			t.Errorf("Failed to get form file: %v", err)
		}
		defer file.Close()

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"text": "%s"}`, expectedTranscript)
	}))
	defer openAIServer.Close()

	// 3. Create client and test
	client := NewClient("test-api-key")
	client.baseURL = openAIServer.URL

	transcript, err := client.TranscribeVideo(context.Background(), videoServer.URL)
	if err != nil {
		t.Fatalf("TranscribeVideo failed: %v", err)
	}

	if transcript != expectedTranscript {
		t.Errorf("Expected transcript %s, got %s", expectedTranscript, transcript)
	}
}

func TestTranscribeVideo_FetchError(t *testing.T) {
	client := NewClient("test-api-key")
	
	// Use an invalid URL to trigger fetch error
	_, err := client.TranscribeVideo(context.Background(), "http://invalid-url-that-does-not-exist")
	if err == nil {
		t.Error("Expected error for invalid video URL, got nil")
	}
}

func TestTranscribeVideo_OpenAIError(t *testing.T) {
	// 1. Mock video server
	videoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("data"))
	}))
	defer videoServer.Close()

	// 2. Mock OpenAI server returning 500
	openAIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("OpenAI internal error"))
	}))
	defer openAIServer.Close()

	client := NewClient("test-api-key")
	client.baseURL = openAIServer.URL

	_, err := client.TranscribeVideo(context.Background(), videoServer.URL)
	if err == nil {
		t.Error("Expected error for OpenAI 500 response, got nil")
	}
	if !strings.Contains(err.Error(), "OpenAI API error (status 500)") {
		t.Errorf("Expected error message to contain 'OpenAI API error (status 500)', got %v", err)
	}
}

// Note: TestTranscribeVideo tests the fallback path since mock servers don't provide real video.
// When FFmpeg extraction fails (which it will with "dummy-video-data"), the test falls back to sending the video directly.
// This is correct behavior - the tests verify the fallback path works as expected.
// Real video files with actual audio would trigger the successful extraction path.
