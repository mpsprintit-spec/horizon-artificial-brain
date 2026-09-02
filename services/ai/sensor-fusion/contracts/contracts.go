package contracts

import (
	"context"

	"github.com/project-horizon/horizon-core/services/ai/sensor-fusion/model"
)

// SensorFusionEngine receives all sensor data, coordinates the fusion pipeline, and produces a unified world state.
type SensorFusionEngine interface {
	Fuse(context.Context, []model.SensorFrame) (model.WorldState, error)
}

// IMUFusion combines accelerometer, gyroscope, and magnetometer data into orientation, motion, and velocity state.
type IMUFusion interface {
	FuseIMU(context.Context, []model.SensorFrame) (model.SensorState, error)
}

// GPSFusion combines GPS, IMU, and vision-derived input into an improved position estimate.
type GPSFusion interface {
	FuseGPS(context.Context, []model.SensorFrame, model.SensorState) (model.Position, error)
}

// VIOEstimator estimates position and rotation from camera and IMU inputs.
type VIOEstimator interface {
	EstimateVIO(context.Context, []model.SensorFrame) (model.Pose, error)
}

// NoiseFilter prepares stable sensor data before the future vision and navigation pipeline consumes it.
type NoiseFilter interface {
	FilterNoise(context.Context, []model.SensorFrame) ([]model.SensorFrame, error)
}

// StateEstimator estimates position, velocity, orientation, and confidence score from sensor state.
type StateEstimator interface {
	EstimateState(context.Context, model.SensorState) (model.WorldState, error)
}
