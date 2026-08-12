package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/schollz/progressbar/v3"
)

func checkVideoFile(filePath string) (bool, int64, string) {
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false, 0, "File không tồn tại"
	}

	fileSize := info.Size()
	if fileSize == 0 {
		return false, fileSize, "File rỗng"
	}
	if fileSize < MinFileSize {
		return false, fileSize, fmt.Sprintf("File quá nhỏ (%.0fKB < %.0fKB)", float64(fileSize)/1024, float64(MinFileSize)/1024)
	}

	f, err := os.Open(filePath)
	if err != nil {
		return false, fileSize, fmt.Sprintf("Không thể mở file: %v", err)
	}
	defer f.Close()

	header := make([]byte, 1024)
	n, _ := f.Read(header)
	headerData := header[:n]

	if bytes.Contains(headerData, []byte("ftyp")) || bytes.Contains(headerData, []byte("moov")) || bytes.Contains(headerData, []byte("mdat")) {
		return true, fileSize, "File MP4 hợp lệ"
	}

	return false, fileSize, "File không phải MP4 hợp lệ"
}

func downloadVideoWithRequests(mp4URL string, cookies []*http.Cookie, outputFile string, maxRetries int) bool {
	client := &http.Client{
		Timeout: DownloadTimeout,
	}

	for attempt := 0; attempt < maxRetries; attempt++ {
		fmt.Printf("--> Đang tải video (lần %d/%d)...\n", attempt+1, maxRetries)

		req, err := http.NewRequest("GET", mp4URL, nil)
		if err != nil {
			fmt.Printf("x Lỗi tạo request: %v\n", err)
			continue
		}

		req.Header.Set("User-Agent", DefaultUserAgent)
		req.Header.Set("Referer", "https://www.tiktok.com/")
		req.Header.Set("Accept", "*/*")
		req.Header.Set("Accept-Language", "vi-VN,vi;q=0.9,en;q=0.8")
		req.Header.Set("Range", "bytes=0-")
		req.Header.Set("Sec-Fetch-Dest", "video")
		req.Header.Set("Sec-Fetch-Mode", "no-cors")
		req.Header.Set("Sec-Fetch-Site", "same-site")
		req.Header.Set("Origin", "https://www.tiktok.com")

		for _, c := range cookies {
			req.AddCookie(c)
		}

		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("x Lỗi kết nối/timeout (lần %d): %v\n", attempt+1, err)
			os.Remove(outputFile)
			time.Sleep(3 * time.Second)
			continue
		}

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
			fmt.Printf("x Status code: %d\n", resp.StatusCode)
			if resp.StatusCode == http.StatusForbidden {
				fmt.Println("⚠️ Có thể cần refresh cookies hoặc sử dụng headers mới")
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
			fmt.Printf("x Lỗi tạo file: %v\n", err)
			return false
		}

		bar := progressbar.DefaultBytes(
			resp.ContentLength,
			"--> Tiến trình",
		)

		_, err = io.Copy(io.MultiWriter(out, bar), resp.Body)
		out.Close()
		resp.Body.Close()

		if err != nil {
			fmt.Printf("x Lỗi ghi file: %v\n", err)
			os.Remove(outputFile)
			if attempt < maxRetries-1 {
				time.Sleep(3 * time.Second)
				continue
			}
			return false
		}

		isValid, fileSize, message := checkVideoFile(outputFile)
		if isValid {
			fmt.Printf("\n✓ Tải thành công: %s (%.2f MB)\n", outputFile, float64(fileSize)/1024/1024)
			return true
		}

		fmt.Printf("\nx File tải về không hợp lệ: %s\n", message)
		os.Remove(outputFile)
		if attempt < maxRetries-1 {
			fmt.Printf("⚠️ Thử lại lần %d...\n", attempt+2)
			time.Sleep(2 * time.Second)
		}
	}

	os.Remove(outputFile)
	return false
}
