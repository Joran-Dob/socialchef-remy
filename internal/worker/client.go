package worker

import (
	"crypto/tls"
	"net/url"
	"strings"

	"github.com/hibiken/asynq"
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

// NewClient creates a new Asynq client for enqueueing tasks
func NewClient(redisURL string) *asynq.Client {
	opt, err := ParseRedisURL(redisURL)
	if err != nil {
		panic("failed to parse Redis URL: " + err.Error())
	}
	return asynq.NewClient(opt)
}

// Close closes the client connection
func Close(client *asynq.Client) error {
	return client.Close()
}
