package main

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/devices"
	"github.com/go-rod/rod/lib/proto"
)

type fbCandidate struct {
	url         string
	contentType string
	size        int64
}

// fbHeader lấy giá trị header không phân biệt hoa thường từ map header của CDP
func fbHeader(headers proto.NetworkHeaders, key string) string {
	for k, v := range headers {
		if strings.EqualFold(k, key) {
			return v.Str()
		}
	}
	return ""
}

func fbHeaderInt(headers proto.NetworkHeaders, key string) int64 {
	s := fbHeader(headers, key)
	if s == "" {
		return 0
	}
	var n int64
	fmt.Sscanf(s, "%d", &n)
	return n
}

// isValidFacebookVideoURL nhận diện request tải video CDN của Facebook
// trong network traffic của trang (video.xx.fbcdn.net, scontent...fna.fbcdn.net, video.fbsbx.com)
func isValidFacebookVideoURL(videoURL, contentType string) bool {
	if strings.Contains(contentType, "video/") {
		return true
	}
	if strings.Contains(contentType, "application/octet-stream") && strings.Contains(videoURL, "fbcdn.net") {
		return true
	}
	if (strings.Contains(videoURL, "fbcdn.net") || strings.Contains(videoURL, "fbsbx.com")) &&
		strings.Contains(videoURL, ".mp4") {
		return true
	}
	return false
}

// isUsableFacebookVideoURL kiểm tra URL có phải link media trực tiếp dùng được không
func isUsableFacebookVideoURL(videoURL string) bool {
	if videoURL == "" ||
		strings.HasPrefix(videoURL, "blob:") ||
		strings.HasPrefix(videoURL, "data:") ||
		strings.HasPrefix(videoURL, "javascript:") {
		return false
	}
	return strings.Contains(videoURL, "fbcdn.net") ||
		strings.Contains(videoURL, "fbsbx.com") ||
		strings.HasSuffix(videoURL, ".mp4")
}

// unescapeFBURL gỡ escape JSON của Facebook (\/, \u0026, \u00253D, ...) trong URL
func unescapeFBURL(s string) string {
	// JSON string cũng là Go string hợp lệ -> dùng strconv.Unquote để giải mã chuẩn
	if unquoted, err := strconv.Unquote(`"` + s + `"`); err == nil {
		return unquoted
	}
	// Fallback thủ công cho các escape phổ biến
	s = strings.ReplaceAll(s, `\/`, "/")
	s = strings.ReplaceAll(s, `\u0026`, "&")
	s = strings.ReplaceAll(s, `\u003d`, "=")
	s = strings.ReplaceAll(s, `\u0025`, "%")
	s = strings.ReplaceAll(s, `\u003a`, ":")
	s = strings.ReplaceAll(s, `\u003f`, "?")
	s = strings.ReplaceAll(s, `\x3a`, ":")
	s = strings.ReplaceAll(s, `\x2f`, "/")
	if strings.HasPrefix(s, "//") {
		s = "https:" + s
	}
	return s
}

// extractVideoURLsFromHTML quét HTML trang để tìm link video trực tiếp.
// Các key JSON của Facebook (ưu tiên chất lượng HD trước):
//   - browser_native_hd_url / browser_native_sd_url
//   - hd_src / sd_src
//   - playable_url_quality_hd / playable_url
func extractVideoURLsFromHTML(htmlStr string) []string {
	var urls []string
	patterns := []string{
		`"browser_native_hd_url"\s*:\s*"([^"]+)"`,
		`"hd_src"\s*:\s*"([^"]+)"`,
		`"playable_url_quality_hd"\s*:\s*"([^"]+)"`,
		`"browser_native_sd_url"\s*:\s*"([^"]+)"`,
		`"sd_src"\s*:\s*"([^"]+)"`,
		`"playable_url"\s*:\s*"([^"]+)"`,
	}
	for _, p := range patterns {
		re := regexp.MustCompile(p)
		for _, m := range re.FindAllStringSubmatch(htmlStr, -1) {
			if len(m) > 1 {
				u := unescapeFBURL(m[1])
				if isUsableFacebookVideoURL(u) {
					urls = append(urls, u)
				}
			}
		}
	}
	return urls
}

// getVideoInfoFromFacebook mở trang Facebook Reel/Video, bắt link MP4
// từ network traffic; nếu không bắt được thì fallback sang <video> tag,
// og:video meta và JSON nhúng trong HTML.
func getVideoInfoFromFacebook(browser *rod.Browser, fbURL string) *VideoInfo {
	info := &VideoInfo{
		Title: "Facebook Video",
	}

	page := browser.MustPage()
	defer page.MustClose()

	// Dùng desktop (reels hiển thị được trên web desktop)
	page.MustEmulate(devices.LaptopWithMDPIScreen)

	page.MustEvalOnNewDocument(`
		Object.defineProperty(navigator, 'webdriver', {
			get: () => undefined
		});
	`)

	// Bắt các response chứa video CDN của Facebook
	var candidatesMu sync.Mutex
	candidates := make(map[string]fbCandidate)

	go page.EachEvent(func(e *proto.NetworkResponseReceived) {
		urlStr := e.Response.URL
		contentType := fbHeader(e.Response.Headers, "content-type")

		if !isValidFacebookVideoURL(urlStr, contentType) {
			return
		}

		size := fbHeaderInt(e.Response.Headers, "content-length")

		candidatesMu.Lock()
		defer candidatesMu.Unlock()
		if c, ok := candidates[urlStr]; ok {
			if size > c.size {
				candidates[urlStr] = fbCandidate{url: urlStr, contentType: contentType, size: size}
			}
		} else {
			candidates[urlStr] = fbCandidate{url: urlStr, contentType: contentType, size: size}
		}
	})()

	err := rod.Try(func() {
		page.Timeout(45 * time.Second).MustNavigate(fbURL)
		page.MustWaitLoad()
	})

	if err != nil {
		fmt.Printf("x Lỗi khi load trang: %v\n", err)
	} else {
		// Chờ player (thẻ <video>) xuất hiện tối đa 20s để video bắt đầu tải.
		// Dùng polling thay vì MustWait để tránh panic khi trang đang điều hướng.
		deadline := time.Now().Add(20 * time.Second)
		for time.Now().Before(deadline) {
			res, evalErr := page.Eval(`!!(document && document.querySelector('video'))`)
			if evalErr == nil && res.Value.Bool() {
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
		time.Sleep(2 * time.Second)
	}

	// 1. Chọn URL từ network candidates: ưu tiên content-type video, rồi dung lượng lớn nhất
	candidatesMu.Lock()
	var best *fbCandidate
	for _, c := range candidates {
		if best == nil {
			best = &c
			continue
		}
		bVideo := strings.Contains(best.contentType, "video")
		cVideo := strings.Contains(c.contentType, "video")
		if cVideo && !bVideo {
			best = &c
			continue
		}
		if bVideo && !cVideo {
			continue
		}
		if c.size > best.size {
			best = &c
		}
	}
	candidatesMu.Unlock()

	if best != nil {
		info.MP4URL = best.url
		fmt.Printf("--> Đã bắt được link video từ network (%.1f MB)\n", float64(best.size)/1024/1024)
	}

	// 2. Fallback: đọc src trực tiếp từ các thẻ <video> trên trang
	if info.MP4URL == "" {
		if res, err := page.Eval(`() => {
			const srcs = [];
			document.querySelectorAll('video').forEach(v => {
				const s = v.currentSrc || v.src;
				if (s && !s.startsWith('blob:') && !s.startsWith('data:')) srcs.push(s);
			});
			return srcs;
		}`); err == nil {
			for _, item := range res.Value.Arr() {
				s := item.Str()
				if isUsableFacebookVideoURL(s) {
					info.MP4URL = s
					fmt.Println("--> Đã lấy link video từ thẻ <video>")
					break
				}
			}
		}
	}

	// 3. Fallback: quét JSON nhúng trong HTML (browser_native / hd_src / sd_src / playable_url)
	if info.MP4URL == "" {
		if htmlStr, err := page.HTML(); err == nil {
			urls := extractVideoURLsFromHTML(htmlStr)
			if len(urls) > 0 {
				info.MP4URL = urls[0]
				fmt.Println("--> Đã lấy link video từ JSON nhúng trong HTML")
			}
		}
	}

	// 4. Fallback cuối: meta og:video
	if info.MP4URL == "" {
		if res, err := page.Eval(`() => {
			const g = (sel) => {
				const el = document.querySelector(sel);
				return el ? (el.getAttribute('content') || el.content || '').trim() : '';
			};
			return g('meta[property="og:video"]') || g('meta[property="og:video:url"]') || g('meta[property="og:video:secure_url"]');
		}`); err == nil {
			s := res.Value.Str()
			if isUsableFacebookVideoURL(s) {
				info.MP4URL = s
				fmt.Println("--> Đã lấy link video từ meta og:video")
			}
		}
	}

	// Lấy cookies từ trang (cần thiết để tải video CDN)
	rodCookies := page.MustCookies()
	for _, c := range rodCookies {
		info.Cookies = append(info.Cookies, &http.Cookie{
			Name:  c.Name,
			Value: c.Value,
		})
	}

	// Lấy tiêu đề + mô tả từ meta og
	if res, err := page.Eval(`() => {
		const g = (sel) => {
			const el = document.querySelector(sel);
			return el ? (el.getAttribute('content') || el.content || '').trim() : '';
		};
		return {
			title: g('meta[property="og:title"]') || g('meta[name="title"]') || document.title,
			desc: g('meta[property="og:description"]') || g('meta[name="description"]')
		};
	}`); err == nil {
		if t := res.Value.Get("title").Str(); t != "" {
			info.Title = t
		}
		if d := res.Value.Get("desc").Str(); d != "" {
			info.Description = d
		}
	}

	fmt.Printf("--> Tiêu đề: %s\n", info.Title)

	return info
}

// processFacebookURLWithFile xử lý 1 URL Facebook: lấy link video -> tải -> xử lý hậu kỳ
func processFacebookURLWithFile(browser *rod.Browser, fbURL string, index, total int) (string, *VideoInfo, ProcessResult) {
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Printf("[%d/%d] Đang xử lý: %s\n", index, total, fbURL)
	fmt.Println(strings.Repeat("=", 50))

	info := getVideoInfoFromFacebook(browser, fbURL)
	if info == nil || info.MP4URL == "" {
		fmt.Println("x Không tìm thấy link video")
		return "", nil, ResultFailed
	}

	fmt.Printf("VideoInfo: %v\n", info)

	// Tên file lưu trên ổ cứng vẫn slugify để tránh ký tự đặc biệt
	safeFilename := slugify(info.Title)
	if safeFilename == "" || safeFilename == "tiktok_video" {
		safeFilename = "facebook_video"
	}
	filename := fmt.Sprintf("%s.mp4", safeFilename)

	// Nối videoId vào tên file để tránh trùng tên
	if videoID := extractFacebookVideoID(fbURL); videoID != "" {
		filename = fmt.Sprintf("%s_%s.mp4", safeFilename, videoID)
	}
	outputFile := getOutputPath(filename)

	success := downloadVideo(info.MP4URL, info.Cookies, outputFile, 3, "https://www.facebook.com/")
	if success {
		fmt.Printf("--> Đang xử lý video (720p + cắt viền + watermark)...\n")
		if err := postProcessVideo(outputFile); err != nil {
			fmt.Printf("⚠️ Không thể xử lý video: %v\n", err)
			fmt.Println("   (File gốc vẫn được giữ nguyên)")
		} else {
			fmt.Printf("✓ Đã xử lý xong (720p + watermark)\n")
		}
		fmt.Printf("✓ Hoàn tất [%d/%d]: %s\n", index, total, filename)
		return outputFile, info, ResultSuccess
	}

	fmt.Printf("x Thất bại [%d/%d]: %s\n", index, total, fbURL)
	return "", nil, ResultFailed
}
