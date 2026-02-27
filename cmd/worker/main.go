package main

import (
	"context"
	"github.com/hibiken/asynq"
	_ "github.com/joho/godotenv/autoload"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/socialchef/remy/internal/config"
	"github.com/socialchef/remy/internal/db"
	"github.com/socialchef/remy/internal/db/generated"
	"github.com/socialchef/remy/internal/services/groq"
	"github.com/socialchef/remy/internal/services/openai"
	"github.com/socialchef/remy/internal/services/scraper"
	"github.com/socialchef/remy/internal/services/storage"
	"github.com/socialchef/remy/internal/services/transcription"
	"github.com/socialchef/remy/internal/metrics"
	"github.com/socialchef/remy/internal/telemetry"
	"github.com/socialchef/remy/internal/logger"
	"github.com/socialchef/remy/internal/worker"
	"github.com/socialchef/remy/internal/sentry"
)
	func main() {
	defer func() {
		sentry.Recover()
	}()

	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize telemetry
	if cfg.OtelExporterOTLPEndpoint != "" {
		shutdown, err := telemetry.InitTelemetry(ctx, "socialchef-remy-worker", "", "", cfg.OtelExporterOTLPEndpoint, nil)
		if err != nil {
			slog.Warn("Failed to init telemetry", "error", err)
		} else {
			defer shutdown(ctx)
		}
	}

	// Initialize Sentry
	if err := sentry.Init(cfg.SentryDSN, cfg.Env, cfg.ServiceName+"-worker", cfg.ServiceVersion); err != nil {
		slog.Warn("Failed to init Sentry", "error", err)
	} else {
		defer sentry.Flush(2 * time.Second)
		sentry.CaptureMessage("Hello Better Stack, this is a test message from Go worker!")
	}
	// Initialize business metrics
	if err := metrics.Init(); err != nil {
		slog.Warn("Failed to init business metrics", "error", err)
	}

	// Initialize logger with OTel support
	logger := logger.New(cfg.Env)
	slog.SetDefault(logger) // Set as default so slog.Info() uses our handler

	// Database connection
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	queries := generated.New(pool)

	// Initialize services
	openaiClient := openai.NewClient(cfg.OpenAIKey)
	groqClient := groq.NewClient(cfg.GroqKey)
	instagramScraper := scraper.NewInstagramScraper(cfg.ProxyServerURL, cfg.ProxyAPIKey)
	tiktokScraper := scraper.NewTikTokScraper(cfg.ApifyAPIKey)
	provider := transcription.NewProvider(cfg.Transcription, cfg.OpenAIKey, cfg.GroqKey)
	transcriptionClient := transcription.NewProviderAdapter(provider)
	storageClient := storage.NewClient(cfg.SupabaseURL, cfg.SupabaseServiceRoleKey)
	broadcaster := worker.NewProgressBroadcaster(cfg.SupabaseURL, cfg.SupabaseServiceRoleKey)

	workerMetrics, err := worker.NewWorkerMetrics()
	if err != nil {
		slog.Warn("Failed to init worker metrics", "error", err)
	}

	asynqClient := worker.NewClient(cfg.RedisURL)
	defer asynqClient.Close()

	// Recipe processor
	processor := worker.NewRecipeProcessor(
		queries,
		instagramScraper,
		tiktokScraper,
		openaiClient,
		transcriptionClient,
		groqClient,
		storageClient,
		broadcaster,
		workerMetrics,
		asynqClient,
	)


	// Asynq server
	srv := worker.NewServer(cfg.RedisURL)

	// Register handlers
	mux := asynq.NewServeMux()
	mux.Use(worker.SentryMiddleware)
	mux.Use(worker.OTelMiddleware)
	mux.HandleFunc(worker.TypeProcessRecipe, processor.HandleProcessRecipe)
	mux.HandleFunc(worker.TypeGenerateEmbedding, processor.HandleGenerateEmbedding)
	mux.HandleFunc(worker.TypeCleanupJobs, processor.HandleCleanupJobs)

	// Handle shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		slog.Info("Shutting down worker...")
		srv.Shutdown()
	}()

	slog.Info("Starting worker", "redis", cfg.RedisURL)

	if err := srv.Run(mux); err != nil {
		log.Fatalf("Worker failed: %v", err)
	}
}
