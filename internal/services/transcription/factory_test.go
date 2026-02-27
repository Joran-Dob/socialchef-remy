package transcription

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/socialchef/remy/internal/config"
)

func TestFactory_Groq(t *testing.T) {
	cfg := config.TranscriptionConfig{
		Provider:        "groq",
		FallbackEnabled: false,
	}

	openAIKey := "test-openai-key"
	groqKey := "test-groq-key"

	provider := NewProvider(cfg, openAIKey, groqKey)

	if _, ok := provider.(*GroqProvider); !ok {
		t.Errorf("Expected GroqProvider, got %T", provider)
	}
}

func TestFactory_OpenAI(t *testing.T) {
	cfg := config.TranscriptionConfig{
		Provider:        "openai",
		FallbackEnabled: false,
	}

	openAIKey := "test-openai-key"
	groqKey := "test-groq-key"

	provider := NewProvider(cfg, openAIKey, groqKey)

	if _, ok := provider.(*OpenAIProvider); !ok {
		t.Errorf("Expected OpenAIProvider, got %T", provider)
	}
}

func TestFactory_Default(t *testing.T) {
	cfg := config.TranscriptionConfig{}

	openAIKey := "test-openai-key"
	groqKey := "test-groq-key"

	provider := NewProvider(cfg, openAIKey, groqKey)

	if _, ok := provider.(*GroqProvider); !ok {
		t.Errorf("Expected default GroqProvider, got %T", provider)
	}
}

func TestFactory_WithFallback(t *testing.T) {
	cfg := config.TranscriptionConfig{
		Provider:         "groq",
		FallbackEnabled:  true,
		FallbackProvider: "openai",
	}

	openAIKey := "test-openai-key"
	groqKey := "test-groq-key"

	provider := NewProvider(cfg, openAIKey, groqKey)

	if _, ok := provider.(*FallbackProvider); !ok {
		t.Errorf("Expected FallbackProvider, got %T", provider)
	}

	fallbackProvider := provider.(*FallbackProvider)
	if _, ok := fallbackProvider.primary.(*GroqProvider); !ok {
		t.Errorf("Expected primary to be GroqProvider, got %T", fallbackProvider.primary)
	}

	if _, ok := fallbackProvider.secondary.(*OpenAIProvider); !ok {
		t.Errorf("Expected secondary to be OpenAIProvider, got %T", fallbackProvider.secondary)
	}
}

func TestFallbackWrapper_PrimaryFails(t *testing.T) {
	tempFile := createTempAudioFileForFactory(t)
	defer os.Remove(tempFile)

	primary := &mockProviderForFactory{
		failWithStatus: http.StatusInternalServerError,
		failWithError:  true,
	}

	secondary := &mockProviderForFactory{
		response:      "Secondary transcription success",
		failWithError: false,
	}

	fallback := NewFallbackProvider(primary, secondary)

	result, err := fallback.Transcribe(context.Background(), tempFile)
	if err != nil {
		t.Fatalf("Transcription failed: %v", err)
	}

	expected := "Secondary transcription success"
	if result != expected {
		t.Errorf("Expected result '%s', got '%s'", expected, result)
	}
}

func TestFallbackWrapper_ClientError(t *testing.T) {
	tempFile := createTempAudioFileForFactory(t)
	defer os.Remove(tempFile)

	primary := &mockProviderForFactory{
		failWithStatus: http.StatusBadRequest,
		failWithError:  true,
	}

	secondary := &mockProviderForFactory{
		response:      "Secondary transcription success",
		failWithError: false,
	}

	fallback := NewFallbackProvider(primary, secondary)

	_, err := fallback.Transcribe(context.Background(), tempFile)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !strings.Contains(err.Error(), "400") {
		t.Errorf("Expected client error (4xx), got: %v", err)
	}
}

func TestFallbackWrapper_PrimarySucceeds(t *testing.T) {
	tempFile := createTempAudioFileForFactory(t)
	defer os.Remove(tempFile)

	primary := &mockProviderForFactory{
		response:      "Primary transcription success",
		failWithError: false,
	}

	secondary := &mockProviderForFactory{
		response:      "Secondary transcription success",
		failWithError: false,
	}

	fallback := NewFallbackProvider(primary, secondary)

	result, err := fallback.Transcribe(context.Background(), tempFile)
	if err != nil {
		t.Fatalf("Transcription failed: %v", err)
	}

	expected := "Primary transcription success"
	if result != expected {
		t.Errorf("Expected result '%s', got '%s'", expected, result)
	}
}

func TestFallbackWrapper_BothFail(t *testing.T) {
	tempFile := createTempAudioFileForFactory(t)
	defer os.Remove(tempFile)

	primary := &mockProviderForFactory{
		failWithStatus: http.StatusInternalServerError,
		failWithError:  true,
	}

	secondary := &mockProviderForFactory{
		failWithStatus: http.StatusInternalServerError,
		failWithError:  true,
	}

	fallback := NewFallbackProvider(primary, secondary)

	_, err := fallback.Transcribe(context.Background(), tempFile)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !strings.Contains(err.Error(), "both primary and secondary providers failed") {
		t.Errorf("Expected both providers failed error, got: %v", err)
	}
}

type mockProviderForFactory struct {
	response       string
	failWithError  bool
	failWithStatus int
}

func (m *mockProviderForFactory) Transcribe(ctx context.Context, audioPath string) (string, error) {
	if m.failWithError {
		errMsg := fmt.Sprintf("Mock error with status %d", m.failWithStatus)
		return "", errors.New(errMsg)
	}
	return m.response, nil
}

func createTempAudioFileForFactory(t *testing.T) string {
	tempDir := t.TempDir()
	tempFile := fmt.Sprintf("%s/audio.mp3", tempDir)
	content := "dummy audio content for testing"

	err := os.WriteFile(tempFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	return tempFile
}
