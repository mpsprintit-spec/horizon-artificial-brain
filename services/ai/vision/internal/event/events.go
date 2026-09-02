package event

import "time"

// Name is the canonical name of a Vision AI domain event.
type Name string

// Canonical Vision AI event names.
const (
	FrameCaptured      Name = "FrameCaptured"
	VideoFrameReceived Name = "VideoFrameReceived"
	FrameAnalyzed      Name = "FrameAnalyzed"
	ObjectDetected     Name = "ObjectDetected"
	SceneUnderstood    Name = "SceneUnderstood"
	DepthEstimated     Name = "DepthEstimated"
	OpticalFlowTracked Name = "OpticalFlowTracked"
)

// Event is the transport-neutral envelope for Vision AI events.
type Event struct {
	ID            string
	Name          Name
	Source        string
	MissionID     string
	CorrelationID string
	OccurredAt    time.Time
	Payload       map[string]string
}
