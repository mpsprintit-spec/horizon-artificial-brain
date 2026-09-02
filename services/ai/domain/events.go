package domain

import "time"

// EventName is the canonical name of an HC-002 domain event.
type EventName string

// Canonical HC-002 event names.
const (
	EventFrameCaptured               EventName = "FrameCaptured"
	EventVideoFrameReceived          EventName = "VideoFrameReceived"
	EventAudioReceived               EventName = "AudioReceived"
	EventSensorDataReceived          EventName = "SensorDataReceived"
	EventTelemetryReceived           EventName = "TelemetryReceived"
	EventLocationUpdated             EventName = "LocationUpdated"
	EventUserContextUpdated          EventName = "UserContextUpdated"
	EventMissionContextUpdated       EventName = "MissionContextUpdated"
	EventSystemStatusUpdated         EventName = "SystemStatusUpdated"
	EventFrameAnalyzed               EventName = "FrameAnalyzed"
	EventObjectDetected              EventName = "ObjectDetected"
	EventSceneUnderstood             EventName = "SceneUnderstood"
	EventDepthEstimated              EventName = "DepthEstimated"
	EventOpticalFlowTracked          EventName = "OpticalFlowTracked"
	EventObstacleDetected            EventName = "ObstacleDetected"
	EventSensorFused                 EventName = "SensorFused"
	EventNavigationStateEstimated    EventName = "NavigationStateEstimated"
	EventObstacleTracked             EventName = "ObstacleTracked"
	EventSituationAssessed           EventName = "SituationAssessed"
	EventRiskAnalyzed                EventName = "RiskAnalyzed"
	EventGoalPlanCreated             EventName = "GoalPlanCreated"
	EventBehaviorSelected            EventName = "BehaviorSelected"
	EventDecisionCreated             EventName = "DecisionCreated"
	EventBehaviorPlanned             EventName = "BehaviorPlanned"
	EventBehaviorExecuted            EventName = "BehaviorExecuted"
	EventBehaviorFailed              EventName = "BehaviorFailed"
	EventPathPlanned                 EventName = "PathPlanned"
	EventObstacleAvoidancePlanned    EventName = "ObstacleAvoidancePlanned"
	EventDroneCommandCreated         EventName = "DroneCommandCreated"
	EventDroneCommandDispatched      EventName = "DroneCommandDispatched"
	EventDroneCommandExecuted        EventName = "DroneCommandExecuted"
	EventFlightStabilized            EventName = "FlightStabilized"
	EventEnergyPlanUpdated           EventName = "EnergyPlanUpdated"
	EventHUDDataPrepared             EventName = "HUDDataPrepared"
	EventHUDRendered                 EventName = "HUDRendered"
	EventAlertPrioritized            EventName = "AlertPrioritized"
	EventAlertDisplayed              EventName = "AlertDisplayed"
	EventUserAttentionUpdated        EventName = "UserAttentionUpdated"
	EventUserFeedbackReceived        EventName = "UserFeedbackReceived"
	EventFailureLogged               EventName = "FailureLogged"
	EventMissionCompleted            EventName = "MissionCompleted"
	EventEnvironmentalChangeDetected EventName = "EnvironmentalChangeDetected"
	EventExperienceRecorded          EventName = "ExperienceRecorded"
	EventLearningUpdated             EventName = "LearningUpdated"
	EventAnomalyLearned              EventName = "AnomalyLearned"
	EventModelImprovementRequested   EventName = "ModelImprovementRequested"
	EventUserPreferenceLearned       EventName = "UserPreferenceLearned"
)

// DomainEvent is the transport-neutral envelope for HC-002 events.
type DomainEvent struct {
	ID            string
	Name          EventName
	Source        ModuleName
	MissionID     string
	CorrelationID string
	OccurredAt    time.Time
	Payload       map[string]string
}
