package worker

import (
	"crypto/tls"
	"net/url"
	"strings"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
)

// ParseRedisURL parses a Redis URL and returns asynq.RedisClientOpt
func ParseRedisURL(redisURL string) (asynq.RedisClientOpt, error) {
	// Handle plain host:port format
	if !strings.HasPrefix(redisURL, "redis://") && !strings.HasPrefix(redisURL, "rediss://") {
		return asynq.RedisClientOpt{Addr: redisURL}, nil
	}

	u, err := url.Parse(redisURL)
	if err != nil {
		return asynq.RedisClientOpt{}, err
	}

	opt := asynq.RedisClientOpt{
		Addr: u.Host,
	}

	if u.User != nil {
		opt.Username = u.User.Username()
		if password, ok := u.User.Password(); ok {
			opt.Password = password
		}
	}

	// For rediss:// (TLS), we need to set TLS config
	if u.Scheme == "rediss" {
		opt.TLSConfig = &tls.Config{InsecureSkipVerify: false}
	}

	return opt, nil
}

// NewClient creates a new Asynq client for enqueueing tasks with OTel instrumentation
func NewClient(redisURL string) *asynq.Client {
	opt, err := ParseRedisURL(redisURL)
	if err != nil {
		panic("failed to parse Redis URL: " + err.Error())
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:      opt.Addr,
		Username:  opt.Username,
		Password:  opt.Password,
		DB:        opt.DB,
		TLSConfig: opt.TLSConfig,
	})

	if err := redisotel.InstrumentTracing(rdb); err != nil {
		panic("failed to instrument Redis client: " + err.Error())
	}

	return asynq.NewClientFromRedisClient(rdb)
}

// Close closes the client connection
func Close(client *asynq.Client) error {
	return client.Close()
}
