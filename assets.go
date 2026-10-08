package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
)

func (cfg apiConfig) ensureAssetsDir() error {
	if _, err := os.Stat(cfg.assetsRoot); os.IsNotExist(err) {
		return os.Mkdir(cfg.assetsRoot, 0755)
	}
	return nil
}

func getVideoAspectRatio(filePath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	
	var buf bytes.Buffer

	cmd.Stdout = &buf
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("issue buffering video data, %w", err)
	}

	var aspectRatio struct {
		Streams []struct {
		Width int
        Height int
		}
	}

	err = json.Unmarshal(buf.Bytes(), &aspectRatio)
	if err != nil {
        return "", fmt.Errorf("issue unmarshalling json, %w", err)
    }
	if len(aspectRatio.Streams) == 0 {
		return "", fmt.Errorf("expected at least one stream, got %d", len(aspectRatio.Streams))
	}
	
	width := float64(aspectRatio.Streams[0].Width)
	height := float64(aspectRatio.Streams[0].Height)
	fltRatio := width / height
	sixteenNine := 16.0 / 9.0
	nineSixteen := 9.0 / 16.0
	tolerance := 0.01
	strRatio := ""
	
	if math.Abs(fltRatio - sixteenNine) < tolerance {
		strRatio = "16:9"
	} else if math.Abs(fltRatio - nineSixteen) < tolerance {
		strRatio = "9:16"
	} else {
		strRatio = "other"
	}

	return strRatio, nil
}

func processVideoForFastStart(filePath string) (string, error) {
	processingPath := filePath + ".processing"

	cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", processingPath)

	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return processingPath, nil
}