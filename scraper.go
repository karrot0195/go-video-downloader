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
			fmt.Println("--> Captured video link: " + urlStr)
		}
	})()

	err := rod.Try(func() {
		page.Timeout(30 * time.Second).MustNavigate(tiktokURL)
		page.MustWaitLoad()
		time.Sleep(3 * time.Second)
	})

	if err != nil {
		fmt.Printf("x Error loading page: %v\n", err)
		return nil
	}

	rodCookies := page.MustCookies()
	for _, c := range rodCookies {
		info.Cookies = append(info.Cookies, &http.Cookie{
			Name:  c.Name,
			Value: c.Value,
		})
	}

	// Get Title and Description from the webpage
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

	fmt.Printf("--> Title: %s\n", info.Title)

	if info.MP4URL == "" {
		time.Sleep(2 * time.Second)
	}

	return info
}

// processTikTokURLWithFile returns file path, video info, and result status
func processTikTokURLWithFile(browser *rod.Browser, tiktokURL string, index, total int, watermarkName string, hasRemoveOldWatermark bool) (string, *VideoInfo, ProcessResult) {
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Printf("[%d/%d] Processing: %s\n", index, total, tiktokURL)
	fmt.Println(strings.Repeat("=", 50))

	info := getVideoInfoFromTikTok(browser, tiktokURL)
	if info == nil || info.MP4URL == "" {
		fmt.Println("x No video link found")
		return "", nil, ResultFailed
	}

	// Show video info in one line
	fmt.Printf("VideoInfo: %v\n", info)

	// Filename saved to disk is still slugified to avoid special characters
	safeFilename := slugify(info.Title)
	if safeFilename == "" {
		safeFilename = "tiktok_video"
	}
	filename := fmt.Sprintf("%s.mp4", safeFilename)

	// Append videoId to filename to avoid duplicates
	if videoID := extractVideoID(tiktokURL); videoID != "" {
		filename = fmt.Sprintf("%s_%s.mp4", safeFilename, videoID)
	}
	outputFile := getOutputPath(filename)

	// if _, err := os.Stat(outputFile); err == nil {
	// 	isValid, fileSize, _ := checkVideoFile(outputFile)
	// 	if isValid {
	// 		fmt.Printf("--> File '%s' already exists and is valid (%.2f MB)\n", filename, float64(fileSize)/1024/1024)
	// 		return outputFile, info, ResultSkipped
	// 	}
	// 	os.Remove(outputFile)
	// }

	success := downloadVideoWithRequests(info.MP4URL, info.Cookies, outputFile, 3)
	if success {
		fmt.Printf("--> Processing video (720p + crop + watermark)...\n")
		if err := postProcessVideo(outputFile, watermarkName, hasRemoveOldWatermark); err != nil {
			fmt.Printf("⚠️ Cannot process video: %v\n", err)
			fmt.Println("   (Original file kept unchanged)")
		} else {
			fmt.Printf("✓ Processing complete (720p + watermark)\n")
		}
		fmt.Printf("✓ Completed [%d/%d]: %s\n", index, total, filename)
		return outputFile, info, ResultSuccess
	}

	fmt.Printf("x Failed [%d/%d]: %s\n", index, total, tiktokURL)
	return "", nil, ResultFailed
}
