package scraper

import (
	"context"
	"net/http"
	"time"

	"github.com/mendableai/firecrawl-go/v2"
	"github.com/socialchef/remy/internal/httpclient"
	"github.com/socialchef/remy/internal/metrics"
	"github.com/socialchef/remy/internal/utils"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
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
	app, err := firecrawl.NewFirecrawlApp(apiKey, "")
	if err != nil {
		// In case of initialization error, we'll create without app
		// The error will be handled when Scrape is called
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

	// Initialize app if not already done
	if s.app == nil {
		app, err := firecrawl.NewFirecrawlApp(s.apiKey, "")
		if err != nil {
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
		res, err := s.app.ScrapeURL(postURL, params)
		if err != nil {
			return nil, err
		}
		return res, nil
	}, config)

	if err != nil {
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
