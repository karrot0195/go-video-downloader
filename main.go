package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	mode := flag.String("mode", "file", "Chế độ: 'file' hoặc 'telegram'")
	inputFile := flag.String("file", "input.txt", "File danh sách URL")
	botTokenFlag := flag.String("token", "", "Telegram Bot Token")
	flag.Parse()

	// Khởi tạo thư mục và nạp lịch sử đã xử lý
	ensureOutputDir()

	// (3) Đọc Token: Ưu tiên flag -> biến môi trường TELEGRAM_BOT_TOKEN
	token := *botTokenFlag
	if token == "" {
		token = os.Getenv("TELEGRAM_BOT_TOKEN")
	}

	var bot *tgbotapi.BotAPI
	var err error
	if token != "" {
		bot, err = tgbotapi.NewBotAPI(token)
		if err != nil {
			fmt.Printf("⚠️ Không thể khởi tạo Telegram Bot: %v\n", err)
			bot = nil
		}
	}

	// Khởi tạo Browser
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
		if bot == nil {
			fmt.Println("x Lỗi: Cần cung cấp Bot Token qua flag -token hoặc biến môi trường TELEGRAM_BOT_TOKEN")
			return
		}
		StartTelegramBot(browser, bot)
		return
	}

	// Chế độ chạy File
	runFileMode(browser, bot, *inputFile)
}

func runFileMode(browser *rod.Browser, bot *tgbotapi.BotAPI, inputFile string) {
	file, err := os.Open(inputFile)
	if err != nil {
		fmt.Printf("x Không tìm thấy file %s\n", inputFile)
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

	for i, videoURL := range urls {
		var outputFile string
		var res ProcessResult

		// Phân loại platform rồi gọi scraper tương ứng
		if isFacebookURL(videoURL) {
			outputFile, _, res = processFacebookURLWithFile(browser, videoURL, i+1, len(urls))
		} else {
			outputFile, _, res = processTikTokURLWithFile(browser, videoURL, i+1, len(urls))
		}

		// Nếu tải thành công và có bot Telegram -> Gửi file (nếu cần)
		if (res == ResultSuccess || res == ResultSkipped) && bot != nil && outputFile != "" {
			// sendVideoToTelegram(bot, targetChatID, outputFile, info)
		}
	}
}
