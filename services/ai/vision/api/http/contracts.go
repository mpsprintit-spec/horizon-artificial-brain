package http

import (
	"context"

	"github.com/project-horizon/horizon-core/services/ai/vision/internal/model"
)

// AnalyzeFrameHandler defines the HTTP contract for frame analysis.
type AnalyzeFrameHandler interface {
	AnalyzeFrame(context.Context, model.Frame) (model.VisionResult, error)
}

// AnalyzeStreamHandler defines the HTTP contract for stream analysis.
type AnalyzeStreamHandler interface {
	AnalyzeStream(context.Context, model.VideoStream) (model.VisionResult, error)
}

// GetVisionResultHandler defines the HTTP contract for retrieving a VisionResult.
type GetVisionResultHandler interface {
	GetVisionResult(context.Context, string) (model.VisionResult, error)
}
