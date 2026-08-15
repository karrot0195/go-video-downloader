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
	mode := flag.String("mode", "file", "Mode: 'file' or 'telegram'")
	inputFile := flag.String("file", "input.txt", "URL list file")
	botTokenFlag := flag.String("token", "", "Telegram Bot Token")
	watermarkName := flag.String("watermark", "Chuyen Co Huong", "Watermark name")
	flag.Parse()

	// Initialize directories and load processed history
	ensureOutputDir()

	// (3) Read Token: Priority flag -> TELEGRAM_BOT_TOKEN environment variable
	token := *botTokenFlag
	if token == "" {
		token = os.Getenv("TELEGRAM_BOT_TOKEN")
	}

	var bot *tgbotapi.BotAPI
	var err error
	if token != "" {
		bot, err = tgbotapi.NewBotAPI(token)
		if err != nil {
			fmt.Printf("⚠️ Cannot initialize Telegram Bot: %v\n", err)
			bot = nil
		}
	}

	// Initialize Browser
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
			fmt.Println("x Error: Bot Token required via -token flag or TELEGRAM_BOT_TOKEN environment variable")
			return
		}
		StartTelegramBot(browser, bot, *watermarkName)
		return
	}

	// File mode execution
	runFileMode(browser, bot, *inputFile, *watermarkName)
}

func runFileMode(browser *rod.Browser, bot *tgbotapi.BotAPI, inputFile string, watermarkName string) {
	file, err := os.Open(inputFile)
	if err != nil {
		fmt.Printf("x File not found: %s\n", inputFile)
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

		// Detect platform and call appropriate scraper
		if isFacebookURL(videoURL) {
			outputFile, _, res = processFacebookURLWithFile(browser, videoURL, i+1, len(urls), watermarkName)
		} else {
			outputFile, _, res = processTikTokURLWithFile(browser, videoURL, i+1, len(urls), watermarkName)
		}

		// If download successful and Telegram bot exists -> Send file (if needed)
		if (res == ResultSuccess || res == ResultSkipped) && bot != nil && outputFile != "" {
			// sendVideoToTelegram(bot, targetChatID, outputFile, info)
		}
	}
}
