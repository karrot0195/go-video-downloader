package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// WatermarkInfo stores watermark position and size information in video
type WatermarkInfo struct {
	X, Y, W, H int // Position and size
	Found      bool
}

// postProcessVideo processes video after download:
//  1. Crop 1px from each edge (remove border frame around video)
//  2. Detect and remove old watermark (if present)
//  3. Convert to 720p (longest edge = 1280, maintain aspect ratio, only downscale)
//  4. Add "Chuyen Co Huong" watermark in top-left corner
//
// Result overwrites the input file.
func postProcessVideo(input string, watermarkName string) error {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("ffmpeg not installed: %v", err)
	}

	fontFile := findFontFile()
	if fontFile == "" {
		return fmt.Errorf("no font found for watermark rendering")
	}

	tmp := input + ".tmp.mp4"

	// Step 1: Create sample image from watermark if present
	watermarkImg := "watermark_sample.png"
	hasWatermark := extractWatermarkSample(input, watermarkImg)

	// Step 2: Build filter chain
	filter := buildFilterChain(input, fontFile, hasWatermark, watermarkName, watermarkImg)

	args := []string{
		"-hide_banner", "-loglevel", "error", "-y",
		"-i", input,
		"-vf", filter,
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-crf", "20",
		"-c:a", "copy",
		"-movflags", "+faststart",
		tmp,
	}

	cmd := exec.Command("ffmpeg", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		os.Remove(tmp)
		return fmt.Errorf("ffmpeg error: %v\n%s", err, string(out))
	}

	if err := os.Rename(tmp, input); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("cannot replace original file: %v", err)
	}

	// Clean up temporary files
	os.Remove(watermarkImg)
	return nil
}

// extractWatermarkSample extracts a watermark sample from the video
func extractWatermarkSample(input, output string) bool {
	// Extract first frame
	frameFile := "first_frame.png"
	defer os.Remove(frameFile)

	cmd := exec.Command("ffmpeg",
		"-hide_banner", "-loglevel", "error",
		"-i", input,
		"-ss", "00:00:00",
		"-vframes", "1",
		frameFile,
	)
	if err := cmd.Run(); err != nil {
		return false
	}

	// Assume watermark is in top-left corner within 200x80px area
	// Use ffmpeg to crop and save as sample
	cmd = exec.Command("ffmpeg",
		"-hide_banner", "-loglevel", "error",
		"-i", frameFile,
		"-vf", "crop=300:80:10:10", // Top-left corner area
		"-vframes", "1",
		output,
	)
	if err := cmd.Run(); err != nil {
		return false
	}

	// Check if file exists and is not empty
	info, err := os.Stat(output)
	if err != nil || info.Size() == 0 {
		return false
	}

	return true
}

// detectWatermark finds watermark position in video
func detectWatermark(input, sampleImg string) *WatermarkInfo {
	// Create log file
	logFile := "watermark_log.csv"
	defer os.Remove(logFile)

	// Use find_rect to locate watermark position
	cmd := exec.Command("ffprobe",
		"-hide_banner", "-loglevel", "error",
		"-f", "lavfi",
		fmt.Sprintf("movie=%s,find_rect=object=%s:threshold=0.3", input, sampleImg),
		"-show_entries", "frame=tags:frame_tags=lavfi.rect.x,lavfi.rect.y,lavfi.rect.w,lavfi.rect.h",
		"-of", "csv",
	)

	output, err := cmd.Output()
	if err != nil {
		return &WatermarkInfo{Found: false}
	}

	// Parse CSV output
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) >= 5 {
			// Find tags containing coordinates
			for _, part := range parts {
				if strings.Contains(part, "lavfi.rect.x=") {
					x, _ := strconv.Atoi(strings.TrimPrefix(part, "lavfi.rect.x="))
					y, _ := strconv.Atoi(strings.TrimPrefix(parts[2], "lavfi.rect.y="))
					w, _ := strconv.Atoi(strings.TrimPrefix(parts[3], "lavfi.rect.w="))
					h, _ := strconv.Atoi(strings.TrimPrefix(parts[4], "lavfi.rect.h="))

					return &WatermarkInfo{
						X: x, Y: y, W: w, H: h,
						Found: true,
					}
				}
			}
		}
	}

	return &WatermarkInfo{Found: false}
}

// buildFilterChain builds ffmpeg filter chain
func buildFilterChain(input, fontFile string, hasWatermark bool, watermarkName string, watermarkImg string) string {
	var filters []string

	// 1. Crop 1px from each edge
	filters = append(filters, "crop=iw-2:ih-2")

	// 2. Remove old watermark if present
	if hasWatermark {
		info := detectWatermark(input, watermarkImg)
		if info.Found {
			// Use delogo to remove watermark
			delogoFilter := fmt.Sprintf("delogo=x=%d:y=%d:w=%d:h=%d:show=0",
				info.X, info.Y, info.W, info.H)
			filters = append(filters, delogoFilter)
		}
	}

	// 3. Scale to 720p
	scaleFilter := "scale='if(gt(iw,ih),-2,720)':'if(gt(iw,ih),720,-2)'"
	filters = append(filters, scaleFilter)

	// 4. Insert new watermark (fixed position in top-left corner)
	textFilter := fmt.Sprintf(
		"drawtext=text='%s':fontfile=%s:fontsize=h/50:"+
			"fontcolor=white@0.9:shadowcolor=black@0.6:shadowx=2:shadowy=2:x=12:y=12",
		watermarkName,
		fontFile,
	)
	filters = append(filters, textFilter)

	return strings.Join(filters, ",")
}

// findFontFile finds an available system font for watermark rendering
func findFontFile() string {
	candidates := []string{
		"/usr/share/fonts/dejavu/DejaVuSans-Bold.ttf",
		"/usr/share/fonts/dejavu/DejaVuSans.ttf",
		"/usr/share/fonts/dejavu-sans-fonts/DejaVuSans-Bold.ttf",
		"/usr/share/fonts/dejavu-sans-fonts/DejaVuSans.ttf",
		"/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf",
		"/usr/share/fonts/liberation-sans/LiberationSans-Bold.ttf",
		"/usr/share/fonts/liberation/LiberationSans-Bold.ttf",
		"/usr/share/fonts/truetype/liberation2/LiberationSans-Bold.ttf",
		"/usr/share/fonts/noto/NotoSans-Bold.ttf",
		"/usr/share/fonts/truetype/noto/NotoSans-Bold.ttf",
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	// Fallback: scan common font directories
	dirs := []string{"/usr/share/fonts", "/usr/local/share/fonts"}
	for _, dir := range dirs {
		found, _ := findTTF(dir)
		if found != "" {
			return found
		}
	}
	return ""
}

// findTTF recursively finds the first .ttf/.otf file in a directory
func findTTF(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		path := dir + "/" + e.Name()
		if e.IsDir() {
			if f, _ := findTTF(path); f != "" {
				return f, nil
			}
			continue
		}
		lower := strings.ToLower(e.Name())
		if strings.HasSuffix(lower, ".ttf") || strings.HasSuffix(lower, ".otf") {
			return path, nil
		}
	}
	return "", nil
}
