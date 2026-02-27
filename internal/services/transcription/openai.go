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

// OpenAIProvider implements the TranscriptionProvider interface for OpenAI
type OpenAIProvider struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
}

// NewOpenAIProvider creates a new OpenAI transcription provider
func NewOpenAIProvider(apiKey string) *OpenAIProvider {
	return &OpenAIProvider{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 3 * time.Minute,
		},
		baseURL: "https://api.openai.com/v1",
	}
}

// Transcribe transcribes an audio file using OpenAI's transcription API
func (p *OpenAIProvider) Transcribe(ctx context.Context, audioPath string) (string, error) {
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
		if err := writer.WriteField("model", "gpt-4o-mini-transcribe"); err != nil {
			return
		}
	}()

	// 3. Send to OpenAI
	openAIReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/audio/transcriptions", pr)
	if err != nil {
		return "", errors.NewTranscriptionError("failed to create OpenAI request", "OPENAI_REQUEST_ERROR", err)
	}

	openAIReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	openAIReq.Header.Set("Content-Type", writer.FormDataContentType())

	openAIResp, err := p.httpClient.Do(openAIReq)
	if err != nil {
		return "", errors.NewTranscriptionError("failed to call OpenAI transcription API", "OPENAI_API_ERROR", err)
	}
	defer openAIResp.Body.Close()

	// 4. Parse response
	respBody, err := io.ReadAll(openAIResp.Body)
	if err != nil {
		return "", errors.NewTranscriptionError("failed to read OpenAI response", "READ_RESPONSE_ERROR", err)
	}

	if openAIResp.StatusCode != http.StatusOK {
		return "", errors.NewTranscriptionError(fmt.Sprintf("OpenAI API error (status %d): %s", openAIResp.StatusCode, string(respBody)), "OPENAI_API_HTTP_ERROR", nil)
	}

	var transResp transcriptionResponse
	if err := json.Unmarshal(respBody, &transResp); err != nil {
		return "", errors.NewTranscriptionError("failed to parse OpenAI response", "PARSE_RESPONSE_ERROR", err)
	}

	return transResp.Text, nil
}

