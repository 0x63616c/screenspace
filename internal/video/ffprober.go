package video

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// FFProber implements Prober using ffprobe and ffmpeg.
type FFProber struct{}

// NewFFProber creates a new FFProber.
func NewFFProber() *FFProber {
	return &FFProber{}
}

type ffprobeOutput struct {
	Streams []ffprobeStream `json:"streams"`
	Format  ffprobeFormat   `json:"format"`
}

type ffprobeStream struct {
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	CodecName string `json:"codec_name"`
}

type ffprobeFormat struct {
	Duration string `json:"duration"`
	Size     string `json:"size"`
}

// Probe runs ffprobe on the given path and returns video metadata.
func (p *FFProber) Probe(ctx context.Context, path string) (*ProbeResult, error) {
	cmd := exec.CommandContext(ctx, "ffprobe", //nolint:gosec // path is internal, not user-controlled
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

	info := &ProbeResult{}

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

// GenerateThumbnail extracts a single frame as a JPEG thumbnail.
func (p *FFProber) GenerateThumbnail(ctx context.Context, inputPath, outputPath string) error {
	cmd := exec.CommandContext(ctx, "ffmpeg", //nolint:gosec // paths are internal
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

// GeneratePreview creates a 10-second scaled-down preview clip.
func (p *FFProber) GeneratePreview(ctx context.Context, inputPath, outputPath string) error {
	cmd := exec.CommandContext(ctx, "ffmpeg", //nolint:gosec // paths are internal
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
