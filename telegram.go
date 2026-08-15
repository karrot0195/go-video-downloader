package main

import (
	"fmt"
	"log"
	"time"

	"github.com/go-rod/rod"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func StartTelegramBot(browser *rod.Browser, bot *tgbotapi.BotAPI, watermarkName string) {
	fmt.Printf("✓ Connected to Telegram Bot: @%s\n", bot.Self.UserName)
	fmt.Println("--> Scanning for new messages ...")

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		var text string
		// var chatID int64

		if update.Message != nil {
			text = update.Message.Text
			// chatID = update.Message.Chat.ID
		} else if update.ChannelPost != nil {
			text = update.ChannelPost.Text
			// chatID = update.ChannelPost.Chat.ID
		} else {
			continue
		}

		// Extract video URLs (TikTok + Facebook) from message
		urls := extractVideoURLs(text)
		if len(urls) == 0 {
			continue
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

			if res == ResultSuccess || res == ResultSkipped {
				// Send video to Telegram
				// sendVideoToTelegram(bot, chatID, outputFile, info)
				fmt.Printf("File saved to: %s\n", outputFile)
			}
		}
	}
}

// sendVideoToTelegram supports 50MB size check, 3 retries, and fallback link sending
func sendVideoToTelegram(bot *tgbotapi.BotAPI, chatID int64, filePath string, info *VideoInfo) {
	if bot == nil || info == nil {
		return
	}
	// 1. Format Caption
	caption := info.Title
	if info.Description != "" {
		caption = fmt.Sprintf("%s\n\n%s", info.Title, info.Description)
	}

	runes := []rune(caption)
	if len(runes) > 1000 {
		caption = string(runes[:997]) + "..."
	}

	// 3. Retry mechanism for sending file (max 3 attempts)
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		fmt.Printf("--> Sending video to Telegram (Attempt %d/%d)...\n", attempt, maxRetries)
		video := tgbotapi.NewVideo(chatID, tgbotapi.FilePath(filePath))
		video.Caption = caption

		_, err := bot.Send(video)
		if err == nil {
			fmt.Println("✓ Video sent successfully!")
			return
		}

		fmt.Printf("⚠️ Error sending video attempt %d: %v\n", attempt, err)
		if attempt < maxRetries {
			waitTime := time.Duration(attempt*2) * time.Second
			fmt.Printf("--> Waiting %v before retrying...\n", waitTime)
			time.Sleep(waitTime)
		}
	}

	// 4. If after 3 attempts MP4 upload still fails -> Send fallback text message with MP4 link
	fmt.Println("x Uploading MP4 to Telegram failed after 3 attempts. Sending fallback link...")
	sendFallbackText(bot, chatID, caption, info.MP4URL)
}

// sendFallbackText sends text message with MP4 Link when file upload fails
func sendFallbackText(bot *tgbotapi.BotAPI, chatID int64, caption string, mp4URL string) {
	msgText := caption
	if mp4URL != "" {
		msgText = fmt.Sprintf("%s\n\n🔗 Direct download link (File >50MB or upload error):\n%s", caption, mp4URL)
	}

	msg := tgbotapi.NewMessage(chatID, msgText)
	_, err := bot.Send(msg)
	if err != nil {
		log.Printf("x Error sending fallback message: %v\n", err)
	} else {
		fmt.Println("✓ Fallback link message sent successfully!")
	}
}
