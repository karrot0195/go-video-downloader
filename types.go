package main

import "net/http"

type ProcessResult int

const (
	ResultSuccess ProcessResult = iota
	ResultSkipped
	ResultFailed
)

type VideoInfo struct {
	Title       string // Original title
	Description string // Detailed description (if available)
	MP4URL      string // MP4 video download link
	Cookies     []*http.Cookie
}
