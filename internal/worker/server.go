package worker

import (
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
)

// NewServer creates a new Asynq server for processing tasks with OTel instrumentation
func NewServer(redisURL string) *asynq.Server {
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

	return asynq.NewServerFromRedisClient(
		rdb,
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
