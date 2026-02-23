package scraper

import "errors"

var (
	ErrPrivateAccount = errors.New("account is private")
	ErrRateLimited    = errors.New("rate limited")
	ErrPostNotFound   = errors.New("post not found")
	ErrVideoNotFound  = errors.New("video not found")
	ErrInvalidURL     = errors.New("invalid URL")
)
