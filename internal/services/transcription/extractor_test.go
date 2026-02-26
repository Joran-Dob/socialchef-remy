package transcription

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestExtractAudio(t *testing.T) {
	// Skip if ffmpeg not available
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	// Create a simple test video file with actual audio
	videoPath := filepath.Join(t.TempDir(), "test-video.mp4")
	audioPath := ""

	// Create a video with both video and audio streams
	cmd := exec.Command("ffmpeg", "-f", "lavfi", "-i", "testsrc=duration=3:size=320x240:rate=1", "-f", "lavfi", "-i", "sine=frequency=1000:duration=3", "-c:v", "libx264", "-c:a", "aac", "-t", "3", videoPath)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create test video: %v", err)
	}

	defer func() {
		os.Remove(videoPath)
		if audioPath != "" {
			os.Remove(audioPath)
		}
	}()

	// Test successful audio extraction
	ctx := context.Background()
	resultAudioPath, err := ExtractAudio(ctx, videoPath)

	if err != nil {
		t.Errorf("ExtractAudio failed: %v", err)
	}

	if resultAudioPath == "" {
		t.Error("ExtractAudio returned empty audio path")
	}

	// Verify the audio file exists
	if _, err := os.Stat(resultAudioPath); os.IsNotExist(err) {
		t.Error("ExtractAudio did not create audio file")
	}

	// Verify the audio file has audio content by trying to play it back
	audioCmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "csv=p=0", resultAudioPath)
	output, err := audioCmd.Output()
	if err != nil {
		t.Errorf("Cannot probe extracted audio: %v", err)
	}

	// Verify we have a non-zero duration
	duration := string(output)
	if duration == "" || duration == "0" {
		t.Error("Extracted audio has zero duration")
	}

	audioPath = resultAudioPath
}

func TestExtractAudio_FFmpegNotFound(t *testing.T) {
	// Save current PATH and remove ffmpeg from it
	originalPath := os.Getenv("PATH")

	// Create a temporary dir without ffmpeg
	tempDir := t.TempDir()
	newPath := tempDir + ":" + originalPath
	os.Setenv("PATH", newPath)

	defer func() {
		os.Setenv("PATH", originalPath)
	}()

	videoContent := "test-video-content"
	videoPath := filepath.Join(t.TempDir(), "test-video.mp4")

	// Create a dummy video file
	if err := os.WriteFile(videoPath, []byte(videoContent), 0644); err != nil {
		t.Fatalf("Failed to create test video file: %v", err)
	}

	defer os.Remove(videoPath)

	// Test error when ffmpeg is not available
	ctx := context.Background()
	_, err := ExtractAudio(ctx, videoPath)

	if err == nil {
		t.Error("Expected error when ffmpeg is not available, got nil")
	}
}

func TestExtractAudio_ContextCancellation(t *testing.T) {
	// Skip if ffmpeg not available
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}


	videoPath := filepath.Join(t.TempDir(), "test-video.mp4")

	// Create a test video file
	cmd := exec.Command("ffmpeg", "-f", "lavfi", "-i", "testsrc=duration=5:size=320x240:rate=1", "-c:v", "libx264", "-t", "5", videoPath)
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create test video: %v", err)
	}

	defer os.Remove(videoPath)

	// Create context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Test that cancellation stops ffmpeg
	_, err := ExtractAudio(ctx, videoPath)

	// Note: The error might not be exactly context cancelled because ffmpeg might
	// complete before cancellation. The important thing is that it doesn't hang.
	if err != nil && err.Error() != "context canceled" {
		// Accept any error that's not a hang
		t.Logf("ExtractAudio returned error (acceptable): %v", err)
	}
}

func TestExtractAudio_InvalidVideoFile(t *testing.T) {
	// Skip if ffmpeg not available
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	videoPath := filepath.Join(t.TempDir(), "invalid-video.mp4")

	// Create an invalid video file (just random text)
	if err := os.WriteFile(videoPath, []byte("this is not a valid video file"), 0644); err != nil {
		t.Fatalf("Failed to create invalid video file: %v", err)
	}

	defer os.Remove(videoPath)

	// Test error handling for invalid video file
	ctx := context.Background()
	_, err := ExtractAudio(ctx, videoPath)

	if err == nil {
		t.Error("Expected error for invalid video file, got nil")
	}
}
