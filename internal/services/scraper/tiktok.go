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
)

type TikTokPost struct {
	ID            string
	Caption       string
	VideoURL      string
	ThumbnailURL  string
	OwnerUsername string
	OwnerName     string
	OwnerAvatar   string
}

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
		"postURLs":               []string{postURL},
		"resultsPerPage":         1,
		"shouldDownloadVideos":   false,
		"shouldDownloadCovers":   true,
		"shouldDownloadSubtitles": false,
	}
	inputData, _ := json.Marshal(input)

	req, err := http.NewRequestWithContext(ctx, "POST",
		"https://api.apify.com/v2/acts/GdWCkxBtKWOsKjdch/run-sync",
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

	if resp.StatusCode == 429 {
		return nil, ErrRateLimited
	}
	if resp.StatusCode == 404 {
		return nil, ErrVideoNotFound
	}

	body, err := io.ReadAll(resp.Body)
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
	return &TikTokPost{
		ID:            getString(item, "id"),
		Caption:       getString(item, "text"),
		VideoURL:      getString(item, "videoUrl"),
		ThumbnailURL:  getStringNested(item, "videoMeta", "coverUrl"),
		OwnerUsername: getStringNested(item, "authorMeta", "name"),
		OwnerName:     getStringNested(item, "authorMeta", "nickName"),
		OwnerAvatar:   getStringNested(item, "authorMeta", "avatar"),
	}, nil
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
