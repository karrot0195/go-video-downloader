package main

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/go-rod/rod"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// StartTelegramBot lắng nghe tin nhắn từ Telegram Channel hoặc tin nhắn riêng
func StartTelegramBot(browser *rod.Browser, botToken string) {
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatalf("x Lỗi khởi tạo Telegram Bot: %v", err)
	}

	fmt.Printf("✓ Đã kết nối Telegram Bot thành công: @%s\n", bot.Self.UserName)
	fmt.Println("--> Đang chờ bài viết mới từ Channel / Message...")

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)
	urlRegex := regexp.MustCompile(`https?://[^\s]+`)

	for update := range updates {
		var text string
		var chatID int64

		// Lấy tin nhắn riêng/nhóm HOẶC bài viết từ Channel
		if update.Message != nil {
			text = update.Message.Text
			chatID = update.Message.Chat.ID
		} else if update.ChannelPost != nil {
			text = update.ChannelPost.Text
			chatID = update.ChannelPost.Chat.ID
		} else {
			continue
		}

		// Trích xuất các URL trong tin nhắn
		urls := urlRegex.FindAllString(text, -1)
		if len(urls) == 0 {
			continue
		}

		for _, tiktokURL := range urls {
			// Chỉ xử lý link TikTok
			if !strings.Contains(tiktokURL, "tiktok.com") {
				continue
			}

			// Gửi thông báo bắt đầu tải
			statusMsg := tgbotapi.NewMessage(chatID, fmt.Sprintf("⏳ Đang xử lý: %s", tiktokURL))
			bot.Send(statusMsg)

			// Tải video
			res := processTikTokURL(browser, tiktokURL, 1, 1)

			// Phản hồi kết quả về lại Telegram
			var replyText string
			switch res {
			case ResultSuccess:
				replyText = fmt.Sprintf("✅ Tải thành công video từ: %s", tiktokURL)
			case ResultSkipped:
				replyText = fmt.Sprintf("⏭️ Video đã tồn tại: %s", tiktokURL)
			default:
				replyText = fmt.Sprintf("❌ Tải thất bại: %s", tiktokURL)
			}

			reply := tgbotapi.NewMessage(chatID, replyText)
			bot.Send(reply)
		}
	}
}
