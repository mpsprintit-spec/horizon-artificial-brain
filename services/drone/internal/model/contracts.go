package model

import "github.com/project-horizon/horizon-core/services/ai/domain"

type PathPlanner = domain.PathPlanner
type ObstacleAvoider = domain.ObstacleAvoider
type AutoTracker = domain.AutoTracker
type FlightStabilizer = domain.FlightStabilizer
type EnergyManager = domain.EnergyManager
type DroneCommandDispatcher = domain.DroneCommandDispatcher

type DroneAIService interface {
	PathPlanner
	ObstacleAvoider
	AutoTracker
	FlightStabilizer
	EnergyManager
	DroneCommandDispatcher
}
