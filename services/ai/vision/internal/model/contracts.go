package model

import "context"

// FrameAnalyzer analyzes one frame and returns a VisionResult.
type FrameAnalyzer interface {
	AnalyzeFrame(context.Context, Frame) (VisionResult, error)
}

// StreamAnalyzer analyzes a video stream and returns a VisionResult.
type StreamAnalyzer interface {
	AnalyzeStream(context.Context, VideoStream) (VisionResult, error)
}

// ObjectDetector detects objects from a frame.
type ObjectDetector interface {
	DetectObjects(context.Context, Frame) ([]Detection, error)
}

// SceneUnderstanding describes scene understanding behavior for a frame.
type SceneUnderstanding interface {
	UnderstandScene(context.Context, Frame) (SceneContext, error)
}

// DepthEstimator estimates depth metadata from a frame.
type DepthEstimator interface {
	EstimateDepth(context.Context, Frame) (DepthMap, error)
}

// OpticalFlowTracker tracks motion metadata between frames.
type OpticalFlowTracker interface {
	TrackOpticalFlow(context.Context, Frame, Frame) (OpticalFlow, error)
}

// LowLightEnhancer prepares low-light frames before analysis.
type LowLightEnhancer interface {
	EnhanceLowLight(context.Context, Frame) (Frame, error)
}
