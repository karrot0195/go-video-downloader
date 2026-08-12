package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

func main() {
	mode := flag.String("mode", "file", "Chế độ chạy: 'file' hoặc 'telegram'")
	inputFile := flag.String("file", "input.txt", "Đường dẫn file input (khi ở mode file)")
	botToken := flag.String("token", "", "Telegram Bot Token (khi ở mode telegram)")
	flag.Parse()

	ensureOutputDir()

	// Khởi tạo Browser dùng chung
	path, _ := launcher.LookPath()
	u := launcher.New().
		Bin(path).
		Headless(true).
		Set("disable-blink-features", "AutomationControlled").
		Set("no-sandbox").
		MustLaunch()

	browser := rod.New().ControlURL(u).MustConnect()
	defer browser.MustClose()

	if *mode == "telegram" {
		if *botToken == "" {
			fmt.Println("x Vui lòng cung cấp Telegram Bot Token qua tham số -token=YOUR_BOT_TOKEN")
			return
		}
		StartTelegramBot(browser, *botToken)
		return
	}

	// Chế độ File input mặc định
	runFileMode(browser, *inputFile)
}

func runFileMode(browser *rod.Browser, inputFile string) {
	file, err := os.Open(inputFile)
	if err != nil {
		fmt.Printf("x Không tìm thấy file %s.\n", inputFile)
		return
	}
	defer file.Close()

	var urls []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			urls = append(urls, line)
		}
	}

	totalURLs := len(urls)
	if totalURLs == 0 {
		fmt.Printf("x File %s trống.\n", inputFile)
		return
	}

	os.Remove(FailedFile)

	fmt.Printf("--> Tìm thấy %d link trong file %s\n", totalURLs, inputFile)
	fmt.Printf("--> Thư mục lưu video: %s/\n", OutputDir)
	fmt.Println("--> Bắt đầu xử lý...\n")

	var successCount, skippedCount, failedCount int

	for i, tiktokURL := range urls {
		res := processTikTokURL(browser, tiktokURL, i+1, totalURLs)

		switch res {
		case ResultSkipped:
			skippedCount++
		case ResultSuccess:
			successCount++
		default:
			fmt.Printf("⚠️ Thử lại lần 2 cho URL: %s\n", tiktokURL)
			time.Sleep(3 * time.Second)
			resRetry := processTikTokURL(browser, tiktokURL, i+1, totalURLs)

			switch resRetry {
			case ResultSkipped:
				skippedCount++
			case ResultSuccess:
				successCount++
			default:
				failedCount++
				fmt.Printf("x Ghi nhận URL thất bại vào file %s\n", FailedFile)
				logFailedURL(tiktokURL)
			}
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("✓ HOÀN THÀNH!")
	fmt.Printf("📁 Thư mục lưu video: %s/\n", OutputDir)
	fmt.Printf("✅ Thành công: %d\n", successCount)
	fmt.Printf("⏭️ Bỏ qua (đã tải): %d\n", skippedCount)
	fmt.Printf("❌ Thất bại: %d\n", failedCount)
	if failedCount > 0 {
		fmt.Printf("📝 Danh sách URL thất bại đã được ghi vào: %s\n", FailedFile)
	}
	fmt.Println(strings.Repeat("=", 50))
}
