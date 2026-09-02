package service

import "github.com/project-horizon/horizon-core/services/ai/sensor-fusion/contracts"

type SensorFusionService interface {
	contracts.SensorFusionEngine
	contracts.IMUFusion
	contracts.GPSFusion
	contracts.VIOEstimator
	contracts.NoiseFilter
	contracts.StateEstimator
}
