package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
func postProcessVideo(input string, watermarkName string, hasRemoveOldWatermark bool) error {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("ffmpeg not installed: %v", err)
	}

	fontFile := findFontFile()
	if fontFile == "" {
		return fmt.Errorf("no font found for watermark rendering")
	}

	tmp := input + ".tmp.mp4"

	// step 1: remove watermark with AI cleaner (best-effort)
	cleanedInput := input

	if hasRemoveOldWatermark {
		if cleaned, err := removeWatermarkWithPython(input); err != nil {
			fmt.Printf("⚠️ Watermark removal skipped: %v\n", err)
		} else {
			cleanedInput = cleaned
			defer os.Remove(cleaned)
		}
	}

	// Step 2: Build filter chain
	filter := buildFilterChain(cleanedInput, fontFile, watermarkName)

	args := []string{
		"-hide_banner", "-loglevel", "error", "-y",
		"-i", cleanedInput,
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

	return nil
}

// findWatermarkCleanerScript locates libs/sorawatermarkcleaner/main.py,
// checking relative to the executable first, then the working directory.
func findWatermarkCleanerScript() string {
	rel := filepath.Join("libs", "sorawatermarkcleaner", "main.py")

	if exe, err := os.Executable(); err == nil {
		p := filepath.Join(filepath.Dir(exe), rel)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if _, err := os.Stat(rel); err == nil {
		return rel
	}
	return ""
}

// removeWatermarkWithPython runs libs/sorawatermarkcleaner/main.py to remove
// the source watermark from the video. The script writes its result as
// "{stem}-v2{ext}" next to the input; that cleaned path is returned.
func removeWatermarkWithPython(input string) (string, error) {
	script := findWatermarkCleanerScript()
	if script == "" {
		return "", fmt.Errorf("watermark cleaner script not found (libs/sorawatermarkcleaner/main.py)")
	}

	// Prefer the project's virtualenv (has numpy/torch installed),
	// fall back to python3/python in PATH.
	pythonBin := filepath.Join(filepath.Dir(script), ".venv", "bin", "python")
	if _, err := os.Stat(pythonBin); err != nil {
		pythonBin = "python3"
		if _, err := exec.LookPath(pythonBin); err != nil {
			pythonBin = "python"
			if _, err := exec.LookPath(pythonBin); err != nil {
				return "", fmt.Errorf("no python interpreter found (tried venv, python3, python)")
			}
		}
	}

	ext := filepath.Ext(input)
	cleaned := strings.TrimSuffix(input, ext) + "-v2" + ext

	fmt.Println("--> Removing watermark with AI cleaner (may take a few minutes)...")
	cmd := exec.Command(pythonBin, script, input)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// Only run YOLO detection on every 5th frame; skipped frames are
	// interpolated by the cleaner's interval logic (much faster).
	cmd.Env = append(os.Environ(), "SORA_DETECT_EVERY_N_FRAMES=5")
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("watermark cleaner failed: %v", err)
	}
	if _, err := os.Stat(cleaned); err != nil {
		return "", fmt.Errorf("watermark cleaner finished but output not found: %s", cleaned)
	}
	return cleaned, nil
}

// buildFilterChain builds ffmpeg filter chain
func buildFilterChain(input, fontFile string, watermarkName string) string {
	var filters []string

	// 1. Crop 1px from each edge
	filters = append(filters, "crop=iw-2:ih-2")

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
