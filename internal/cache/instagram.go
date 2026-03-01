package cache

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// CachedPost represents a cached Instagram post.
type CachedPost struct {
	ID            string `json:"id"`
	Caption       string `json:"caption"`
	ImageURL      string `json:"image_url"`
	VideoURL      string `json:"video_url"`
	OwnerUsername string `json:"owner_username"`
	OwnerName     string `json:"owner_name"`
	OwnerAvatar   string `json:"owner_avatar"`
	OwnerID       string `json:"owner_id"`
}

// InstagramCache provides Redis-backed caching for Instagram post data.
type InstagramCache struct {
	client *redis.Client
	prefix string
}

// NewInstagramCache creates a new Instagram cache with the given Redis client.
func NewInstagramCache(client *redis.Client) *InstagramCache {
	return &InstagramCache{
		client: client,
		prefix: "instagram:",
	}
}

// makeKey creates a cache key from a URL by hashing it.
func (c *InstagramCache) makeKey(url string) string {
	hash := sha256.Sum256([]byte(url))
	return fmt.Sprintf("%s%x", c.prefix, hash)
}

// Get retrieves a cached Instagram post by URL.
func (c *InstagramCache) Get(ctx context.Context, url string) (*CachedPost, error) {
	if c.client == nil {
		return nil, nil
	}

	key := c.makeKey(url)
	data, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		slog.Warn("Redis cache get failed", "error", err)
		return nil, nil
	}

	var post CachedPost
	if err := json.Unmarshal([]byte(data), &post); err != nil {
		slog.Warn("Failed to unmarshal cached post", "error", err)
		return nil, nil
	}

	return &post, nil
}

// Set stores an Instagram post in the cache with the given TTL.
func (c *InstagramCache) Set(ctx context.Context, url string, post *CachedPost, ttl time.Duration) error {
	if c.client == nil {
		return nil
	}

	data, err := json.Marshal(post)
	if err != nil {
		return err
	}

	key := c.makeKey(url)
	if err := c.client.Set(ctx, key, data, ttl).Err(); err != nil {
		slog.Warn("Redis cache set failed", "error", err)
	}

	return nil
}

// Delete removes an Instagram post from the cache.
func (c *InstagramCache) Delete(ctx context.Context, url string) error {
	if c.client == nil {
		return nil
	}

	key := c.makeKey(url)
	if err := c.client.Del(ctx, key).Err(); err != nil {
		slog.Warn("Redis cache delete failed", "error", err)
	}

	return nil
}
