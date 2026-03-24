package video

import "context"

// ProbeResult holds metadata about a probed video file.
type ProbeResult struct {
	Width    int
	Height   int
	Duration float64
	Size     int64
	Format   string
}

// Prober probes video files for metadata and generates derivatives.
type Prober interface {
	Probe(ctx context.Context, path string) (*ProbeResult, error)
	GenerateThumbnail(ctx context.Context, inputPath, outputPath string) error
	GeneratePreview(ctx context.Context, inputPath, outputPath string) error
}
