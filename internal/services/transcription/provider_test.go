package transcription

import (
	"context"
	"errors"
	"testing"
)

// mockProvider implements the TranscriptionProvider interface for testing
type mockProvider struct {
	transcription string
	err           error
}

func (m *mockProvider) Transcribe(ctx context.Context, audioPath string) (string, error) {
	return m.transcription, m.err
}

func TestProviderInterface(t *testing.T) {
	// Test that mockProvider satisfies the interface
	var _ TranscriptionProvider = &mockProvider{}

	// Test successful transcription
	successProvider := &mockProvider{
		transcription: "This is a test transcription",
		err:           nil,
	}

	result, err := successProvider.Transcribe(context.Background(), "/tmp/test.mp3")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != "This is a test transcription" {
		t.Errorf("Expected transcription 'This is a test transcription', got '%s'", result)
	}

	// Test transcription error
	errorProvider := &mockProvider{
		transcription: "",
		err:           errors.New("transcription failed"),
	}

	_, err = errorProvider.Transcribe(context.Background(), "/tmp/test.mp3")
	if err == nil {
		t.Error("Expected error, got none")
	} else if err.Error() != "transcription failed" {
		t.Errorf("Expected error message 'transcription failed', got '%s'", err.Error())
	}
}
