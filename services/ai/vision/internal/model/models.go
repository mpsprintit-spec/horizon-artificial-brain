package model

import "time"

// Frame is a single image frame from glasses, drone, or another camera source.
type Frame struct {
	ID          string
	SourceID    string
	MissionID   string
	CapturedAt  time.Time
	ContentType string
	URI         string
	Metadata    map[string]string
}

// VideoStream is an ordered video stream with source and mission metadata.
type VideoStream struct {
	ID        string
	SourceID  string
	MissionID string
	StartedAt time.Time
	Metadata  map[string]string
}

// Detection describes object, person, target, or obstacle metadata from visual input.
type Detection struct {
	ID         string
	FrameID    string
	Label      string
	Confidence float64
	Bounds     BoundingBox
	Attributes map[string]string
}

// BoundingBox identifies a rectangular region in a frame.
type BoundingBox struct {
	X      float64
	Y      float64
	Width  float64
	Height float64
}

// SceneContext describes semantic understanding of an environment or scene.
type SceneContext struct {
	ID          string
	FrameID     string
	MissionID   string
	Description string
	Labels      []string
	Confidence  float64
}

// DepthMap describes estimated distance or depth metadata from visual input.
type DepthMap struct {
	ID        string
	FrameID   string
	URI       string
	MinDepth  float64
	MaxDepth  float64
	MeanDepth float64
}

// OpticalFlow contains motion metadata between frames.
type OpticalFlow struct {
	ID           string
	FrameID      string
	PreviousID   string
	VectorURI    string
	MeanVelocity float64
}

// VisionResult is the aggregate result envelope for Vision AI analysis.
type VisionResult struct {
	ID          string
	FrameID     string
	StreamID    string
	MissionID   string
	Detections  []Detection
	Scene       SceneContext
	Depth       DepthMap
	OpticalFlow OpticalFlow
	Confidence  float64
	GeneratedAt time.Time
	ResultURI   string
	Metadata    map[string]string
}
