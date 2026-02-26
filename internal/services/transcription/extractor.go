package transcription

import (
	"context"
	"os"
	"os/exec"

	"github.com/socialchef/remy/internal/errors"
)

// ExtractAudio extracts audio from a video file using FFmpeg
func ExtractAudio(ctx context.Context, videoPath string) (audioPath string, err error) {
	// Create a temporary file for the audio output
	tempFile, err := os.CreateTemp("", "audio-*.mp3")
	if err != nil {
		return "", errors.NewTranscriptionError("failed to create temp file", "AUDIO_EXTRACTION_ERROR", err)
	}
	defer tempFile.Close()

	audioPath = tempFile.Name()

	// Prepare the FFmpeg command
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", videoPath,
		"-vn",
		"-acodec", "libmp3lame",
		"-ab", "64k",
		"-y", audioPath,
	)


	// Run the command
	if err := cmd.Run(); err != nil {
		// Clean up the temp file if the command failed
		os.Remove(audioPath)
		return "", errors.NewTranscriptionError("failed to extract audio with FFmpeg", "AUDIO_EXTRACTION_ERROR", err)
	}
	return audioPath, nil

}
