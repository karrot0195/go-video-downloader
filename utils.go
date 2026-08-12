package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

func ensureOutputDir() error {
	if _, err := os.Stat(OutputDir); os.IsNotExist(err) {
		err := os.MkdirAll(OutputDir, 0755)
		if err != nil {
			return err
		}
		fmt.Printf("--> Đã tạo thư mục: %s\n", OutputDir)
	}
	return nil
}

func getOutputPath(filename string) string {
	return filepath.Join(OutputDir, filename)
}

func slugify(text string) string {
	text = strings.ReplaceAll(text, "See more", "")
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, "đ", "d")
	text = strings.ReplaceAll(text, "Đ", "D")

	// Loại bỏ dấu tiếng Việt (Unicode NFD)
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, err := transform.String(t, text)
	if err != nil {
		result = text
	}

	regNonAlpha := regexp.MustCompile(`[^a-zA-Z0-9\s]`)
	result = regNonAlpha.ReplaceAllString(result, "")

	regSpaces := regexp.MustCompile(`\s+`)
	result = regSpaces.ReplaceAllString(result, "-")
	result = strings.ToLower(strings.Trim(result, "-"))

	if len(result) > 100 {
		result = result[:100]
	}
	if result == "" {
		return "tiktok_video"
	}
	return result
}

func logFailedURL(url string) {
	f, err := os.OpenFile(FailedFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(url + "\n")
}
