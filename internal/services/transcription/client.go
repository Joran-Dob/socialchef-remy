package transcription

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/socialchef/remy/internal/errors"
)


// Client handles video transcription using OpenAI's API
type Client struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new transcription client
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 3 * time.Minute,
		},
		baseURL: "https://api.openai.com/v1",
	}
}

// transcriptionResponse represents the response from OpenAI's transcription API
type transcriptionResponse struct {
	Text string `json:"text"`
}

// TranscribeVideo fetches a video from a URL and transcribes its audio using OpenAI
func (c *Client) TranscribeVideo(ctx context.Context, videoURL string) (string, error) {
	// 1. Fetch video from URL
	req, err := http.NewRequestWithContext(ctx, "GET", videoURL, nil)
	if err != nil {
		return "", errors.NewTranscriptionError("failed to create video fetch request", "FETCH_REQUEST_ERROR", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", errors.NewTranscriptionError("failed to fetch video", "VIDEO_FETCH_ERROR", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.NewTranscriptionError(fmt.Sprintf("failed to fetch video: status %d", resp.StatusCode), "VIDEO_FETCH_HTTP_ERROR", nil)
	}

	// 2. Prepare multipart form via pipe to avoid buffering in memory
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		defer pw.Close()
		defer writer.Close()

		part, err := writer.CreateFormFile("file", "video.mp4")
		if err != nil {
			return
		}

		if _, err := io.Copy(part, resp.Body); err != nil {
			return
		}

		if err := writer.WriteField("model", "gpt-4o-mini-transcribe"); err != nil {
			return
		}
	}()

	// 3. Send to OpenAI
	openAIReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/audio/transcriptions", pr)

	if err != nil {
		return "", errors.NewTranscriptionError("failed to create OpenAI request", "OPENAI_REQUEST_ERROR", err)
	}

	openAIReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	openAIReq.Header.Set("Content-Type", writer.FormDataContentType())

	openAIResp, err := c.httpClient.Do(openAIReq)
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
