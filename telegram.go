package main

import (
	"fmt"
	"log"
	"time"

	"github.com/go-rod/rod"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func StartTelegramBot(browser *rod.Browser, bot *tgbotapi.BotAPI) {
	fmt.Printf("✓ Đã kết nối Telegram Bot: @%s\n", bot.Self.UserName)
	fmt.Println("--> Đang quét tin nhắn mới ...")

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

		// Trích xuất các URL video (TikTok + Facebook) từ tin nhắn
		urls := extractVideoURLs(text)
		if len(urls) == 0 {
			continue
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

			if res == ResultSuccess || res == ResultSkipped {
				// Gửi video về Telegram
				// sendVideoToTelegram(bot, chatID, outputFile, info)
				fmt.Printf("File saved into %s\n", outputFile)
			}
		}
	}
}

// sendVideoToTelegram hỗ trợ Kiểm tra dung lượng 50MB, Retry 3 lần và Fallback gửi Link
func sendVideoToTelegram(bot *tgbotapi.BotAPI, chatID int64, filePath string, info *VideoInfo) {
	if bot == nil || info == nil {
		return
	}
	// 1. Định dạng Caption
	caption := info.Title
	if info.Description != "" {
		caption = fmt.Sprintf("%s\n\n%s", info.Title, info.Description)
	}

	runes := []rune(caption)
	if len(runes) > 1000 {
		caption = string(runes[:997]) + "..."
	}

	// 3. Cơ chế Retry gửi file tối đa 3 lần
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		fmt.Printf("--> Đang gửi video về Telegram (Lần %d/%d)...\n", attempt, maxRetries)
		video := tgbotapi.NewVideo(chatID, tgbotapi.FilePath(filePath))
		video.Caption = caption

		_, err := bot.Send(video)
		if err == nil {
			fmt.Println("✓ Đã gửi video thành công!")
			return
		}

		fmt.Printf("⚠️ Lỗi gửi video lần %d: %v\n", attempt, err)
		if attempt < maxRetries {
			waitTime := time.Duration(attempt*2) * time.Second
			fmt.Printf("--> Chờ %v trước khi thử lại...\n", waitTime)
			time.Sleep(waitTime)
		}
	}

	// 4. Nếu sau 3 lần thử gửi file MP4 vẫn lỗi -> Gửi tin nhắn text dự phòng kèm link MP4
	fmt.Println("x Tải file MP4 lên Telegram thất bại sau 3 lần thử. Gửi link dự phòng...")
	sendFallbackText(bot, chatID, caption, info.MP4URL)
}

// sendFallbackText gửi tin nhắn chữ kèm Link MP4 khi không thể upload file
func sendFallbackText(bot *tgbotapi.BotAPI, chatID int64, caption string, mp4URL string) {
	msgText := caption
	if mp4URL != "" {
		msgText = fmt.Sprintf("%s\n\n🔗 Link tải trực tiếp (File >50MB hoặc lỗi upload):\n%s", caption, mp4URL)
	}

	msg := tgbotapi.NewMessage(chatID, msgText)
	_, err := bot.Send(msg)
	if err != nil {
		log.Printf("x Lỗi gửi tin nhắn fallback: %v\n", err)
	} else {
		fmt.Println("✓ Đã gửi tin nhắn link dự phòng thành công!")
	}
}
