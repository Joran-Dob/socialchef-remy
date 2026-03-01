package cache

import (
	"context"
	"time"
)

// Cache defines the interface for caching operations.
type Cache interface {
	// Get retrieves a value from the cache by key.
	// Returns nil if the key is not found or has expired.
	Get(ctx context.Context, key string) (interface{}, error)

	// Set stores a value in the cache with the given key and TTL.
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error

	// Delete removes a value from the cache by key.
	Delete(ctx context.Context, key string) error
}
