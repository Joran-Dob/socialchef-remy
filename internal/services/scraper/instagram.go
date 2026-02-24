package scraper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/socialchef/remy/internal/utils"
)

const (
	lsdToken  = "AVqbxe3J_YA"
	asbdID    = "129477"
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"
)

type InstagramPost struct {
	ID            string
	Caption       string
	ImageURL      string
	VideoURL      string
	OwnerUsername string
	OwnerName     string
	OwnerAvatar   string
}

type InstagramScraper struct {
	proxyURL   string
	proxyKey   string
	httpClient *http.Client
}

func NewInstagramScraper(proxyURL, proxyKey string) *InstagramScraper {
	return &InstagramScraper{
		proxyURL:   proxyURL,
		proxyKey:   proxyKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func IsInstagramURL(u string) bool {
	matched, _ := regexp.MatchString(`instagram\.com/(p|reel|reels)/`, u)
	return matched
}

func extractShortcode(u string) (string, error) {
	re := regexp.MustCompile(`instagram\.com/(?:[A-Za-z0-9_.]+/)?(p|reels?)/([A-Za-z0-9-_]+)`)
	matches := re.FindStringSubmatch(u)
	if len(matches) < 3 {
		return "", ErrInvalidURL
	}
	return matches[2], nil
}

type graphqlResponse struct {
	Data struct {
		ShortcodeMedia struct {
			Shortcode    string `json:"shortcode"`
			DisplayURL   string `json:"display_url"`
			VideoURL     string `json:"video_url"`
			ThumbnailSrc string `json:"thumbnail_src"`
			EdgeMediaToCaption struct {
				Edges []struct {
					Node struct {
						Text string `json:"text"`
					} `json:"node"`
				} `json:"edges"`
			} `json:"edge_media_to_caption"`
			Owner struct {
				ID       string `json:"id"`
				Username string `json:"username"`
			} `json:"owner"`
		} `json:"xdt_shortcode_media"`
	} `json:"data"`
}

func (s *InstagramScraper) Scrape(ctx context.Context, postURL string) (*InstagramPost, error) {
	shortcode, err := extractShortcode(postURL)
	if err != nil {
		return nil, err
	}

	graphQLURL := fmt.Sprintf("https://www.instagram.com/api/graphql?variables={\"shortcode\":\"%s\"}&doc_id=10015901848480474", shortcode)

	config := utils.DefaultRetryConfig()

	data, err := utils.WithRetry(ctx, func(attemptCtx context.Context) ([]byte, error) {
		reqBody := map[string]interface{}{
			"url":    graphQLURL,
			"method": "POST",
			"headers": map[string]string{
				"User-Agent":     userAgent,
				"Content-Type":   "application/x-www-form-urlencoded",
				"X-IG-App-ID":    "936619743392459",
				"X-FB-LSD":       lsdToken,
				"X-ASBD-ID":      asbdID,
				"Sec-Fetch-Site": "same-origin",
			},
		}
		body, _ := json.Marshal(reqBody)

		req, err := http.NewRequestWithContext(attemptCtx, "POST", s.proxyURL, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", s.proxyKey)

		resp, err := s.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests {
			return nil, ErrRateLimited
		}
		if resp.StatusCode >= 500 {
			return nil, fmt.Errorf("server error: %d", resp.StatusCode)
		}

		return io.ReadAll(resp.Body)
	}, config)

	if err != nil {
		return nil, err
	}

	var proxyResp struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(data, &proxyResp); err != nil {
		return nil, err
	}

	var gqlResp graphqlResponse
	if err := json.Unmarshal([]byte(proxyResp.Data), &gqlResp); err != nil {
		return nil, err
	}

	media := gqlResp.Data.ShortcodeMedia
	if media.Shortcode == "" {
		return nil, ErrPostNotFound
	}

	caption := ""
	if len(media.EdgeMediaToCaption.Edges) > 0 {
		caption = media.EdgeMediaToCaption.Edges[0].Node.Text
	}

	return &InstagramPost{
		ID:            media.Shortcode,
		Caption:       caption,
		ImageURL:      media.DisplayURL,
		VideoURL:      media.VideoURL,
		OwnerUsername: media.Owner.Username,
	}, nil
}

func (s *InstagramScraper) GetPostDescription(ctx context.Context, postURL string) (string, error) {
	post, err := s.Scrape(ctx, postURL)
	if err != nil {
		return "", err
	}
	return post.Caption, nil
}
