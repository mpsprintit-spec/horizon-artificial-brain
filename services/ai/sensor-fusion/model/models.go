package model

import "time"

// SensorFrame represents one normalized sensor sample frame from a device source.
type SensorFrame struct {
	ID           string
	SourceID     string
	SensorType   string
	MissionID    string
	CapturedAt   time.Time
	Measurements map[string]float64
	Metadata     map[string]string
}

// SensorState represents the current normalized state derived from sensor frames.
type SensorState struct {
	ID          string
	MissionID   string
	Frames      []SensorFrame
	Pose        Pose
	Velocity    Velocity
	Confidence  Confidence
	GeneratedAt time.Time
}

// Pose combines position and orientation for a device or world entity.
type Pose struct {
	Position    Position
	Orientation Orientation
}

// Orientation describes roll, pitch, yaw, and heading values.
type Orientation struct {
	Roll    float64
	Pitch   float64
	Yaw     float64
	Heading float64
}

// Velocity describes motion along three axes and an aggregate speed value.
type Velocity struct {
	X     float64
	Y     float64
	Z     float64
	Speed float64
}

// Position describes latitude, longitude, altitude, and accuracy values.
type Position struct {
	Latitude  float64
	Longitude float64
	Altitude  float64
	Accuracy  float64
}

// WorldState represents the unified state produced by the future fusion pipeline.
type WorldState struct {
	ID          string
	MissionID   string
	State       SensorState
	Obstacles   []string
	Confidence  Confidence
	GeneratedAt time.Time
	Metadata    map[string]string
}

// Confidence describes the quality score and explanation for estimated state.
type Confidence struct {
	Score       float64
	Explanation string
}
