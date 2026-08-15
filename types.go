package main

import "net/http"

type ProcessResult int

const (
	ResultSuccess ProcessResult = iota
	ResultSkipped
	ResultFailed
)

type VideoInfo struct {
	Title       string // Tiêu đề gốc
	Description string // Mô tả chi tiết (nếu có)
	MP4URL      string // Link tải video MP4
	Cookies     []*http.Cookie
}
