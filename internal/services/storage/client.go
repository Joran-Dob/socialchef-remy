package storage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"github.com/google/uuid"

)

type Client struct {
	supabaseURL string
	serviceKey  string
	httpClient  *http.Client
}

var ErrUploadFailed = errors.New("upload failed")

func NewClient(supabaseURL, serviceKey string) *Client {
	return &Client{
		supabaseURL: supabaseURL,
		serviceKey:  serviceKey,
		httpClient:  &http.Client{},
	}
}

func HashContent(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

type uploadResponse struct {
	Key string `json:"Key"`
	Id  string `json:"Id"`
}

func (c *Client) UploadImage(ctx context.Context, bucket, path string, data []byte, contentType string) (string, error) {
	uploadURL := fmt.Sprintf("%s/storage/v1/object/%s/%s", c.supabaseURL, bucket, path)

	req, err := http.NewRequestWithContext(ctx, "POST", uploadURL, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.serviceKey)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("x-upsert", "true")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("%w: %s", ErrUploadFailed, string(body))
	}

	var result uploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return c.GetPublicURL(bucket, path), nil
}

func (c *Client) GetPublicURL(bucket, path string) string {
	return fmt.Sprintf("%s/storage/v1/object/public/%s/%s", c.supabaseURL, bucket, path)
}

type existingImageResponse struct {
	ID           string `json:"id"`
	ContentHash  string `json:"content_hash"`
	StoragePath  string `json:"storage_path"`
}

func (c *Client) GetImageByHash(ctx context.Context, hash string) (*existingImageResponse, error) {
	url := fmt.Sprintf("%s/rest/v1/stored_images?content_hash=eq.%s&select=*", c.supabaseURL, hash)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.serviceKey)
	req.Header.Set("apikey", c.serviceKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var results []existingImageResponse
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, nil
	}

	return &results[0], nil
}

func (c *Client) CreateStoredImageRecord(ctx context.Context, id, hash, storagePath, sourceURL string) error {
	url := fmt.Sprintf("%s/rest/v1/stored_images", c.supabaseURL)

	body := map[string]string{
		"id":           id,
		"content_hash": hash,
		"storage_path": storagePath,
		"source_url":   sourceURL,
	}
	data, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.serviceKey)
	req.Header.Set("apikey", c.serviceKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=minimal")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create image record: %s", string(body))
	}

	return nil
}

func (c *Client) UploadImageWithHash(ctx context.Context, bucket, path, sourceURL string, data []byte) (string, error) {
	hash := HashContent(data)

	existing, err := c.GetImageByHash(ctx, hash)
	if err != nil {
		return "", err
	}
	if existing != nil {
		return c.GetPublicURL(bucket, existing.StoragePath), nil
	}

	contentType := "image/jpeg"
	if len(data) > 4 && string(data[:4]) == "\x89PNG" {
		contentType = "image/png"
	}

	publicURL, err := c.UploadImage(ctx, bucket, path, data, contentType)
	if err != nil {
		return "", err
	}

	imageID := generateUUID()
	if err := c.CreateStoredImageRecord(ctx, imageID, hash, path, sourceURL); err != nil {
		return "", err
	}

	return publicURL, nil
}

func generateUUID() string {
	return uuid.New().String()
}
