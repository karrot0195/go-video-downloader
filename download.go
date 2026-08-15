package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/schollz/progressbar/v3"
)

// checkVideoFile validates a downloaded video file
func checkVideoFile(filePath string) (bool, int64, string) {
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false, 0, "File does not exist"
	}

	fileSize := info.Size()
	if fileSize == 0 {
		return false, fileSize, "File is empty"
	}
	if fileSize < MinFileSize {
		return false, fileSize, fmt.Sprintf("File too small (%.0fKB < %.0fKB)", float64(fileSize)/1024, float64(MinFileSize)/1024)
	}

	f, err := os.Open(filePath)
	if err != nil {
		return false, fileSize, fmt.Sprintf("Cannot open file: %v", err)
	}
	defer f.Close()

	header := make([]byte, 1024)
	n, _ := f.Read(header)
	headerData := header[:n]

	// Check for MP4 container signatures
	if bytes.Contains(headerData, []byte("ftyp")) || bytes.Contains(headerData, []byte("moov")) || bytes.Contains(headerData, []byte("mdat")) {
		return true, fileSize, "Valid MP4 file"
	}

	return false, fileSize, "File is not a valid MP4"
}

// downloadVideoWithRequests downloads a video with TikTok-specific headers
func downloadVideoWithRequests(mp4URL string, cookies []*http.Cookie, outputFile string, maxRetries int) bool {
	// Default header for TikTok; other platforms (Facebook) pass their own referer
	return downloadVideo(mp4URL, cookies, outputFile, maxRetries, "https://www.tiktok.com/")
}

// downloadVideo downloads a video from mp4URL to outputFile with platform-specific referer
// (TikTok: https://www.tiktok.com/, Facebook: https://www.facebook.com/)
func downloadVideo(mp4URL string, cookies []*http.Cookie, outputFile string, maxRetries int, referer string) bool {
	client := &http.Client{
		Timeout: DownloadTimeout,
	}

	origin := strings.TrimSuffix(referer, "/")

	for attempt := 0; attempt < maxRetries; attempt++ {
		fmt.Printf("--> Downloading video %s (attempt %d/%d)...\n", mp4URL, attempt+1, maxRetries)

		req, err := http.NewRequest("GET", mp4URL, nil)
		if err != nil {
			fmt.Printf("x Request creation error: %v\n", err)
			continue
		}

		req.Header.Set("User-Agent", DefaultUserAgent)
		req.Header.Set("Referer", referer)
		req.Header.Set("Accept", "*/*")
		req.Header.Set("Accept-Language", "vi-VN,vi;q=0.9,en;q=0.8")
		req.Header.Set("Range", "bytes=0-")
		req.Header.Set("Sec-Fetch-Dest", "video")
		req.Header.Set("Sec-Fetch-Mode", "no-cors")
		req.Header.Set("Sec-Fetch-Site", "same-site")
		req.Header.Set("Origin", origin)

		for _, c := range cookies {
			req.AddCookie(c)
		}

		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("x Connection/timeout error (attempt %d): %v\n", attempt+1, err)
			os.Remove(outputFile)
			time.Sleep(3 * time.Second)
			continue
		}

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
			fmt.Printf("x Status code: %d\n", resp.StatusCode)
			if resp.StatusCode == http.StatusForbidden {
				fmt.Println("⚠️ May need to refresh cookies or use new headers")
			}
			resp.Body.Close()
			if attempt < maxRetries-1 {
				time.Sleep(3 * time.Second)
				continue
			}
			return false
		}

		out, err := os.Create(outputFile)
		if err != nil {
			resp.Body.Close()
			fmt.Printf("x File creation error: %v\n", err)
			return false
		}

		bar := progressbar.DefaultBytes(
			resp.ContentLength,
			"--> Progress",
		)

		_, err = io.Copy(io.MultiWriter(out, bar), resp.Body)
		out.Close()
		resp.Body.Close()

		if err != nil {
			fmt.Printf("x File write error: %v\n", err)
			os.Remove(outputFile)
			if attempt < maxRetries-1 {
				time.Sleep(3 * time.Second)
				continue
			}
			return false
		}

		isValid, fileSize, message := checkVideoFile(outputFile)
		if isValid {
			fmt.Printf("\n✓ Download successful: %s (%.2f MB)\n", outputFile, float64(fileSize)/1024/1024)
			return true
		}

		fmt.Printf("\nx Downloaded file is invalid: %s\n", message)
		os.Remove(outputFile)
		if attempt < maxRetries-1 {
			fmt.Printf("⚠️ Retrying attempt %d...\n", attempt+2)
			time.Sleep(2 * time.Second)
		}
	}

	os.Remove(outputFile)
	return false
}
