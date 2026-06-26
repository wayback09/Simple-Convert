package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
	"golang.org/x/image/webp"
)

// App struct
type App struct {
	ctx          context.Context
	ffmpegPath   string
	ffprobePath  string
	activeCmd    *exec.Cmd
	mu           sync.Mutex
	wasCancelled atomic.Bool
	hwEncodersMu sync.Mutex
	hwEncoders   map[string]bool
}

// NewApp creates a new App struct instance
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	copyFile("logo.png", filepath.Join("frontend", "logo.png"))
	copyFile("logo.png", filepath.Join("build", "appicon.png"))
	copyFile("logo.ico", filepath.Join("build", "windows", "icon.ico"))
}

// copyFile copies a file from src to dst. Silently ignores errors to prevent blocking startup.
func copyFile(src, dst string) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return
	}

	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer out.Close()

	_, _ = io.Copy(out, in)
}

// MediaInfo contains properties parsed by ffprobe
type MediaInfo struct {
	Path          string  `json:"path"`
	Size          int64   `json:"size"`
	Format        string  `json:"format"`
	Duration      float64 `json:"duration"`
	VideoCodec    string  `json:"videoCodec"`
	Resolution    string  `json:"resolution"`
	FrameRate     string  `json:"frameRate"`
	AudioCodec    string  `json:"audioCodec"`
	AudioChannels int     `json:"audioChannels"`
	HasVideo      bool    `json:"hasVideo"`
	HasAudio      bool    `json:"hasAudio"`
}

// FFProbeResult maps to standard ffprobe JSON output structure
type FFProbeResult struct {
	Format struct {
		Filename   string `json:"filename"`
		FormatName string `json:"format_name"`
		Duration   string `json:"duration"`
		Size       string `json:"size"`
	} `json:"format"`
	Streams []struct {
		CodecType  string `json:"codec_type"`
		CodecName  string `json:"codec_name"`
		Width      int    `json:"width"`
		Height     int    `json:"height"`
		RFrameRate string `json:"r_frame_rate"`
		Channels   int    `json:"channels"`
	} `json:"streams"`
}

// ConversionParams holds input configuration from UI
type ConversionParams struct {
	InputPath       string `json:"inputPath"`
	OutputPath      string `json:"outputPath"`
	TargetFormat    string `json:"targetFormat"`
	ResolutionScale string `json:"resolutionScale"`
	QualityPreset   string `json:"qualityPreset"`
	EnableTrim      bool   `json:"enableTrim"`
	TrimStart       string `json:"trimStart"`
	TrimEnd         string `json:"trimEnd"`
	StripAudio      bool   `json:"stripAudio"`
	ExtractAudio    bool   `json:"extractAudio"`
	HwAccel         string `json:"hwAccel"`
	NoReencode      bool   `json:"noReencode"`
}

// ProgressData represents progress metrics sent to the UI
type ProgressData struct {
	Percent   int    `json:"percent"`
	Frame     string `json:"frame"`
	Fps       string `json:"fps"`
	Speed     string `json:"speed"`
	Bitrate   string `json:"bitrate"`
	OutTimeMs int64  `json:"outTimeMs"`
}

// isExecutable checks if a binary is runnable by calling it with the -version flag and testing redirected pipes
func isExecutable(path string) bool {
	pathLower := strings.ToLower(path)
	if strings.Contains(pathLower, "chocolatey\\bin") || strings.Contains(pathLower, "scoop\\shims") {
		return false
	}

	cmd := exec.Command(path, "-version")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	stdout, err1 := cmd.StdoutPipe()
	stderr, err2 := cmd.StderrPipe()
	if err1 != nil || err2 != nil {
		return false
	}

	if err := cmd.Start(); err != nil {
		return false
	}

	// Consume stdout and stderr in background to prevent process block
	go io.Copy(io.Discard, stdout)
	go io.Copy(io.Discard, stderr)

	return cmd.Wait() == nil
}

// CheckFFmpeg checks if FFmpeg and FFprobe are available either in the system PATH, local AppData folder, or embedded in the binary
func (a *App) CheckFFmpeg() (bool, error) {
	pathFfmpeg, err1 := exec.LookPath("ffmpeg")
	pathFfprobe, err2 := exec.LookPath("ffprobe")
	if err1 == nil && err2 == nil {
		if isExecutable(pathFfmpeg) && isExecutable(pathFfprobe) {
			a.ffmpegPath = pathFfmpeg
			a.ffprobePath = pathFfprobe
			return true, nil
		}
	}

	appDataDir, err := os.UserConfigDir()
	if err != nil {
		return false, err
	}

	localBinDir := filepath.Join(appDataDir, "simple-convert", "bin")
	localFFmpeg := filepath.Join(localBinDir, "ffmpeg.exe")
	localFFprobe := filepath.Join(localBinDir, "ffprobe.exe")

	if _, err := os.Stat(localFFmpeg); err == nil {
		if _, err := os.Stat(localFFprobe); err == nil {
			if isExecutable(localFFmpeg) && isExecutable(localFFprobe) {
				a.ffmpegPath = localFFmpeg
				a.ffprobePath = localFFprobe
				return true, nil
			}
		}
	}

	// Try extracting from embedded files if compiled with embedded build constraint
	extracted, err := a.extractEmbeddedBinaries()
	if err == nil && extracted {
		if isExecutable(a.ffmpegPath) && isExecutable(a.ffprobePath) {
			return true, nil
		}
	}

	return false, nil
}

// SetupFFmpeg downloads and extracts FFmpeg Windows binaries from Gyan.dev
func (a *App) SetupFFmpeg() error {
	appDataDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	localBinDir := filepath.Join(appDataDir, "simple-convert", "bin")
	err = os.MkdirAll(localBinDir, 0755)
	if err != nil {
		return err
	}

	url := "https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-essentials.zip"
	tempZip := filepath.Join(localBinDir, "ffmpeg_temp.zip")

	// Request with standard User-Agent header to avoid 403 blocks
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request to Gyan.dev: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("gyan.dev returned HTTP status: %s", resp.Status)
	}

	totalSize := resp.ContentLength
	out, err := os.Create(tempZip)
	if err != nil {
		return fmt.Errorf("failed to create temporary zip: %v", err)
	}
	defer out.Close()
	defer os.Remove(tempZip)

	buffer := make([]byte, 65536)
	var downloaded int64
	startTime := time.Now()

	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			_, writeErr := out.Write(buffer[:n])
			if writeErr != nil {
				return fmt.Errorf("failed writing to temp file: %v", writeErr)
			}
			downloaded += int64(n)

			pct := 0.0
			if totalSize > 0 {
				pct = float64(downloaded) / float64(totalSize) * 100.0
			}

			elapsed := time.Since(startTime).Seconds()
			speed := 0.0
			if elapsed > 0 {
				speed = float64(downloaded) / (1024 * 1024 * elapsed)
			}

			downloadedMB := float64(downloaded) / (1024 * 1024)
			totalMB := float64(totalSize) / (1024 * 1024)

			runtime.EventsEmit(a.ctx, "setup:progress", map[string]interface{}{
				"percent":    int(pct),
				"speed":      fmt.Sprintf("%.2f MB/s", speed),
				"downloaded": fmt.Sprintf("%.2f MB", downloadedMB),
				"total":      fmt.Sprintf("%.2f MB", totalMB),
				"status":     "Downloading FFmpeg...",
			})
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading response stream: %v", err)
		}
	}
	out.Close() // close file handle before unzip

	runtime.EventsEmit(a.ctx, "setup:progress", map[string]interface{}{
		"percent": 100,
		"status":  "Extracting FFmpeg binaries...",
	})

	r, err := zip.OpenReader(tempZip)
	if err != nil {
		return fmt.Errorf("failed to open downloaded zip archive: %v", err)
	}
	defer r.Close()

	var extractedFfmpeg, extractedFfprobe bool

	for _, f := range r.File {
		baseName := filepath.Base(f.Name)
		if baseName == "ffmpeg.exe" || baseName == "ffprobe.exe" {
			destPath := filepath.Join(localBinDir, baseName)
			
			rc, err := f.Open()
			if err != nil {
				return fmt.Errorf("failed to open file %s inside zip: %v", f.Name, err)
			}
			
			destFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				rc.Close()
				return fmt.Errorf("failed to create destination file %s: %v", destPath, err)
			}

			_, err = io.Copy(destFile, rc)
			destFile.Close()
			rc.Close()
			if err != nil {
				return fmt.Errorf("failed to extract file %s: %v", f.Name, err)
			}

			if baseName == "ffmpeg.exe" {
				extractedFfmpeg = true
			} else {
				extractedFfprobe = true
			}
		}
	}

	if !extractedFfmpeg || !extractedFfprobe {
		return fmt.Errorf("ffmpeg.exe or ffprobe.exe was not found inside the downloaded archive")
	}

	a.ffmpegPath = filepath.Join(localBinDir, "ffmpeg.exe")
	a.ffprobePath = filepath.Join(localBinDir, "ffprobe.exe")

	return nil
}

// SelectInputFile opens a native Open File Dialog and returns selected file path
func (a *App) SelectInputFile() (string, error) {
	file, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Source Media File",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "Media Files (*.mp4;*.mkv;*.webm;*.avi;*.mov;*.mp3;*.wav;*.ogg;*.flac;*.m4a)",
				Pattern:     "*.mp4;*.mkv;*.webm;*.avi;*.mov;*.mp3;*.wav;*.ogg;*.flac;*.m4a",
			},
			{
				DisplayName: "All Files (*.*)",
				Pattern:     "*.*",
			},
		},
	})
	return file, err
}

// SelectOutputFile opens a native Save File Dialog and returns selected path
func (a *App) SelectOutputFile(defaultName string, filterPattern string) (string, error) {
	file, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Select Destination File",
		DefaultFilename: defaultName,
		Filters: []runtime.FileFilter{
			{
				DisplayName: fmt.Sprintf("Target File (%s)", filterPattern),
				Pattern:     filterPattern,
			},
		},
	})
	return file, err
}

// GetMediaInfo uses ffprobe to inspect properties of selected media file
func (a *App) GetMediaInfo(path string) (*MediaInfo, error) {
	if a.ffprobePath == "" {
		if ok, _ := a.CheckFFmpeg(); !ok {
			return nil, fmt.Errorf("ffprobe path not configured")
		}
	}

	cmd := exec.Command(a.ffprobePath, "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", path)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("ffprobe execution failed: %v. Stderr: %s", err, stderr.String())
	}

	var raw FFProbeResult
	err = json.Unmarshal(stdout.Bytes(), &raw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe stdout: %v", err)
	}

	info := &MediaInfo{
		Path: path,
	}

	if raw.Format.Duration != "" {
		if d, err := strconv.ParseFloat(raw.Format.Duration, 64); err == nil {
			info.Duration = d
		}
	}

	if raw.Format.Size != "" {
		if s, err := strconv.ParseInt(raw.Format.Size, 10, 64); err == nil {
			info.Size = s
		}
	}

	info.Format = raw.Format.FormatName

	for _, stream := range raw.Streams {
		if stream.CodecType == "video" && !info.HasVideo {
			info.HasVideo = true
			info.VideoCodec = stream.CodecName
			info.Resolution = fmt.Sprintf("%dx%d", stream.Width, stream.Height)

			if stream.RFrameRate != "" {
				parts := strings.Split(stream.RFrameRate, "/")
				if len(parts) == 2 {
					num, _ := strconv.ParseFloat(parts[0], 64)
					den, _ := strconv.ParseFloat(parts[1], 64)
					if den > 0 {
						info.FrameRate = fmt.Sprintf("%.2f fps", num/den)
					}
				} else {
					info.FrameRate = stream.RFrameRate + " fps"
				}
			}
		} else if stream.CodecType == "audio" && !info.HasAudio {
			info.HasAudio = true
			info.AudioCodec = stream.CodecName
			info.AudioChannels = stream.Channels
		}
	}

	return info, nil
}

// supportsEncoder checks if a specific encoder is supported by the host system
func (a *App) supportsEncoder(encoder string) bool {
	if a.ffmpegPath == "" {
		if ok, _ := a.CheckFFmpeg(); !ok {
			return false
		}
	}
	cmd := exec.Command(a.ffmpegPath, "-f", "lavfi", "-i", "color=c=black:s=64x64:d=0.04", "-c:v", encoder, "-f", "null", "-")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Run() == nil
}

// GetAvailableHardwareEncoders queries FFmpeg to determine which hardware encoders actually work on the system.
// It caches the result after the first check.
func (a *App) GetAvailableHardwareEncoders() map[string]bool {
	a.hwEncodersMu.Lock()
	defer a.hwEncodersMu.Unlock()

	if a.hwEncoders != nil {
		return a.hwEncoders
	}

	a.hwEncoders = map[string]bool{
		"nvenc": false,
		"amf":   false,
		"qsv":   false,
	}

	if a.ffmpegPath == "" {
		if ok, _ := a.CheckFFmpeg(); !ok {
			return a.hwEncoders
		}
	}

	probe := func(encoder string) bool {
		cmd := exec.Command(a.ffmpegPath, "-f", "lavfi", "-i", "color=c=black:s=64x64:d=0.04", "-c:v", encoder, "-f", "null", "-")
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		return cmd.Run() == nil
	}

	if probe("h264_nvenc") {
		a.hwEncoders["nvenc"] = true
	}
	if probe("h264_amf") {
		a.hwEncoders["amf"] = true
	}
	if probe("h264_qsv") {
		a.hwEncoders["qsv"] = true
	}

	return a.hwEncoders
}

// StartConversion executes FFmpeg with selected parameters and parses progress updates
func (a *App) StartConversion(params ConversionParams) error {
	a.wasCancelled.Store(false)

	if a.ffmpegPath == "" {
		if ok, _ := a.CheckFFmpeg(); !ok {
			return fmt.Errorf("ffmpeg not found")
		}
	}

	// Fetch input details to know duration for percentage calculations
	info, err := a.GetMediaInfo(params.InputPath)
	if err != nil {
		return fmt.Errorf("failed inspecting input file metadata: %v", err)
	}

	totalDuration := info.Duration

	// Build FFmpeg argument list
	var args []string

	// Overwrite existing files
	args = append(args, "-y")

	// Hardware decoding (must go before -i)
	if !params.NoReencode && params.HwAccel != "none" && params.HwAccel != "" {
		args = append(args, "-hwaccel", "auto")
	}

	// Set inputs
	args = append(args, "-i", params.InputPath)

	// Trim settings
	if params.EnableTrim {
		if params.TrimStart != "" {
			args = append(args, "-ss", params.TrimStart)
		}
		if params.TrimEnd != "" {
			args = append(args, "-to", params.TrimEnd)
		}
	}

	isAudioOnlyOutput := params.TargetFormat == "mp3" || params.TargetFormat == "wav"

	if params.NoReencode {
		if params.ExtractAudio || isAudioOnlyOutput {
			args = append(args, "-vn", "-c:a", "copy")
		} else {
			if params.StripAudio {
				args = append(args, "-an")
			} else {
				args = append(args, "-c:a", "copy")
			}
			args = append(args, "-c:v", "copy")
		}
	} else {
		// Re-encoding logic
		selectedEncoder := ""
		selectedHw := ""

		if params.HwAccel != "none" && params.HwAccel != "" {
			available := a.GetAvailableHardwareEncoders()
			if params.HwAccel == "auto" {
				if available["nvenc"] {
					selectedHw = "nvenc"
				} else if available["amf"] {
					selectedHw = "amf"
				} else if available["qsv"] {
					selectedHw = "qsv"
				}
			} else {
				if available[params.HwAccel] {
					selectedHw = params.HwAccel
				}
			}
		}

		if params.ExtractAudio || isAudioOnlyOutput {
			// Strip video entirely
			args = append(args, "-vn")

			if params.TargetFormat == "mp3" {
				args = append(args, "-c:a", "libmp3lame")
				switch params.QualityPreset {
				case "high":
					args = append(args, "-b:a", "320k")
				case "low":
					args = append(args, "-b:a", "128k")
				default:
					args = append(args, "-b:a", "192k")
				}
			} else if params.TargetFormat == "wav" {
				args = append(args, "-c:a", "pcm_s16le")
			} else {
				// default to standard copying or converting audio for WebM / MP4 extraction
				args = append(args, "-c:a", "aac", "-b:a", "192k")
			}
		} else {
			// Video conversion options
			if params.TargetFormat == "webm" {
				// Check for VP9 hardware encoder
				if selectedHw == "nvenc" && a.supportsEncoder("vp9_nvenc") {
					selectedEncoder = "vp9_nvenc"
				} else if selectedHw == "qsv" && a.supportsEncoder("vp9_qsv") {
					selectedEncoder = "vp9_qsv"
				} else {
					selectedEncoder = "libvpx-vp9"
				}

				args = append(args, "-c:v", selectedEncoder)

				if params.StripAudio {
					args = append(args, "-an")
				} else {
					args = append(args, "-c:a", "libopus", "-b:a", "128k")
				}

				// Quality parameters
				if selectedEncoder == "vp9_nvenc" {
					switch params.QualityPreset {
					case "high":
						args = append(args, "-rc", "vbr", "-cq", "20", "-preset", "slow")
					case "low":
						args = append(args, "-rc", "vbr", "-cq", "30", "-preset", "fast")
					default:
						args = append(args, "-rc", "vbr", "-cq", "25", "-preset", "medium")
					}
				} else if selectedEncoder == "vp9_qsv" {
					switch params.QualityPreset {
					case "high":
						args = append(args, "-global_quality", "20", "-preset", "slow")
					case "low":
						args = append(args, "-global_quality", "30", "-preset", "fast")
					default:
						args = append(args, "-global_quality", "25", "-preset", "medium")
					}
				} else {
					// software VP9
					switch params.QualityPreset {
					case "high":
						args = append(args, "-crf", "20", "-b:v", "0", "-deadline", "good", "-cpu-used", "2")
					case "low":
						args = append(args, "-crf", "40", "-b:v", "0", "-deadline", "realtime", "-cpu-used", "5")
					default: // medium
						args = append(args, "-crf", "30", "-b:v", "0", "-deadline", "good", "-cpu-used", "3")
					}
				}
			} else {
				// Default to H.264 (MP4 / MKV)
				if selectedHw == "nvenc" {
					selectedEncoder = "h264_nvenc"
				} else if selectedHw == "amf" {
					selectedEncoder = "h264_amf"
				} else if selectedHw == "qsv" {
					selectedEncoder = "h264_qsv"
				} else {
					selectedEncoder = "libx264"
				}

				args = append(args, "-c:v", selectedEncoder)

				if params.StripAudio {
					args = append(args, "-an")
				} else {
					args = append(args, "-c:a", "aac", "-b:a", "192k")
				}

				// Quality parameters
				if selectedEncoder == "h264_nvenc" {
					switch params.QualityPreset {
					case "high":
						args = append(args, "-rc", "vbr", "-cq", "18", "-preset", "slow")
					case "low":
						args = append(args, "-rc", "vbr", "-cq", "28", "-preset", "fast")
					default:
						args = append(args, "-rc", "vbr", "-cq", "23", "-preset", "medium")
					}
				} else if selectedEncoder == "h264_amf" {
					switch params.QualityPreset {
					case "high":
						args = append(args, "-qp_i", "18", "-qp_p", "18", "-quality", "quality")
					case "low":
						args = append(args, "-qp_i", "28", "-qp_p", "28", "-quality", "speed")
					default:
						args = append(args, "-qp_i", "23", "-qp_p", "23", "-quality", "balanced")
					}
				} else if selectedEncoder == "h264_qsv" {
					switch params.QualityPreset {
					case "high":
						args = append(args, "-global_quality", "18", "-preset", "slow")
					case "low":
						args = append(args, "-global_quality", "28", "-preset", "fast")
					default:
						args = append(args, "-global_quality", "23", "-preset", "medium")
					}
				} else {
					// software H.264
					switch params.QualityPreset {
					case "high":
						args = append(args, "-crf", "18", "-preset", "slow")
					case "low":
						args = append(args, "-crf", "28", "-preset", "fast")
					default: // medium
						args = append(args, "-crf", "23", "-preset", "medium")
					}
				}
			}

			// Resolution scaling
			if params.ResolutionScale != "source" {
				switch params.ResolutionScale {
				case "1080p":
					args = append(args, "-vf", "scale=-2:1080")
				case "720p":
					args = append(args, "-vf", "scale=-2:720")
				case "480p":
					args = append(args, "-vf", "scale=-2:480")
				}
			}
		}
	}

	// Add output file
	args = append(args, "-progress", "pipe:1", params.OutputPath)

	cmd := exec.Command(a.ffmpegPath, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed creating stdout pipe: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed creating stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed starting FFmpeg process: %v", err)
	}

	a.mu.Lock()
	a.activeCmd = cmd
	a.mu.Unlock()

	// Capture stderr for the terminal diagnostics console in real-time
	go func() {
		errScanner := bufio.NewScanner(stderr)
		for errScanner.Scan() {
			line := errScanner.Text()
			runtime.EventsEmit(a.ctx, "conversion:log", line)
		}
	}()

	// Capture stdout for progress calculations in real-time
	go func() {
		outScanner := bufio.NewScanner(stdout)
		progressData := ProgressData{
			Frame:   "-",
			Fps:     "-",
			Speed:   "-",
			Bitrate: "-",
		}

		for outScanner.Scan() {
			line := outScanner.Text()
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])

			switch key {
			case "frame":
				progressData.Frame = val
			case "fps":
				progressData.Fps = val
			case "speed":
				progressData.Speed = val
			case "bitrate":
				progressData.Bitrate = val
			case "out_time_us":
				if us, err := strconv.ParseInt(val, 10, 64); err == nil {
					progressData.OutTimeMs = us / 1000
					if totalDuration > 0 {
						pct := (float64(progressData.OutTimeMs) / 1000.0) / totalDuration * 100.0
						progressData.Percent = int(pct)
						if progressData.Percent > 100 {
							progressData.Percent = 100
						}
						if progressData.Percent < 0 {
							progressData.Percent = 0
						}
					}
				}
			case "progress":
				runtime.EventsEmit(a.ctx, "conversion:progress", progressData)
			}
		}
	}()

	err = cmd.Wait()

	a.mu.Lock()
	a.activeCmd = nil
	a.mu.Unlock()

	if err != nil {
		if a.wasCancelled.Load() {
			return fmt.Errorf("cancelled")
		}
		return fmt.Errorf("ffmpeg process exited with code: %v", err)
	}

	return nil
}

// CancelConversion aborts the running FFmpeg process
func (a *App) CancelConversion() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.activeCmd != nil && a.activeCmd.Process != nil {
		a.wasCancelled.Store(true)
		err := a.activeCmd.Process.Kill()
		return err
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION: Universal Conversion Router
// ─────────────────────────────────────────────────────────────────────────────

// ConversionResult is returned by HandleConversion to the frontend.
type ConversionResult struct {
	OutputPath string `json:"outputPath"`
	Category   string `json:"category"` // "media", "image", "data", "document"
}

// HandleConversion is the single Wails-exposed entry point for all file types.
// It inspects the source extension and routes to the correct handler.
func (a *App) HandleConversion(inputPath, targetExt string) (*ConversionResult, error) {
	targetExt = strings.ToLower(strings.TrimPrefix(targetExt, "."))
	srcExt    := strings.ToLower(strings.TrimPrefix(filepath.Ext(inputPath), "."))

	// Derive a sensible output path: same directory, same stem, new extension.
	stem      := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	outputDir := filepath.Dir(inputPath)
	outputPath := filepath.Join(outputDir, stem+"_converted."+targetExt)

	switch srcExt {
	// ── Video ──────────────────────────────────────────────────────────────
	case "mp4", "mkv", "avi", "mov", "webm":
		return a.routeToFFmpeg(inputPath, outputPath, targetExt, "media")

	// ── Audio ──────────────────────────────────────────────────────────────
	case "mp3", "wav", "m4a", "flac", "ogg", "aac":
		return a.routeToFFmpeg(inputPath, outputPath, targetExt, "media")

	// ── Image ──────────────────────────────────────────────────────────────
	case "png", "jpg", "jpeg", "webp":
		out, err := convertImageNative(inputPath, outputPath, targetExt)
		if err != nil {
			return nil, err
		}
		return &ConversionResult{OutputPath: out, Category: "image"}, nil

	// ── Data ───────────────────────────────────────────────────────────────
	case "csv", "json":
		out, err := convertDataNative(inputPath, outputPath, srcExt, targetExt)
		if err != nil {
			return nil, err
		}
		return &ConversionResult{OutputPath: out, Category: "data"}, nil

	// ── Document ───────────────────────────────────────────────────────────
	case "md", "markdown":
		out, err := convertMarkdownNative(inputPath, outputPath)
		if err != nil {
			return nil, err
		}
		return &ConversionResult{OutputPath: out, Category: "document"}, nil

	default:
		return nil, fmt.Errorf("unsupported source format: .%s", srcExt)
	}
}

// routeToFFmpeg wraps the existing FFmpeg pipeline for media files.
// It builds a minimal ConversionParams and calls StartConversion.
func (a *App) routeToFFmpeg(inputPath, outputPath, targetExt, category string) (*ConversionResult, error) {
	params := ConversionParams{
		InputPath:       inputPath,
		OutputPath:      outputPath,
		TargetFormat:    targetExt,
		ResolutionScale: "source",
		QualityPreset:   "medium",
		HwAccel:         "auto",
	}
	if err := a.StartConversion(params); err != nil {
		return nil, fmt.Errorf("media conversion failed: %w", err)
	}
	return &ConversionResult{OutputPath: outputPath, Category: category}, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION: Native Image Converter
// Supports: PNG ↔ JPEG, PNG/JPEG → WEBP (decode only; encodes as PNG fallback)
// ─────────────────────────────────────────────────────────────────────────────

// convertImageNative decodes the source image with stdlib decoders and
// re-encodes it into the target format. Supports PNG, JPEG, and WEBP (decode).
func convertImageNative(inputPath, outputPath, targetExt string) (string, error) {
	// ── Open source ──────────────────────────────────────────────────────────
	srcFile, err := os.Open(inputPath)
	if err != nil {
		return "", fmt.Errorf("image: cannot open source file: %w", err)
	}
	defer srcFile.Close()

	// ── Decode ───────────────────────────────────────────────────────────────
	srcExt := strings.ToLower(strings.TrimPrefix(filepath.Ext(inputPath), "."))
	var img image.Image

	switch srcExt {
	case "webp":
		img, err = webp.Decode(srcFile)
	case "jpg", "jpeg":
		img, err = jpeg.Decode(srcFile)
	case "png":
		img, err = png.Decode(srcFile)
	default:
		// Fallback: let the stdlib sniff the format
		img, _, err = image.Decode(srcFile)
	}
	if err != nil {
		return "", fmt.Errorf("image: decode failed (%s): %w", srcExt, err)
	}

	// ── Create output ────────────────────────────────────────────────────────
	dstFile, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("image: cannot create output file: %w", err)
	}
	defer dstFile.Close()

	// ── Encode ───────────────────────────────────────────────────────────────
	switch targetExt {
	case "jpg", "jpeg":
		opts := &jpeg.Options{Quality: 92}
		if err := jpeg.Encode(dstFile, img, opts); err != nil {
			return "", fmt.Errorf("image: JPEG encode failed: %w", err)
		}
	case "png":
		enc := png.Encoder{CompressionLevel: png.DefaultCompression}
		if err := enc.Encode(dstFile, img); err != nil {
			return "", fmt.Errorf("image: PNG encode failed: %w", err)
		}
	case "webp":
		// The golang.org/x/image/webp package is decode-only.
		// We encode as PNG and rename the output accordingly.
		dstFile.Close()
		pngPath := strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + ".png"
		pngFile, err := os.Create(pngPath)
		if err != nil {
			return "", fmt.Errorf("image: cannot create PNG fallback output: %w", err)
		}
		defer pngFile.Close()
		enc := png.Encoder{CompressionLevel: png.DefaultCompression}
		if err := enc.Encode(pngFile, img); err != nil {
			return "", fmt.Errorf("image: PNG fallback encode failed: %w", err)
		}
		_ = os.Remove(outputPath) // remove the empty placeholder
		return pngPath, nil
	default:
		return "", fmt.Errorf("image: unsupported target format: .%s", targetExt)
	}

	return outputPath, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION: Native Data Converter  (CSV ↔ JSON)
// ─────────────────────────────────────────────────────────────────────────────

// convertDataNative handles two-way conversion between CSV and JSON.
// CSV → JSON produces a JSON array of objects keyed by the header row.
// JSON → CSV expects a JSON array of flat objects and emits a CSV file.
func convertDataNative(inputPath, outputPath, srcExt, targetExt string) (string, error) {
	switch {
	case srcExt == "csv" && targetExt == "json":
		return csvToJSON(inputPath, outputPath)
	case srcExt == "json" && targetExt == "csv":
		return jsonToCSV(inputPath, outputPath)
	default:
		return "", fmt.Errorf("data: unsupported conversion .%s → .%s", srcExt, targetExt)
	}
}

// csvToJSON reads a CSV file and writes a JSON array of row-objects.
func csvToJSON(inputPath, outputPath string) (string, error) {
	srcFile, err := os.Open(inputPath)
	if err != nil {
		return "", fmt.Errorf("csv→json: cannot open source: %w", err)
	}
	defer srcFile.Close()

	reader := csv.NewReader(srcFile)
	reader.TrimLeadingSpace = true

	headers, err := reader.Read()
	if err != nil {
		return "", fmt.Errorf("csv→json: failed reading header row: %w", err)
	}

	var rows []map[string]string
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("csv→json: parse error: %w", err)
		}
		row := make(map[string]string, len(headers))
		for i, h := range headers {
			if i < len(record) {
				row[h] = record[i]
			} else {
				row[h] = ""
			}
		}
		rows = append(rows, row)
	}

	dstFile, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("csv→json: cannot create output: %w", err)
	}
	defer dstFile.Close()

	enc := json.NewEncoder(dstFile)
	enc.SetIndent("", "  ")
	if err := enc.Encode(rows); err != nil {
		return "", fmt.Errorf("csv→json: JSON encode failed: %w", err)
	}

	return outputPath, nil
}

// jsonToCSV reads a JSON array of flat objects and writes a CSV file.
func jsonToCSV(inputPath, outputPath string) (string, error) {
	srcBytes, err := os.ReadFile(inputPath)
	if err != nil {
		return "", fmt.Errorf("json→csv: cannot read source: %w", err)
	}

	// Accept a JSON array of objects where each object is map[string]any.
	var rows []map[string]any
	if err := json.Unmarshal(srcBytes, &rows); err != nil {
		return "", fmt.Errorf("json→csv: JSON parse failed (expected array of objects): %w", err)
	}
	if len(rows) == 0 {
		return "", fmt.Errorf("json→csv: input JSON array is empty")
	}

	// Collect ordered keys from the first object as headers.
	headers := make([]string, 0)
	headerSeen := map[string]bool{}
	for _, row := range rows {
		for k := range row {
			if !headerSeen[k] {
				headers = append(headers, k)
				headerSeen[k] = true
			}
		}
	}

	dstFile, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("json→csv: cannot create output: %w", err)
	}
	defer dstFile.Close()

	writer := csv.NewWriter(dstFile)
	defer writer.Flush()

	if err := writer.Write(headers); err != nil {
		return "", fmt.Errorf("json→csv: failed writing header row: %w", err)
	}

	for _, row := range rows {
		record := make([]string, len(headers))
		for i, h := range headers {
			if v, ok := row[h]; ok {
				record[i] = fmt.Sprintf("%v", v)
			}
		}
		if err := writer.Write(record); err != nil {
			return "", fmt.Errorf("json→csv: failed writing row: %w", err)
		}
	}

	if err := writer.Error(); err != nil {
		return "", fmt.Errorf("json→csv: CSV writer error: %w", err)
	}

	return outputPath, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// SECTION: Native Markdown → HTML Converter
// ─────────────────────────────────────────────────────────────────────────────

// htmlTemplate wraps the rendered Markdown body in a clean, self-contained HTML page.
const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>%s</title>
  <style>
    body { max-width: 800px; margin: 40px auto; padding: 0 24px;
           font-family: system-ui, sans-serif; line-height: 1.7; color: #1a1a1a; }
    pre  { background: #f4f4f4; padding: 16px; border-radius: 6px; overflow-x: auto; }
    code { font-family: 'JetBrains Mono', monospace; font-size: .9em; }
    blockquote { border-left: 4px solid #ccc; margin: 0; padding-left: 16px; color: #555; }
    img  { max-width: 100%%; }
    table { border-collapse: collapse; width: 100%%; }
    th, td { border: 1px solid #ddd; padding: 8px 12px; text-align: left; }
    th { background: #f9f9f9; }
  </style>
</head>
<body>
%s
</body>
</html>
`

// convertMarkdownNative parses a Markdown file with goldmark and writes
// a fully-formed HTML file. GFM tables, strikethrough, and autolinks are enabled.
func convertMarkdownNative(inputPath, outputPath string) (string, error) {
	srcBytes, err := os.ReadFile(inputPath)
	if err != nil {
		return "", fmt.Errorf("md→html: cannot read source: %w", err)
	}

	// Configure goldmark with GitHub-Flavoured Markdown extensions.
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,        // tables, strikethrough, task lists, autolinks
			extension.Footnote,
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
		),
	)

	var bodyBuf bytes.Buffer
	if err := md.Convert(srcBytes, &bodyBuf); err != nil {
		return "", fmt.Errorf("md→html: goldmark conversion failed: %w", err)
	}

	// Use the filename (without extension) as the page title.
	title := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	full  := fmt.Sprintf(htmlTemplate, title, bodyBuf.String())

	if err := os.WriteFile(outputPath, []byte(full), 0644); err != nil {
		return "", fmt.Errorf("md→html: cannot write output: %w", err)
	}

	return outputPath, nil
}
