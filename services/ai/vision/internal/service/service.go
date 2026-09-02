package service

import "github.com/project-horizon/horizon-core/services/ai/vision/internal/model"

// VisionService groups the public Vision AI analysis contracts.
type VisionService interface {
	model.FrameAnalyzer
	model.StreamAnalyzer
	model.ObjectDetector
	model.SceneUnderstanding
	model.DepthEstimator
	model.OpticalFlowTracker
	model.LowLightEnhancer
}
