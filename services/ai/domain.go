package main

import (
	"errors"
	"math"
	"strings"
	"time"
)

// SignalType represents one of the HC-002 real-time data input classes.
type SignalType string

const (
	SignalVision    SignalType = "vision"
	SignalAudio     SignalType = "audio"
	SignalTelemetry SignalType = "telemetry"
	SignalSensor    SignalType = "sensor"
	SignalLocation  SignalType = "location"
	SignalUser      SignalType = "user_context"
)

// Signal is an immutable input sample presented to the canonical neural runtime.
type Signal struct {
	Type       SignalType
	Source     string
	Confidence float64
	Timestamp  time.Time
	Payload    map[string]float64
}

// Validate ensures the signal is safe and meaningful before processing.
func (s Signal) Validate() error {
	switch s.Type {
	case SignalVision, SignalAudio, SignalTelemetry, SignalSensor, SignalLocation, SignalUser:
	default:
		return errors.New("unsupported signal type")
	}
	if strings.TrimSpace(s.Source) == "" {
		return errors.New("signal source is required")
	}
	if math.IsNaN(s.Confidence) || s.Confidence < 0 || s.Confidence > 1 {
		return errors.New("signal confidence must be between 0 and 1")
	}
	if s.Timestamp.IsZero() {
		return errors.New("signal timestamp is required")
	}
	return nil
}
