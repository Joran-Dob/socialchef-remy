package scraper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/socialchef/remy/internal/utils"
)

type TikTokPost struct {
	ID            string
	Caption       string
	VideoURL      string
	ThumbnailURL  string
	OwnerUsername string
	OwnerName     string
	OwnerAvatar   string
	OwnerID       string
}

const (
	apifyActorID   = "GdWCkxBtKWOsKjdch"
	videoKvStoreID = "wHhZCBV1UdGLJZHkV"
)

type TikTokScraper struct {
	apifyKey   string
	httpClient *http.Client
}

func NewTikTokScraper(apifyKey string) *TikTokScraper {
	return &TikTokScraper{
		apifyKey:   apifyKey,
		httpClient: &http.Client{Timeout: 180 * time.Second},
	}
}

func IsTikTokURL(u string) bool {
	matched, _ := regexp.MatchString(`tiktok\.com/`, u)
	return matched
}

func (s *TikTokScraper) Scrape(ctx context.Context, postURL string) (*TikTokPost, error) {
	input := map[string]interface{}{
		"postURLs":                []string{postURL},
		"resultsPerPage":          1,
		"shouldDownloadVideos":    true,
		"shouldDownloadCovers":    true,
		"shouldDownloadSubtitles": false,
		"videoKvStoreIdOrName":    videoKvStoreID,
	}
	inputData, _ := json.Marshal(input)

	config := utils.DefaultRetryConfig()

	body, err := utils.WithRetry(ctx, func(attemptCtx context.Context) ([]byte, error) {
		req, err := http.NewRequestWithContext(attemptCtx, "POST",
			fmt.Sprintf("https://api.apify.com/v2/acts/%s/run-sync", apifyActorID),
			bytes.NewReader(inputData))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+s.apifyKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := s.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests {
			return nil, ErrRateLimited
		}
		if resp.StatusCode == http.StatusNotFound {
			return nil, ErrVideoNotFound
		}
		if resp.StatusCode >= 500 {
			return nil, fmt.Errorf("server error: %d", resp.StatusCode)
		}

		return io.ReadAll(resp.Body)
	}, config)

	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, fmt.Errorf("failed to parse TikTok response: %w", err)
	}

	if len(results) == 0 {
		return nil, ErrVideoNotFound
	}

	item := results[0]
	post := &TikTokPost{
		ID:            getString(item, "id"),
		Caption:       getString(item, "text"),
		VideoURL:      getString(item, "videoUrl"),
		ThumbnailURL:  getStringNested(item, "videoMeta", "coverUrl"),
		OwnerUsername: getStringNested(item, "authorMeta", "name"),
		OwnerName:     getStringNested(item, "authorMeta", "nickName"),
		OwnerAvatar:   getStringNested(item, "authorMeta", "avatar"),
		OwnerID:       getStringNested(item, "authorMeta", "id"),
	}

	// If videoUrl is not directly available, try to get it from the key-value store
	if post.VideoURL == "" && post.ID != "" {
		if videoURL := s.getVideoUrlFromStore(ctx, post.ID); videoURL != "" {
			post.VideoURL = videoURL
		}
	}

	return post, nil
}

func (s *TikTokScraper) getVideoUrlFromStore(ctx context.Context, videoID string) string {
	config := utils.DefaultRetryConfig()

	videoKey, err := utils.WithRetry(ctx, func(attemptCtx context.Context) (string, error) {
		url := fmt.Sprintf("https://api.apify.com/v2/key-value-stores/%s/keys?token=%s", videoKvStoreID, s.apifyKey)
		req, err := http.NewRequestWithContext(attemptCtx, "GET", url, nil)
		if err != nil {
			return "", err
		}

		resp, err := s.httpClient.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		var result struct {
			Data struct {
				Items []struct {
					Key string `json:"key"`
				} `json:"items"`
			} `json:"data"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return "", err
		}

		for _, item := range result.Data.Items {
			if strings.Contains(item.Key, videoID) && strings.HasSuffix(item.Key, ".mp4") {
				return item.Key, nil
			}
		}

		return "", fmt.Errorf("video key not found for ID: %s", videoID)
	}, config)

	if err != nil {
		return ""
	}

	return fmt.Sprintf("https://api.apify.com/v2/key-value-stores/%s/records/%s?token=%s", videoKvStoreID, videoKey, s.apifyKey)
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getStringNested(m map[string]interface{}, key1, key2 string) string {
	if v, ok := m[key1]; ok {
		if nested, ok := v.(map[string]interface{}); ok {
			return getString(nested, key2)
		}
	}
	return ""
}
