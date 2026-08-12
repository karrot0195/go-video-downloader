package main

import "net/http"

// ProcessResult định nghĩa trạng thái sau khi xử lý một URL
type ProcessResult string

const (
	ResultSuccess ProcessResult = "SUCCESS"
	ResultSkipped ProcessResult = "SKIPPED"
	ResultFailed  ProcessResult = "FAILED"
)

// VideoInfo chứa thông tin video bóc tách được từ trang TikTok
type VideoInfo struct {
	MP4URL  string
	Title   string
	Cookies []*http.Cookie
}
