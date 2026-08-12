package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/devices"
	"github.com/go-rod/rod/lib/proto"
)

func isValidVideoURL(videoURL, contentType string) bool {
	for _, kw := range excludeKeywords {
		if strings.Contains(videoURL, kw) {
			return false
		}
	}
	if strings.Contains(contentType, "video/mp4") {
		return true
	}
	if (strings.Contains(videoURL, "tiktokcdn") || strings.Contains(videoURL, "tiktokv")) &&
		(strings.Contains(videoURL, "mime_type=video_mp4") || strings.Contains(videoURL, ".mp4")) {
		return true
	}
	return false
}

func getVideoInfoFromTikTok(browser *rod.Browser, tiktokURL string) *VideoInfo {
	info := &VideoInfo{
		Title: "tiktok_video",
	}

	page := browser.MustPage()
	defer page.MustClose()

	page.MustEmulate(devices.IPhoneX)

	page.MustEvalOnNewDocument(`
		Object.defineProperty(navigator, 'webdriver', {
			get: () => undefined
		});
	`)

	go page.EachEvent(func(e *proto.NetworkResponseReceived) {
		urlStr := e.Response.URL
		contentType := ""
		if ct, ok := e.Response.Headers["content-type"]; ok {
			contentType = fmt.Sprintf("%v", ct)
		} else if ct, ok := e.Response.Headers["Content-Type"]; ok {
			contentType = fmt.Sprintf("%v", ct)
		}

		if isValidVideoURL(urlStr, contentType) && info.MP4URL == "" {
			info.MP4URL = urlStr
			fmt.Println("--> Đã bắt được link video")
		}
	})()

	err := rod.Try(func() {
		page.Timeout(30 * time.Second).MustNavigate(tiktokURL)
		page.MustWaitLoad()
		time.Sleep(3 * time.Second)
	})

	if err != nil {
		fmt.Printf("x Lỗi khi load trang: %v\n", err)
		return nil
	}

	rodCookies := page.MustCookies()
	for _, c := range rodCookies {
		info.Cookies = append(info.Cookies, &http.Cookie{
			Name:  c.Name,
			Value: c.Value,
		})
	}

	titleText := ""
	el, err := page.Element("article")
	if err == nil {
		titleText, _ = el.Text()
	} else {
		pageInfo, err := page.Info()
		if err == nil {
			titleText = pageInfo.Title
		}
	}
	info.Title = slugify(titleText)
	fmt.Printf("--> Tiêu đề: %s\n", info.Title)

	if info.MP4URL == "" {
		time.Sleep(2 * time.Second)
	}

	return info
}

// processTikTokURL xử lý tải một link TikTok duy nhất
func processTikTokURL(browser *rod.Browser, tiktokURL string, index, total int) ProcessResult {
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Printf("[%d/%d] Đang xử lý: %s\n", index, total, tiktokURL)
	fmt.Println(strings.Repeat("=", 50))

	info := getVideoInfoFromTikTok(browser, tiktokURL)
	if info == nil || info.MP4URL == "" {
		fmt.Println("x Không tìm thấy link video")
		return ResultFailed
	}

	filename := fmt.Sprintf("%s.mp4", info.Title)
	outputFile := getOutputPath(filename)

	if _, err := os.Stat(outputFile); err == nil {
		isValid, fileSize, message := checkVideoFile(outputFile)
		if isValid {
			fmt.Printf("--> File '%s' đã tồn tại và hợp lệ (%.2f MB) -> Bỏ qua!\n", filename, float64(fileSize)/1024/1024)
			return ResultSkipped
		}
		fmt.Printf("⚠️ File '%s' tồn tại nhưng không hợp lệ: %s\n", filename, message)
		fmt.Println("--> Sẽ tải lại file...")
		os.Remove(outputFile)
	}

	success := downloadVideoWithRequests(info.MP4URL, info.Cookies, outputFile, 3)
	if success {
		fmt.Printf("✓ Hoàn tất [%d/%d]: %s\n", index, total, filename)
		return ResultSuccess
	}

	fmt.Printf("x Thất bại [%d/%d]: %s\n", index, total, tiktokURL)
	return ResultFailed
}
