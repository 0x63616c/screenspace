package video

import "context"

// MockProber is a test double for the Prober interface.
// Set Fn fields for methods you need in each test scenario.
type MockProber struct {
	ProbeFn             func(ctx context.Context, path string) (*ProbeResult, error)
	GenerateThumbnailFn func(ctx context.Context, inputPath, outputPath string) error
	GeneratePreviewFn   func(ctx context.Context, inputPath, outputPath string) error
}

func (m *MockProber) Probe(ctx context.Context, path string) (*ProbeResult, error) {
	if m.ProbeFn != nil {
		return m.ProbeFn(ctx, path)
	}
	return &ProbeResult{}, nil
}

func (m *MockProber) GenerateThumbnail(ctx context.Context, inputPath, outputPath string) error {
	if m.GenerateThumbnailFn != nil {
		return m.GenerateThumbnailFn(ctx, inputPath, outputPath)
	}
	return nil
}

func (m *MockProber) GeneratePreview(ctx context.Context, inputPath, outputPath string) error {
	if m.GeneratePreviewFn != nil {
		return m.GeneratePreviewFn(ctx, inputPath, outputPath)
	}
	return nil
}

var _ Prober = (*MockProber)(nil)
