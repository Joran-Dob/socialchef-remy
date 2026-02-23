package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

type ProgressUpdate struct {
	JobID   string `json:"job_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type ProgressBroadcaster struct {
	supabaseURL string
	serviceKey  string
	httpClient  *http.Client
	mu          sync.Mutex
}

func NewProgressBroadcaster(supabaseURL, serviceKey string) *ProgressBroadcaster {
	return &ProgressBroadcaster{
		supabaseURL: supabaseURL,
		serviceKey:  serviceKey,
		httpClient:  &http.Client{},
	}
}

func (b *ProgressBroadcaster) Broadcast(userID string, update ProgressUpdate) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	channel := fmt.Sprintf("user:%s:imports", userID)

	payload := map[string]interface{}{
		"channel": channel,
		"type":    "broadcast",
		"event":   "progress",
		"payload": update,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal broadcast payload: %w", err)
	}

	url := fmt.Sprintf("%s/rest/v1/rpc/broadcast", b.supabaseURL)

	req, err := http.NewRequestWithContext(context.Background(), "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+b.serviceKey)
	req.Header.Set("apikey", b.serviceKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to broadcast: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("broadcast failed with status %d", resp.StatusCode)
	}

	return nil
}

func (b *ProgressBroadcaster) BroadcastViaRealtime(userID string, update ProgressUpdate) error {
	return b.Broadcast(userID, update)
}
