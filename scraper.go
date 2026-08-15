package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/devices"
	"github.com/go-rod/rod/lib/proto"
)

func isValidVideoURL(videoURL, contentType string) bool {
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
		Title: "TikTok Video",
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

	// Lấy Title và Description từ trang web
	if el, err := page.Element("article"); err == nil {
		text, _ := el.Text()
		lines := strings.Split(strings.TrimSpace(text), "\n")
		if len(lines) > 0 {
			info.Title = strings.TrimSpace(lines[0])
		}
		if len(lines) > 1 {
			info.Description = strings.TrimSpace(strings.Join(lines[1:], "\n"))
		}
	} else if pageInfo, err := page.Info(); err == nil && pageInfo.Title != "" {
		info.Title = strings.TrimSpace(pageInfo.Title)
	}

	fmt.Printf("--> Tiêu đề: %s\n", info.Title)

	if info.MP4URL == "" {
		time.Sleep(2 * time.Second)
	}

	return info
}

// processTikTokURLWithFile trả về đường dẫn file, thông tin video và kết quả
func processTikTokURLWithFile(browser *rod.Browser, tiktokURL string, index, total int) (string, *VideoInfo, ProcessResult) {
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Printf("[%d/%d] Đang xử lý: %s\n", index, total, tiktokURL)
	fmt.Println(strings.Repeat("=", 50))

	info := getVideoInfoFromTikTok(browser, tiktokURL)
	if info == nil || info.MP4URL == "" {
		fmt.Println("x Không tìm thấy link video")
		return "", nil, ResultFailed
	}

	// show info video in a line
	fmt.Printf("VideoInfo: %v\n", info)

	// Tên file lưu trên ổ cứng vẫn slugify để tránh ký tự đặc biệt
	safeFilename := slugify(info.Title)
	if safeFilename == "" {
		safeFilename = "tiktok_video"
	}
	filename := fmt.Sprintf("%s.mp4", safeFilename)

	// Nối videoId vào tên file để tránh trùng tên
	if videoID := extractVideoID(tiktokURL); videoID != "" {
		filename = fmt.Sprintf("%s_%s.mp4", safeFilename, videoID)
	}
	outputFile := getOutputPath(filename)

	// if _, err := os.Stat(outputFile); err == nil {
	// 	isValid, fileSize, _ := checkVideoFile(outputFile)
	// 	if isValid {
	// 		fmt.Printf("--> File '%s' đã tồn tại và hợp lệ (%.2f MB)\n", filename, float64(fileSize)/1024/1024)
	// 		return outputFile, info, ResultSkipped
	// 	}
	// 	os.Remove(outputFile)
	// }

	success := downloadVideoWithRequests(info.MP4URL, info.Cookies, outputFile, 3)
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

	fmt.Printf("x Thất bại [%d/%d]: %s\n", index, total, tiktokURL)
	return "", nil, ResultFailed
}
