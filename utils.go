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
		fmt.Printf("--> Đã tạo thư mục: %s\n", OutputDir)
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

	// Loại bỏ dấu tiếng Việt (Unicode NFD)
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

// extractVideoID trích xuất videoId từ URL TikTok, ví dụ:
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

// hostMatches kiểm tra hostname của URL có khớp đúng domain (hoặc subdomain) không
func hostMatches(rawURL, domain string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return strings.Contains(rawURL, domain)
	}
	h := strings.ToLower(parsed.Hostname())
	return h == domain || strings.HasSuffix(h, "."+domain)
}

// isTikTokURL kiểm tra URL có thuộc TikTok không
func isTikTokURL(rawURL string) bool {
	return hostMatches(rawURL, "tiktok.com")
}

// isFacebookURL kiểm tra URL có thuộc Facebook không
// (bao gồm cả link rút gọn fb.watch)
func isFacebookURL(rawURL string) bool {
	return hostMatches(rawURL, "facebook.com") ||
		hostMatches(rawURL, "fb.watch") ||
		hostMatches(rawURL, "fb.com")
}

// isSupportedVideoURL kiểm tra URL có thuộc platform được hỗ trợ không
func isSupportedVideoURL(rawURL string) bool {
	return isTikTokURL(rawURL) || isFacebookURL(rawURL)
}

// extractFacebookVideoID trích xuất ID video từ URL Facebook, ví dụ:
// https://www.facebook.com/share/v/1Q2g4xhfGh/?mibextid=wwXIfr -> 1Q2g4xhfGh
// https://www.facebook.com/reel/123456789/                        -> 123456789
// https://www.facebook.com/user/videos/987654321/                 -> 987654321
func extractFacebookVideoID(fbURL string) string {
	// Bỏ query string và fragment
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

// extractVideoURLs trích xuất tất cả URL video được hỗ trợ
// (TikTok + Facebook) từ một đoạn text (tin nhắn Telegram, file, ...)
func extractVideoURLs(text string) []string {
	// 1. Thay thế tất cả dấu phẩy, xuống dòng, tab thành khoảng trắng
	replacer := strings.NewReplacer(",", " ", "\n", " ", "\r", " ", "\t", " ")
	cleanedText := replacer.Replace(text)

	// 2. Tìm tất cả chuỗi dạng URL
	urlRegex := regexp.MustCompile(`https?://[^\s]+`)
	rawURLs := urlRegex.FindAllString(cleanedText, -1)

	var urls []string
	seen := make(map[string]bool)

	for _, u := range rawURLs {
		// Làm sạch các ký tự rác hoặc dấu câu dính ở 2 đầu URL
		cleanURL := strings.Trim(u, `"',.;()[]{} `)

		// Chỉ lấy URL thuộc platform được hỗ trợ và chưa bị trùng
		if isSupportedVideoURL(cleanURL) && !seen[cleanURL] {
			seen[cleanURL] = true
			urls = append(urls, cleanURL)
		}
	}

	return urls
}
