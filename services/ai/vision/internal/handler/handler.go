package handler

import (
	"context"

	"github.com/project-horizon/horizon-core/services/ai/vision/internal/event"
	"github.com/project-horizon/horizon-core/services/ai/vision/internal/model"
)

// FrameHandler handles frame analysis requests.
type FrameHandler interface {
	AnalyzeFrame(context.Context, model.Frame) (model.VisionResult, error)
}

// StreamHandler handles stream analysis requests.
type StreamHandler interface {
	AnalyzeStream(context.Context, model.VideoStream) (model.VisionResult, error)
}

// EventHandler handles Vision AI domain events.
type EventHandler interface {
	HandleEvent(context.Context, event.Event) error
}
