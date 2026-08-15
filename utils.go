package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

func ensureOutputDir() error {
	if _, err := os.Stat(OutputDir); os.IsNotExist(err) {
		err := os.MkdirAll(OutputDir, 0755)
		if err != nil {
			return err
		}
		fmt.Printf("--> Created directory: %s\n", OutputDir)
	}
	return nil
}

func getOutputPath(filename string) string {
	return filepath.Join(OutputDir, filename)
}

func slugify(text string) string {
	text = strings.ReplaceAll(text, "See more", "")
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, "đ", "d")
	text = strings.ReplaceAll(text, "Đ", "D")

	// Remove Vietnamese diacritics (Unicode NFD)
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, err := transform.String(t, text)
	if err != nil {
		result = text
	}

	regNonAlpha := regexp.MustCompile(`[^a-zA-Z0-9\s]`)
	result = regNonAlpha.ReplaceAllString(result, "")

	regSpaces := regexp.MustCompile(`\s+`)
	result = regSpaces.ReplaceAllString(result, "-")
	result = strings.ToLower(strings.Trim(result, "-"))

	if len(result) > 100 {
		result = result[:100]
	}
	if result == "" {
		return "tiktok_video"
	}
	return result
}

// extractVideoID extracts videoId from TikTok URL, e.g.:
// https://www.tiktok.com/@hacaoaudio/video/7672310445612535058?_r=1
// -> 7672310445612535058
func extractVideoID(tiktokURL string) string {
	re := regexp.MustCompile(`/video/(\d+)`)
	m := re.FindStringSubmatch(tiktokURL)
	if len(m) > 1 {
		return m[1]
	}
	return ""
}

// hostMatches checks if URL hostname matches the domain (or subdomain)
func hostMatches(rawURL, domain string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return strings.Contains(rawURL, domain)
	}
	h := strings.ToLower(parsed.Hostname())
	return h == domain || strings.HasSuffix(h, "."+domain)
}

// isTikTokURL checks if URL belongs to TikTok
func isTikTokURL(rawURL string) bool {
	return hostMatches(rawURL, "tiktok.com")
}

// isFacebookURL checks if URL belongs to Facebook
// (including shortened fb.watch links)
func isFacebookURL(rawURL string) bool {
	return hostMatches(rawURL, "facebook.com") ||
		hostMatches(rawURL, "fb.watch") ||
		hostMatches(rawURL, "fb.com")
}

// isSupportedVideoURL checks if URL belongs to a supported platform
func isSupportedVideoURL(rawURL string) bool {
	return isTikTokURL(rawURL) || isFacebookURL(rawURL)
}

// extractFacebookVideoID extracts video ID from Facebook URL, e.g.:
// https://www.facebook.com/share/v/1Q2g4xhfGh/?mibextid=wwXIfr -> 1Q2g4xhfGh
// https://www.facebook.com/reel/123456789/                        -> 123456789
// https://www.facebook.com/user/videos/987654321/                 -> 987654321
func extractFacebookVideoID(fbURL string) string {
	// Remove query string and fragment
	clean := strings.SplitN(fbURL, "?", 2)[0]
	clean = strings.SplitN(clean, "#", 2)[0]
	clean = strings.TrimRight(clean, "/")

	parts := strings.Split(clean, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] != "" {
			return parts[i]
		}
	}
	return ""
}

func logFailedURL(url string) {
	f, err := os.OpenFile(FailedFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(url + "\n")
}

// extractVideoURLs extracts all supported video URLs
// (TikTok + Facebook) from a text (Telegram message, file, etc.)
func extractVideoURLs(text string) []string {
	// 1. Replace all commas, newlines, tabs with spaces
	replacer := strings.NewReplacer(",", " ", "\n", " ", "\r", " ", "\t", " ")
	cleanedText := replacer.Replace(text)

	// 2. Find all URL-like strings
	urlRegex := regexp.MustCompile(`https?://[^\s]+`)
	rawURLs := urlRegex.FindAllString(cleanedText, -1)

	var urls []string
	seen := make(map[string]bool)

	for _, u := range rawURLs {
		// Clean trailing garbage or punctuation from both ends of URL
		cleanURL := strings.Trim(u, `"',.;()[]{} `)

		// Only keep URLs from supported platforms and avoid duplicates
		if isSupportedVideoURL(cleanURL) && !seen[cleanURL] {
			seen[cleanURL] = true
			urls = append(urls, cleanURL)
		}
	}

	return urls
}
