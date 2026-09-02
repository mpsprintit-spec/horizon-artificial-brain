package grpc

import (
	"context"

	"github.com/project-horizon/horizon-core/services/ai/vision/internal/model"
)

// AnalyzeFrameService defines the gRPC contract for frame analysis.
type AnalyzeFrameService interface {
	AnalyzeFrame(context.Context, model.Frame) (model.VisionResult, error)
}

// AnalyzeStreamService defines the gRPC contract for stream analysis.
type AnalyzeStreamService interface {
	AnalyzeStream(context.Context, model.VideoStream) (model.VisionResult, error)
}

// GetVisionResultService defines the gRPC contract for retrieving a VisionResult.
type GetVisionResultService interface {
	GetVisionResult(context.Context, string) (model.VisionResult, error)
}
