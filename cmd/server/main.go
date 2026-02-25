package main

import (
	_ "github.com/joho/godotenv/autoload"
	"context"
	"log"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/socialchef/remy/internal/api"
	"github.com/socialchef/remy/internal/config"
	"github.com/socialchef/remy/internal/db"
	"github.com/socialchef/remy/internal/db/generated"
	"github.com/socialchef/remy/internal/middleware"
	"github.com/socialchef/remy/internal/telemetry"
	"github.com/socialchef/remy/internal/worker"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize telemetry
	if cfg.OtelExporterOTLPEndpoint != "" {
		shutdown, err := telemetry.InitTelemetry(ctx, "socialchef-remy", "", "", cfg.OtelExporterOTLPEndpoint, nil)
		if err != nil {
			slog.Warn("Failed to init telemetry", "error", err)
		} else {
			defer shutdown(ctx)
		}
	}

	// Database connection
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	queries := generated.New(pool)

	// Asynq client for enqueuing tasks
	asynqClient := worker.NewClient(cfg.RedisURL)
	defer asynqClient.Close()

	// Initialize broadcaster for progress updates
	_ = worker.NewProgressBroadcaster(cfg.SupabaseURL, cfg.SupabaseServiceRoleKey)

	// API handlers
	apiServer := api.NewServer(cfg, queries, asynqClient)

	// Router
	r := chi.NewRouter()

	// Middleware
	r.Use(telemetry.Middleware())
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	// Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Protected API routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(cfg))
		r.Post("/api/recipe", apiServer.HandleImportRecipe)
		r.Get("/api/recipe-status", apiServer.HandleJobStatus)
		r.Get("/api/user-import-status", apiServer.HandleUserImportStatus)
		r.Post("/api/generate-embedding", apiServer.HandleGenerateEmbedding)
	})

	// Start server
	port := cfg.Port
	if port == "" {
		port = "8080"
	}

	slog.Info("Starting server", "port", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
