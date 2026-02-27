package transcription

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"github.com/socialchef/remy/internal/errors"
)

// ProviderAdapter adapts a TranscriptionProvider to implement the TranscriptionClient interface
type ProviderAdapter struct {
	provider TranscriptionProvider
}

// NewProviderAdapter creates a new ProviderAdapter that wraps a TranscriptionProvider
func NewProviderAdapter(provider TranscriptionProvider) *ProviderAdapter {
	return &ProviderAdapter{
		provider: provider,
	}
}

// TranscribeVideo implements the TranscriptionClient interface by downloading a video,
// extracting audio, and using the wrapped provider to transcribe the audio
func (a *ProviderAdapter) TranscribeVideo(ctx context.Context, videoURL string) (string, error) {
	// 1. Fetch video from URL
	req, err := http.NewRequestWithContext(ctx, "GET", videoURL, nil)
	if err != nil {
		return "", errors.NewTranscriptionError("failed to create video fetch request", "FETCH_REQUEST_ERROR", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", errors.NewTranscriptionError("failed to fetch video", "VIDEO_FETCH_ERROR", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.NewTranscriptionError(fmt.Sprintf("failed to fetch video: status %d", resp.StatusCode), "VIDEO_FETCH_HTTP_ERROR", nil)
	}

	// Save video to temp file
	videoFile, err := os.CreateTemp("", "video-*.mp4")
	if err != nil {
		return "", errors.NewTranscriptionError("failed to create temp video file", "VIDEO_TEMP_FILE_ERROR", err)
	}
	defer videoFile.Close()
	defer os.Remove(videoFile.Name())

	if _, err := io.Copy(videoFile, resp.Body); err != nil {
		return "", errors.NewTranscriptionError("failed to save video to temp file", "VIDEO_SAVE_ERROR", err)
	}
	videoPath := videoFile.Name()

	// Try to extract audio from video
	audioPath, err := ExtractAudio(ctx, videoPath)
	if err != nil {
		return "", errors.NewTranscriptionError("failed to extract audio from video", "AUDIO_EXTRACTION_ERROR", err)
	}

	defer os.Remove(audioPath)
	return a.provider.Transcribe(ctx, audioPath)
}
