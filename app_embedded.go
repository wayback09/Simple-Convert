//go:build embedded

package main

import (
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

//go:embed binaries/ffmpeg.exe binaries/ffprobe.exe
var embeddedBinaries embed.FS

// extractEmbeddedBinaries extracts pre-bundled binaries on startup.
func (a *App) extractEmbeddedBinaries() (bool, error) {
	appDataDir, err := os.UserConfigDir()
	if err != nil {
		return false, err
	}

	localBinDir := filepath.Join(appDataDir, "hacdot-convert", "bin")
	err = os.MkdirAll(localBinDir, 0755)
	if err != nil {
		return false, err
	}

	ffmpegPath := filepath.Join(localBinDir, "ffmpeg.exe")
	ffprobePath := filepath.Join(localBinDir, "ffprobe.exe")

	// Unpack ffmpeg.exe using memory-efficient streaming
	err = extractEmbeddedFile("binaries/ffmpeg.exe", ffmpegPath)
	if err != nil {
		return false, fmt.Errorf("failed to extract embedded ffmpeg: %v", err)
	}

	// Unpack ffprobe.exe
	err = extractEmbeddedFile("binaries/ffprobe.exe", ffprobePath)
	if err != nil {
		return false, fmt.Errorf("failed to extract embedded ffprobe: %v", err)
	}

	a.ffmpegPath = ffmpegPath
	a.ffprobePath = ffprobePath

	return true, nil
}

// extractEmbeddedFile opens an embedded asset file and streams it to a destination file on disk
func extractEmbeddedFile(embeddedPath, destPath string) error {
	srcFile, err := embeddedBinaries.Open(embeddedPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	return err
}
