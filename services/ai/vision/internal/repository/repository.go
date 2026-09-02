package repository

import (
	"context"

	"github.com/project-horizon/horizon-core/services/ai/vision/internal/model"
)

// VisionResultRepository stores and retrieves Vision AI analysis metadata.
type VisionResultRepository interface {
	SaveVisionResult(context.Context, model.VisionResult) error
	GetVisionResult(context.Context, string) (model.VisionResult, error)
}
