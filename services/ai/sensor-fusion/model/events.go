package model

import "time"

// EventName is the canonical name of a Sensor Fusion AI domain event.
type EventName string

// Sensor Fusion AI domain event names.
const (
	SensorFused              EventName = "SensorFused"
	NavigationStateEstimated EventName = "NavigationStateEstimated"
	NoiseFiltered            EventName = "NoiseFiltered"
	ObstacleTracked          EventName = "ObstacleTracked"
)

// SensorFusedEvent documents the future event emitted when sensor inputs are combined.
type SensorFusedEvent struct {
	ID         string
	MissionID  string
	WorldState WorldState
	OccurredAt time.Time
}

// NavigationStateEstimatedEvent documents the future event emitted when navigation state is estimated.
type NavigationStateEstimatedEvent struct {
	ID          string
	MissionID   string
	SensorState SensorState
	OccurredAt  time.Time
}

// NoiseFilteredEvent documents the future event emitted after unstable sensor data is prepared.
type NoiseFilteredEvent struct {
	ID         string
	MissionID  string
	Frames     []SensorFrame
	OccurredAt time.Time
}

// ObstacleTrackedEvent documents the future event emitted when an obstacle is tracked.
type ObstacleTrackedEvent struct {
	ID         string
	MissionID  string
	ObstacleID string
	Position   Position
	Confidence Confidence
	OccurredAt time.Time
}
