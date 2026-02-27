package transcription

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	"github.com/socialchef/remy/internal/errors"
)

// GroqProvider implements the TranscriptionProvider interface for Groq
type GroqProvider struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
}

// NewGroqProvider creates a new Groq transcription provider
func NewGroqProvider(apiKey string) *GroqProvider {
	return &GroqProvider{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 3 * time.Minute,
		},
		baseURL: "https://api.groq.com/openai/v1",
	}
}

// Transcribe transcribes an audio file using Groq's transcription API
func (p *GroqProvider) Transcribe(ctx context.Context, audioPath string) (string, error) {
	// 1. Open audio file
	audioFile, err := os.Open(audioPath)
	if err != nil {
		return "", errors.NewTranscriptionError("failed to open audio file", "AUDIO_FILE_ERROR", err)
	}
	defer audioFile.Close()

	// 2. Prepare multipart form via pipe to avoid buffering in memory
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		defer pw.Close()
		defer writer.Close()

		// Create form file
		part, err := writer.CreateFormFile("file", "audio.mp3")
		if err != nil {
			return
		}

		// Copy audio file content
		if _, err := io.Copy(part, audioFile); err != nil {
			return
		}

		// Add model field
		if err := writer.WriteField("model", "whisper-large-v3-turbo"); err != nil {
			return
		}
	}()

	// 3. Send to Groq
	groqReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/audio/transcriptions", pr)
	if err != nil {
		return "", errors.NewTranscriptionError("failed to create Groq request", "GROQ_REQUEST_ERROR", err)
	}

	groqReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	groqReq.Header.Set("Content-Type", writer.FormDataContentType())

	groqResp, err := p.httpClient.Do(groqReq)
	if err != nil {
		return "", errors.NewTranscriptionError("failed to call Groq transcription API", "GROQ_API_ERROR", err)
	}
	defer groqResp.Body.Close()

	// 4. Parse response
	respBody, err := io.ReadAll(groqResp.Body)
	if err != nil {
		return "", errors.NewTranscriptionError("failed to read Groq response", "READ_RESPONSE_ERROR", err)
	}

	if groqResp.StatusCode != http.StatusOK {
		return "", errors.NewTranscriptionError(fmt.Sprintf("Groq API error (status %d): %s", groqResp.StatusCode, string(respBody)), "GROQ_API_HTTP_ERROR", nil)
	}

	var transResp transcriptionResponse
	if err := json.Unmarshal(respBody, &transResp); err != nil {
		return "", errors.NewTranscriptionError("failed to parse Groq response", "PARSE_RESPONSE_ERROR", err)
	}

	return transResp.Text, nil
}


