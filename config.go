package main

import "time"

const (
	FailedFile       = "failed_urls.txt"
	DownloadTimeout  = 120 * time.Second
	MinFileSize      = 1024 * 100 // 100KB
	OutputDir        = "mp4"
	DefaultUserAgent = "Mozilla/5.0 (iPhone; CPU iPhone OS 15_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.0 Mobile/15E148 Safari/604.1"
)

var excludeKeywords = []string{
	"neutral.ttwstatic.com",
	"tiktok_web_login_static",
	"byteoversea",
	"pstatp.com",
	"avatar",
	"cover",
	"logo",
}
