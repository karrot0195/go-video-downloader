# Video Downloader (Go)

Hệ thống tải video TikTok & Facebook Reel tự động viết bằng Go, sử dụng **go-rod** để cào dữ liệu và hỗ trợ 2 chế độ: tải hàng loạt từ file `input.txt` hoặc tự động bắt link từ **Telegram Channel / Group / Bot**.

---

## 📌 Tính năng chính

* 🚀 **Tải video không watermark**: Tự động bắt luồng MP4 từ giao diện web (TikTok + Facebook Reel/Video).
* 📁 **Tải theo danh sách (File Mode)**: Đọc danh sách link từ file `input.txt`, hỗ trợ bỏ qua link đã tải và tự động ghi log các link lỗi.
* 🤖 **Tự động hóa qua Telegram (Telegram Mode)**:
* Lắng nghe tin nhắn/bài viết mới từ Telegram Channel, Group hoặc chat riêng.
* Tự động tải video và phản hồi trạng thái xử lý trực tiếp về Telegram.


* 🔄 **Cơ chế Retry & Skip**: Tự động thử lại khi tải thất bại, bỏ qua các file MP4 đã tải hoàn chỉnh để tiết kiệm băng thông.

---

## 🛠 Yêu cầu hệ thống

* **Go**: Phiên bản `1.18` trở lên.
* **Trình duyệt Web**: Google Chrome hoặc Chromium (Go-Rod sẽ tự động tải phiên bản Chromium phù hợp nếu chưa có).

---

## 📥 Cài đặt

1. **Clone dự án hoặc tải mã nguồn về máy:**
```bash
git clone https://github.com/your-username/tiktok-downloader.git
cd tiktok-downloader

```


2. **Cài đặt các thư viện phụ thuộc (Dependencies):**
```bash
go mod tidy

```



---

## 🚀 Hướng dẫn sử dụng

### Chế độ 1: Tải hàng loạt từ file (`input.txt`)

1. Tạo hoặc chỉnh sửa file `input.txt` trong thư mục gốc. Nhập danh sách URL TikTok hoặc Facebook (mỗi link 1 dòng, dùng `#` ở đầu dòng để bỏ qua):
```text
# Danh sách link TikTok
https://www.tiktok.com/@user/video/1234567890
https://vt.tiktok.com/ZSxxxxxx/

# Danh sách link Facebook Reel/Video
https://www.facebook.com/share/v/xxxxxxxx/
https://www.facebook.com/reel/1234567890
https://fb.watch/xxxxxxx/

```


2. Chạy chương trình:
```bash
go run . -mode=file -file=input.txt

```



---

### Chế độ 2: Tự động tải qua Telegram Bot

1. **Tạo Telegram Bot**:
* Nhắn tin cho [@BotFather](https://t.me/BotFather) trên Telegram để tạo Bot mới và lấy **API Token**.
* Thêm Bot vào **Channel / Group** của bạn và cấp quyền **Administrator** (để Bot đọc được bài viết).


2. **Khởi chạy Bot**:
```bash
go run . -mode=telegram -token="YOUR_TELEGRAM_BOT_TOKEN"

```


3. **Sử dụng**:
* Đăng bài viết chứa link TikTok vào Channel hoặc gửi trực tiếp link cho Bot.
* Bot sẽ tự động nhận diện URL, tiến hành tải và báo kết quả lại Telegram.



---

## 🏗 Biên dịch thành file thực thi (Binary)

Nếu muốn đóng gói chương trình thành file chạy trực tiếp mà không cần cài đặt Go trên máy khác:

```bash
# Biên dịch
go build -o tiktok-downloader .

# Chạy Chế độ File
./tiktok-downloader -mode=file -file=input.txt

# Chạy Chế độ Telegram
./tiktok-downloader -mode=telegram -token="YOUR_TELEGRAM_BOT_TOKEN"

```

---

## ⚙️ Bảng tham số (Command Flags)

| Tham số | Mặc định | Mô tả |
| --- | --- | --- |
| `-mode` | `file` | Chế độ hoạt động: `file` hoặc `telegram` |
| `-file` | `input.txt` | Đường dẫn tới file chứa danh sách URL (Chỉ dùng khi `-mode=file`) |
| `-token` | `""` | Bot Token do Telegram cấp (Bắt buộc khi `-mode=telegram`) |

---

## 📂 Cấu trúc dự án

```text
.
├── main.go         # Luồng điều khiển chính, xử lý tham số dòng lệnh
├── telegram.go     # Xử lý kết nối và sự kiện từ Telegram Bot API
├── scraper.go      # Tương tác trình duyệt (Go-Rod), lấy URL MP4 và thông tin video TikTok
├── facebook.go     # Lấy URL MP4 và thông tin video Facebook Reel/Video
├── downloader.go   # Tải file MP4, kiểm tra tính hợp lệ của file
├── utils.go        # Hàm bổ trợ (slugify tiêu đề, tạo đường dẫn, nhận diện platform...)
├── types.go        # Khai báo kiểu dữ liệu và hằng số
├── config.go       # Cấu hình danh sách từ khóa loại trừ, thư mục xuất file
└── input.txt       # File chứa danh sách link TikTok + Facebook cần tải

```

---

## ❓ Sửa lỗi thường gặp (Troubleshooting)

* **Lỗi `undefined: ...` khi chạy `go run main.go`:**
* *Nguyên nhân:* Dự án gồm nhiều file Go trong cùng package `main`.
* *Khắc phục:* Luôn dùng lệnh `go run .` thay vì chỉ định tên 1 file.


* **Kẹt ở màn hình Captcha / Không bắt được link video:**
* *Nguyên nhân:* TikTok phát hiện thao tác tự động.
* *Khắc phục:* Chuyển `Headless(false)` trong `main.go` để mở giao diện trình duyệt trực quan và kiểm tra.