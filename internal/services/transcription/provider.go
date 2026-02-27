package transcription

import (
	"context"
)

type ProviderType string

const (
	ProviderGroq   ProviderType = "groq"
	ProviderOpenAI ProviderType = "openai"
)

type TranscriptionProvider interface {
	Transcribe(ctx context.Context, audioPath string) (string, error)
}
