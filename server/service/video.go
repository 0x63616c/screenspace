package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type VideoInfo struct {
	Width    int
	Height   int
	Duration float64
	Format   string
	Size     int64
}

type VideoService struct{}

func NewVideoService() *VideoService {
	return &VideoService{}
}

type ffprobeOutput struct {
	Streams []ffprobeStream `json:"streams"`
	Format  ffprobeFormat   `json:"format"`
}

type ffprobeStream struct {
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	CodecName string `json:"codec_name"`
}

type ffprobeFormat struct {
	Duration string `json:"duration"`
	Size     string `json:"size"`
}

func (v *VideoService) Probe(ctx context.Context, path string) (*VideoInfo, error) {
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		"-show_format",
		path,
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe: %w", err)
	}

	var probe ffprobeOutput
	if err := json.Unmarshal(out, &probe); err != nil {
		return nil, fmt.Errorf("parse ffprobe output: %w", err)
	}

	info := &VideoInfo{}

	for _, s := range probe.Streams {
		if s.Width > 0 && s.Height > 0 {
			info.Width = s.Width
			info.Height = s.Height
			info.Format = normalizeCodec(s.CodecName)
			break
		}
	}

	if probe.Format.Duration != "" {
		d, err := strconv.ParseFloat(probe.Format.Duration, 64)
		if err == nil {
			info.Duration = d
		}
	}

	if probe.Format.Size != "" {
		s, err := strconv.ParseInt(probe.Format.Size, 10, 64)
		if err == nil {
			info.Size = s
		}
	}

	return info, nil
}

func (v *VideoService) GenerateThumbnail(ctx context.Context, inputPath, outputPath string) error {
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y",
		"-i", inputPath,
		"-ss", "2",
		"-vframes", "1",
		"-q:v", "2",
		outputPath,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg thumbnail: %w: %s", err, string(out))
	}
	return nil
}

func (v *VideoService) GeneratePreview(ctx context.Context, inputPath, outputPath string) error {
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y",
		"-i", inputPath,
		"-t", "10",
		"-vf", "scale=-2:640",
		"-an",
		"-c:v", "libx264",
		"-preset", "fast",
		"-crf", "28",
		outputPath,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg preview: %w: %s", err, string(out))
	}
	return nil
}

func normalizeCodec(codec string) string {
	c := strings.ToLower(codec)
	switch {
	case c == "h264" || c == "avc" || c == "avc1" || strings.HasPrefix(c, "h264"):
		return "h264"
	case c == "h265" || c == "hevc" || c == "hev1" || strings.HasPrefix(c, "h265"):
		return "h265"
	default:
		return c
	}
}
