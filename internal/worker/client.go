package worker

import "github.com/hibiken/asynq"

// NewClient creates a new Asynq client for enqueueing tasks
func NewClient(redisURL string) *asynq.Client {
	return asynq.NewClient(asynq.RedisClientOpt{Addr: redisURL})
}

// Close closes the client connection
func Close(client *asynq.Client) error {
	return client.Close()
}
