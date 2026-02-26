package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/socialchef/remy/internal/httpclient"
	"github.com/socialchef/remy/internal/metrics"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"io"

	"net/http"
)

type chatRequest struct {
	Model          string          `json:"model"`
	Messages       []chatMessage   `json:"messages"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type embeddingRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type embeddingResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
}

func callOpenAIChat(ctx context.Context, apiKey, model, systemPrompt, userContent string, jsonMode bool) (string, error) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime).Seconds()
		attrs := []attribute.KeyValue{attribute.String("provider", "openai")}
		metrics.ExternalAPIDuration.Record(ctx, duration, metric.WithAttributes(attrs...))
		metrics.ExternalAPICallsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	}()

	req := chatRequest{
		Model: model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userContent},
		},
	}
	if jsonMode {
		req.ResponseFormat = &responseFormat{Type: "json_object"}
	}

	body, _ := json.Marshal(req)
	httpReq, err := http.NewRequestWithContext(httpclient.WithProvider(ctx, "OpenAI"), "POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpclient.InstrumentedClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("OpenAI API error: %s", string(respBody))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", err
	}

	if len(chatResp.Choices) == 0 {
		return "", ErrNoResponse
	}

	return chatResp.Choices[0].Message.Content, nil
}

func callOpenAIEmbedding(ctx context.Context, apiKey, model, text string) ([]float32, error) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime).Seconds()
		attrs := []attribute.KeyValue{attribute.String("provider", "openai")}
		metrics.ExternalAPIDuration.Record(ctx, duration, metric.WithAttributes(attrs...))
		metrics.ExternalAPICallsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	}()

	req := embeddingRequest{
		Model: model,
		Input: text,
	}

	body, _ := json.Marshal(req)
	httpReq, err := http.NewRequestWithContext(httpclient.WithProvider(ctx, "OpenAI"), "POST", "https://api.openai.com/v1/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpclient.InstrumentedClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("OpenAI API error: %s", string(respBody))
	}

	var embResp embeddingResponse
	if err := json.Unmarshal(respBody, &embResp); err != nil {
		return nil, err
	}

	if len(embResp.Data) == 0 {
		return nil, ErrNoEmbedding
	}

	embedding := make([]float32, len(embResp.Data[0].Embedding))
	for i, v := range embResp.Data[0].Embedding {
		embedding[i] = float32(v)
	}
	return embedding, nil
}
