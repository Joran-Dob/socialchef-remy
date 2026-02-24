package worker

import (
	"github.com/hibiken/asynq"
)

// NewServer creates a new Asynq server for processing tasks
func NewServer(redisURL string) *asynq.Server {
	opt, err := ParseRedisURL(redisURL)
	if err != nil {
		panic("failed to parse Redis URL: " + err.Error())
	}

	return asynq.NewServer(
		opt,
		asynq.Config{
			Concurrency: 10,
		},
	)
}

// Start starts the server with the given handlers
func Start(srv *asynq.Server, handlers map[string]asynq.HandlerFunc) error {
	mux := asynq.NewServeMux()
	for taskType, handler := range handlers {
		mux.HandleFunc(taskType, handler)
	}
	return srv.Start(mux)
}
