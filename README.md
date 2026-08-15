Here's the English translation of the Video Downloader (Go) documentation:

---

# Video Downloader (Go)

An automated TikTok & Facebook Reel video downloader system written in Go, using **go-rod** for web scraping with support for 2 modes: batch downloading from `input.txt` file or automatic link detection from **Telegram Channel / Group / Bot**.

---

## 📌 Key Features

* 🚀 **Download watermark-free videos**: Automatically captures MP4 streams from web interfaces (TikTok + Facebook Reel/Video).
* 📁 **Batch download from list (File Mode)**: Reads a list of links from `input.txt`, skips already downloaded links, and automatically logs failed links.
* 🤖 **Automation via Telegram (Telegram Mode)**:
  * Listens to new messages/posts from Telegram Channels, Groups, or private chats.
  * Automatically downloads videos and replies with processing status directly to Telegram.
* 🔄 **Retry & Skip mechanism**: Automatically retries on download failure, skips already completed MP4 files to save bandwidth.

---

## 🛠 System Requirements

* **Go**: Version `1.18` or higher.
* **Web Browser**: Google Chrome or Chromium (Go-Rod will automatically download a compatible Chromium version if not present).

---

## 📥 Installation

1. **Clone the project or download the source code:**
```bash
git clone https://github.com/your-username/tiktok-downloader.git
cd tiktok-downloader
```

2. **Install dependencies:**
```bash
go mod tidy
```

---

## 🚀 Usage Guide

### Mode 1: Batch download from file (`input.txt`)

1. Create or edit the `input.txt` file in the root directory. Enter the list of TikTok or Facebook URLs (one link per line, use `#` at the beginning of a line to skip):
```text
# List of TikTok links
https://www.tiktok.com/@user/video/1234567890
https://vt.tiktok.com/ZSxxxxxx/

# List of Facebook Reel/Video links
https://www.facebook.com/share/v/xxxxxxxx/
https://www.facebook.com/reel/1234567890
https://fb.watch/xxxxxxx/
```

2. Run the program:
```bash
go run . -mode=file -file=input.txt
```

---

### Mode 2: Automatic download via Telegram Bot

1. **Create a Telegram Bot**:
   * Message [@BotFather](https://t.me/BotFather) on Telegram to create a new Bot and get the **API Token**.
   * Add the Bot to your **Channel / Group** and grant **Administrator** permissions (so the Bot can read posts).

2. **Launch the Bot**:
```bash
go run . -mode=telegram -token="YOUR_TELEGRAM_BOT_TOKEN"
```

3. **Usage**:
   * Post content containing TikTok/Facebook links in the Channel or send links directly to the Bot.
   * The Bot will automatically detect the URL, proceed with downloading, and report results back to Telegram.

---

## 🏗 Compile to Binary Executable

To package the program into a standalone executable without needing Go installed on other machines:

```bash
# Compile
go build -o tiktok-downloader .

# Run File Mode
./tiktok-downloader -mode=file -file=input.txt

# Run Telegram Mode
./tiktok-downloader -mode=telegram -token="YOUR_TELEGRAM_BOT_TOKEN"
```

---

## ⚙️ Command Flags

| Flag | Default | Description |
| --- | --- | --- |
| `-mode` | `file` | Operation mode: `file` or `telegram` |
| `-file` | `input.txt` | Path to the file containing the URL list (Only used when `-mode=file`) |
| `-token` | `""` | Bot Token provided by Telegram (Required when `-mode=telegram`) |

---

## 📂 Project Structure

```text
.
├── main.go         # Main control flow, command-line argument handling
├── telegram.go     # Connection and event handling from Telegram Bot API
├── scraper.go      # Browser interaction (Go-Rod), fetching MP4 URLs and TikTok video info
├── facebook.go     # Fetching MP4 URLs and Facebook Reel/Video info
├── downloader.go   # Download MP4 files, validate file integrity
├── utils.go        # Helper functions (slugify titles, path generation, platform detection...)
├── types.go        # Data type declarations and constants
├── config.go       # Configuration for exclusion keywords, output directory
└── input.txt       # File containing list of TikTok + Facebook links to download
```

---

## ❓ Troubleshooting

* **Error `undefined: ...` when running `go run main.go`:**
  * *Cause:* The project consists of multiple Go files in the same `main` package.
  * *Solution:* Always use `go run .` instead of specifying a single file name.

* **Stuck on Captcha screen / Cannot fetch video link:**
  * *Cause:* TikTok detects automated activity.
  * *Solution:* Switch `Headless(false)` in `main.go` to open the browser interface visually for inspection.

---

Let me know if you'd like any adjustments to the translation!