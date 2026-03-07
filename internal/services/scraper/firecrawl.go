package scraper

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/mendableai/firecrawl-go/v2"
	"github.com/socialchef/remy/internal/httpclient"
	"github.com/socialchef/remy/internal/metrics"
	"github.com/socialchef/remy/internal/utils"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"log/slog"
)

type FirecrawlPost struct {
	ID            string
	Caption       string
	ImageURL      string
	VideoURL      string
	ThumbnailURL  string
	OwnerUsername string
	OwnerName     string
	OwnerAvatar   string
	OwnerID       string
}

type FirecrawlScraper struct {
	apiKey     string
	httpClient *http.Client
	app        *firecrawl.FirecrawlApp
}

func NewFirecrawlScraper(apiKey string) *FirecrawlScraper {
	// Validate API key
	if apiKey == "" {
		slog.Error("Firecrawl scraper initialized with empty API key")
	}

	app, err := firecrawl.NewFirecrawlApp(apiKey, "https://api.firecrawl.dev")
	if err != nil {
		// In case of initialization error, we'll create without app
		// The error will be handled when Scrape is called
		slog.Error("Failed to initialize Firecrawl app", "error", err)
		return &FirecrawlScraper{
			apiKey:     apiKey,
			httpClient: httpclient.NewInstrumentedClient(30 * time.Second),
			app:        nil,
		}
	}

	return &FirecrawlScraper{
		apiKey:     apiKey,
		httpClient: httpclient.NewInstrumentedClient(30 * time.Second),
		app:        app,
	}
}

func IsFirecrawlURL(u string) bool {
	// Firecrawl should not handle Instagram or TikTok URLs
	if IsInstagramURL(u) || IsTikTokURL(u) {
		return false
	}
	return true
}

func (s *FirecrawlScraper) Scrape(ctx context.Context, postURL string) (*FirecrawlPost, error) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime).Seconds()
		attrs := []attribute.KeyValue{attribute.String("provider", "firecrawl")}
		metrics.ExternalAPIDuration.Record(ctx, duration, metric.WithAttributes(attrs...))
		metrics.ExternalAPICallsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	}()

	// Validate API key
	if s.apiKey == "" {
		slog.Error("Firecrawl scrape failed: API key is empty", "url", postURL)
		return nil, fmt.Errorf("firecrawl API key is not configured")
	}

	// Validate URL
	if postURL == "" {
		slog.Error("Firecrawl scrape failed: URL is empty")
		return nil, fmt.Errorf("URL cannot be empty")
	}
	parsedURL, err := url.Parse(postURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		slog.Error("Firecrawl scrape failed: Invalid URL format", "url", postURL, "error", err)
		return nil, fmt.Errorf("invalid URL format: %s", postURL)
	}

	// Initialize app if not already done
	if s.app == nil {
		slog.Info("Re-initializing Firecrawl app", "url", postURL)
		app, err := firecrawl.NewFirecrawlApp(s.apiKey, "https://api.firecrawl.dev")
		if err != nil {
			slog.Error("Failed to re-initialize Firecrawl app", "url", postURL, "error", err)
			return nil, err
		}
		s.app = app
	}

	timeout := int(30) // 30 seconds timeout in seconds
	params := &firecrawl.ScrapeParams{
		Formats: []string{"markdown"},
		Timeout: &timeout,
	}

	config := utils.DefaultRetryConfig()

	result, err := utils.WithRetry(ctx, func(attemptCtx context.Context) (*firecrawl.FirecrawlDocument, error) {
		slog.Debug("Attempting Firecrawl scrape", "url", postURL, "attempt", config.MaxAttempts)
		res, err := s.app.ScrapeURL(postURL, params)
		if err != nil {
			slog.Error("Firecrawl scrape attempt failed",
				"url", postURL,
				"error", err,
				"attempt", config.MaxAttempts)
			return nil, err
		}
		return res, nil
	}, config)

	if err != nil {
		slog.Error("Firecrawl scrape failed after all retries",
			"url", postURL,
			"error", err,
			"api_key_configured", s.apiKey != "")
		return nil, err
	}

	// Extract metadata if available
	ownerUsername := "Unknown"
	ownerName := "Unknown"
	id := ""
	if result.Metadata != nil {
		if result.Metadata.Title != nil && *result.Metadata.Title != "" {
			ownerUsername = *result.Metadata.Title
			ownerName = *result.Metadata.Title
		}
		if result.Metadata.Title != nil && *result.Metadata.Title != "" {
			id = *result.Metadata.Title
			// Use title as a fallback for owner name
			if ownerName == "Unknown" {
				ownerName = *result.Metadata.Title
			}
		}
	}

	post := &FirecrawlPost{
		ID:            id,
		Caption:       result.Markdown,
		ImageURL:      "",
		VideoURL:      "",
		ThumbnailURL:  "",
		OwnerUsername: ownerUsername,
		OwnerName:     ownerName,
		OwnerAvatar:   "",
		OwnerID:       "",
	}

	return post, nil
}
