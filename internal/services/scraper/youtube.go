package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/socialchef/remy/internal/errors"
	"github.com/socialchef/remy/internal/httpclient"
	"github.com/socialchef/remy/internal/metrics"
	"github.com/socialchef/remy/internal/utils"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type YouTubePost struct {
	ID            string
	Caption       string
	VideoURL      string
	ThumbnailURL  string
	OwnerUsername string
	OwnerName     string
	OwnerAvatar   string
	OwnerID       string
}

type YouTubeScraper struct {
	apiKey     string
	httpClient *http.Client
}

func NewYouTubeScraper(apiKey string) *YouTubeScraper {
	return &YouTubeScraper{
		apiKey:     apiKey,
		httpClient: httpclient.NewInstrumentedClient(30 * time.Second),
	}
}

func IsYouTubeURL(u string) bool {
	patterns := []string{
		`youtube\.com/watch\?`,
		`youtube\.com/shorts/`,
		`youtu\.be/`,
	}
	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, u); matched {
			return true
		}
	}
	return false
}

func extractVideoID(u string) (string, error) {
	if strings.Contains(u, "youtu.be/") {
		re := regexp.MustCompile(`youtu\.be/([a-zA-Z0-9_-]{11})`)
		matches := re.FindStringSubmatch(u)
		if len(matches) >= 2 {
			return matches[1], nil
		}
	}

	if strings.Contains(u, "youtube.com/shorts/") {
		re := regexp.MustCompile(`youtube\.com/shorts/([a-zA-Z0-9_-]{11})`)
		matches := re.FindStringSubmatch(u)
		if len(matches) >= 2 {
			return matches[1], nil
		}
	}

	parsedURL, err := url.Parse(u)
	if err != nil {
		return "", ErrInvalidURL
	}

	videoID := parsedURL.Query().Get("v")
	if videoID == "" {
		return "", ErrInvalidURL
	}

	if matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]{11}$`, videoID); !matched {
		return "", ErrInvalidURL
	}

	return videoID, nil
}

func (s *YouTubeScraper) Scrape(ctx context.Context, postURL string) (*YouTubePost, error) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime).Seconds()
		attrs := []attribute.KeyValue{attribute.String("provider", "youtube")}
		metrics.ExternalAPIDuration.Record(ctx, duration, metric.WithAttributes(attrs...))
		metrics.ExternalAPICallsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	}()

	videoID, err := extractVideoID(postURL)
	if err != nil {
		return nil, errors.NewScraperError(
			"Invalid YouTube URL",
			"YOUTUBE_INVALID_URL",
			err,
		)
	}

	config := utils.DefaultRetryConfig()

	data, err := utils.WithRetry(ctx, func(attemptCtx context.Context) ([]byte, error) {
		apiURL := fmt.Sprintf(
			"https://www.googleapis.com/youtube/v3/videos?part=snippet&id=%s&key=%s",
			videoID,
			s.apiKey,
		)

		req, err := http.NewRequestWithContext(httpclient.WithProvider(attemptCtx, "YouTube"), "GET", apiURL, nil)
		if err != nil {
			return nil, err
		}

		resp, err := s.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		switch resp.StatusCode {
		case http.StatusTooManyRequests:
			return nil, ErrRateLimited
		case http.StatusNotFound:
			return nil, ErrVideoNotFound
		case http.StatusForbidden:
			return nil, fmt.Errorf("API key invalid or quota exceeded")
		}

		if resp.StatusCode >= 500 {
			return nil, fmt.Errorf("server error: %d", resp.StatusCode)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		return io.ReadAll(resp.Body)
	}, config)

	if err != nil {
		if err == ErrRateLimited {
			return nil, errors.NewRateLimitError(
				"YouTube API rate limit exceeded",
				"YOUTUBE_RATE_LIMIT",
				"Wait a few minutes before trying again",
			)
		}
		if err == ErrVideoNotFound {
			return nil, errors.NewNotFoundError(
				"YouTube video not found",
				"YOUTUBE_VIDEO_NOT_FOUND",
				"Verify the video URL is correct and the video is public",
			)
		}
		return nil, errors.NewScraperError(
			"Failed to fetch YouTube video metadata",
			"YOUTUBE_API_ERROR",
			err,
		)
	}

	var youtubeResp struct {
		Items []struct {
			ID      string `json:"id"`
			Snippet struct {
				Title       string `json:"title"`
				Description string `json:"description"`
				Thumbnails  struct {
					Default struct {
						URL string `json:"url"`
					} `json:"default"`
					Medium struct {
						URL string `json:"url"`
					} `json:"medium"`
					High struct {
						URL string `json:"url"`
					} `json:"high"`
					Standard struct {
						URL string `json:"url"`
					} `json:"standard"`
					MaxRes struct {
						URL string `json:"url"`
					} `json:"maxres"`
				} `json:"thumbnails"`
				ChannelTitle string `json:"channelTitle"`
				ChannelID    string `json:"channelId"`
			} `json:"snippet"`
		} `json:"items"`
	}

	if err := json.Unmarshal(data, &youtubeResp); err != nil {
		return nil, errors.NewScraperError(
			"Failed to parse YouTube API response",
			"YOUTUBE_PARSE_ERROR",
			err,
		)
	}

	if len(youtubeResp.Items) == 0 {
		return nil, errors.NewNotFoundError(
			"YouTube video not found",
			"YOUTUBE_VIDEO_NOT_FOUND",
			"Verify the video URL is correct and the video is public",
		)
	}

	item := youtubeResp.Items[0]
	snippet := item.Snippet

	thumbnailURL := ""
	if snippet.Thumbnails.MaxRes.URL != "" {
		thumbnailURL = snippet.Thumbnails.MaxRes.URL
	} else if snippet.Thumbnails.Standard.URL != "" {
		thumbnailURL = snippet.Thumbnails.Standard.URL
	} else if snippet.Thumbnails.High.URL != "" {
		thumbnailURL = snippet.Thumbnails.High.URL
	} else if snippet.Thumbnails.Medium.URL != "" {
		thumbnailURL = snippet.Thumbnails.Medium.URL
	} else if snippet.Thumbnails.Default.URL != "" {
		thumbnailURL = snippet.Thumbnails.Default.URL
	}

	var captionParts []string
	if snippet.Title != "" {
		captionParts = append(captionParts, snippet.Title)
	}
	if snippet.Description != "" {
		captionParts = append(captionParts, snippet.Description)
	}
	caption := strings.Join(captionParts, "\n\n")

	return &YouTubePost{
		ID:            item.ID,
		Caption:       caption,
		VideoURL:      fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID),
		ThumbnailURL:  thumbnailURL,
		OwnerUsername: snippet.ChannelTitle,
		OwnerName:     snippet.ChannelTitle,
		OwnerAvatar:   "",
		OwnerID:       snippet.ChannelID,
	}, nil
}

func (s *YouTubeScraper) GetPostDescription(ctx context.Context, postURL string) (string, error) {
	post, err := s.Scrape(ctx, postURL)
	if err != nil {
		return "", err
	}
	return post.Caption, nil
}
