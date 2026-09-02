package domain

import "context"

// AICluster starts, stops, and reports module status.
type AICluster interface {
	Start(context.Context) error
	Stop(context.Context) error
	Status(context.Context) (SystemStatus, error)
}

// AIModule is the standard lifecycle and health contract for AI modules.
type AIModule interface {
	Name() ModuleName
	Start(context.Context) error
	Stop(context.Context) error
	Health(context.Context) (SystemStatus, error)
}

// InferenceRequestHandler accepts inference requests and returns typed AI results.
type InferenceRequestHandler interface {
	HandleInference(context.Context, AIWorkload) (AIResult, error)
}

// EventPublisher publishes domain events.
type EventPublisher interface {
	Publish(context.Context, DomainEvent) error
}

// EventSubscriber subscribes to domain events.
type EventSubscriber interface {
	Subscribe(context.Context, []EventName, EventHandler) error
}

// EventHandler handles a domain event delivered by an event subscriber.
type EventHandler interface {
	HandleEvent(context.Context, DomainEvent) error
}

// ModelRegistry locates approved AI model versions and runtime metadata.
type ModelRegistry interface {
	GetModelVersion(context.Context, ModuleName, string) (ModelVersion, error)
}

// ContextProvider provides mission, user, environment, and system context.
type ContextProvider interface {
	MissionContext(context.Context, string) (MissionContext, error)
	UserContext(context.Context, string) (UserContext, error)
	SystemStatus(context.Context, string) (SystemStatus, error)
}

// FrameAnalyzer analyzes one frame and returns a vision result.
type FrameAnalyzer interface {
	AnalyzeFrame(context.Context, Frame) (AIResult, error)
}

// StreamAnalyzer analyzes continuous video streams.
type StreamAnalyzer interface {
	AnalyzeStream(context.Context, VideoStream) (AIResult, error)
}

// ObjectDetector returns detections from a frame.
type ObjectDetector interface {
	DetectObjects(context.Context, Frame) ([]Detection, error)
}

// SceneUnderstander returns scene context from a frame.
type SceneUnderstander interface {
	UnderstandScene(context.Context, Frame) (SceneContext, error)
}

// DepthEstimator returns depth information from a frame.
type DepthEstimator interface {
	EstimateDepth(context.Context, Frame) (DepthMap, error)
}

// OpticalFlowTracker tracks motion between frames.
type OpticalFlowTracker interface {
	TrackOpticalFlow(context.Context, Frame, Frame) (OpticalFlow, error)
}

// LowLightEnhancer improves low-light frame quality before inference.
type LowLightEnhancer interface {
	EnhanceLowLight(context.Context, Frame) (Frame, error)
}

// SensorFusionEngine merges sensor samples into a fused state.
type SensorFusionEngine interface {
	FuseSensors(context.Context, []SensorData, []Telemetry, []LocationSignal) (FusedSensorState, error)
}

// IMUFuser processes accelerometer, gyroscope, and magnetometer signals.
type IMUFuser interface {
	FuseIMU(context.Context, []SensorData) (FusedSensorState, error)
}

// GPSFuser processes GPS and location data.
type GPSFuser interface {
	FuseGPS(context.Context, []LocationSignal) (LocationSignal, error)
}

// VIOEstimator combines visual and inertial odometry.
type VIOEstimator interface {
	EstimateVIO(context.Context, []Detection, []SensorData) (NavigationState, error)
}

// NoiseFilter filters unreliable sensor measurements.
type NoiseFilter interface {
	FilterNoise(context.Context, []SensorData) ([]SensorData, error)
}

// StateEstimator creates navigation state and confidence metrics.
type StateEstimator interface {
	EstimateState(context.Context, FusedSensorState) (NavigationState, error)
}

// SituationAssessor interprets current mission and environment state.
type SituationAssessor interface {
	AssessSituation(context.Context, MissionContext, NavigationState, []Detection) (SituationAssessment, error)
}

// RiskAnalyzer determines safety and operational risk.
type RiskAnalyzer interface {
	AnalyzeRisk(context.Context, SituationAssessment) (RiskAssessment, error)
}

// GoalPlanner creates or updates mission goals.
type GoalPlanner interface {
	PlanGoals(context.Context, MissionContext, RiskAssessment) (GoalPlan, error)
}

// BehaviorSelector chooses a behavior from assessed state and goals.
type BehaviorSelector interface {
	SelectBehavior(context.Context, SituationAssessment, GoalPlan) (BehaviorSelection, error)
}

// DecisionOptimizer performs multi-objective optimization.
type DecisionOptimizer interface {
	OptimizeDecision(context.Context, []BehaviorSelection, RiskAssessment) (BehaviorSelection, error)
}

// ContextAwareDecisionEngine creates final Decision objects.
type ContextAwareDecisionEngine interface {
	CreateDecision(context.Context, MissionContext, UserContext, BehaviorSelection) (Decision, error)
}

// ExperienceRecorder stores mission and action outcomes.
type ExperienceRecorder interface {
	RecordExperience(context.Context, LearningRecord) error
}

// FeedbackProcessor processes user and system feedback.
type FeedbackProcessor interface {
	ProcessFeedback(context.Context, LearningRecord) error
}

// ModelTrainingCoordinator schedules or requests model improvement.
type ModelTrainingCoordinator interface {
	RequestModelImprovement(context.Context, ModelVersion, []LearningRecord) error
}

// UserPreferenceLearner updates learned user preferences.
type UserPreferenceLearner interface {
	LearnUserPreference(context.Context, UserContext, LearningRecord) error
}

// AnomalyLearner converts anomalies into learning records.
type AnomalyLearner interface {
	LearnAnomaly(context.Context, DomainEvent) (LearningRecord, error)
}

// PerformanceImprover evaluates model and behavior performance.
type PerformanceImprover interface {
	EvaluatePerformance(context.Context, []LearningRecord) (AIResult, error)
}

// BehaviorExecutor executes a selected behavior.
type BehaviorExecutor interface {
	ExecuteBehavior(context.Context, BehaviorPlan) (BehaviorExecution, error)
}

// BehaviorPlanner converts decisions into behavior plans.
type BehaviorPlanner interface {
	PlanBehavior(context.Context, Decision) (BehaviorPlan, error)
}

// ActionDispatcher dispatches actions to device-specific services.
type ActionDispatcher interface {
	DispatchAction(context.Context, BehaviorExecution) error
}

// BehaviorPolicy validates behavior safety and policy compliance.
type BehaviorPolicy interface {
	ValidateBehavior(context.Context, BehaviorPlan) error
}

// PathPlanner creates safe flight paths.
type PathPlanner interface {
	PlanPath(context.Context, BehaviorPlan, NavigationState) (FlightPlan, error)
}

// ObstacleAvoider adjusts paths around obstacles.
type ObstacleAvoider interface {
	AvoidObstacles(context.Context, FlightPlan, []Obstacle) (FlightPlan, error)
}

// AutoTracker tracks user, object, or mission target.
type AutoTracker interface {
	TrackTarget(context.Context, string, NavigationState) (AIResult, error)
}

// FlightStabilizer stabilizes flight commands.
type FlightStabilizer interface {
	StabilizeFlight(context.Context, DroneCommand, NavigationState) (DroneCommand, error)
}

// EnergyManager optimizes energy usage and return-to-base behavior.
type EnergyManager interface {
	PlanEnergy(context.Context, Telemetry, FlightPlan) (EnergyPlan, error)
}

// DroneCommandDispatcher dispatches commands to drones.
type DroneCommandDispatcher interface {
	DispatchDroneCommand(context.Context, DroneCommand) error
}

// InformationPrioritizer ranks information for display.
type InformationPrioritizer interface {
	PrioritizeInformation(context.Context, []HUDData) (HUDData, error)
}

// ARPlacementEngine places augmented content in the user's field of view.
type ARPlacementEngine interface {
	PlaceAROverlay(context.Context, HUDData) (AROverlay, error)
}

// TargetHighlighter marks relevant targets.
type TargetHighlighter interface {
	HighlightTarget(context.Context, Detection) (AROverlay, error)
}

// AlertNotifier prepares attention-worthy alerts.
type AlertNotifier interface {
	PrepareAlert(context.Context, Alert) (HUDData, error)
}

// AttentionManager tracks user focus and avoids overload.
type AttentionManager interface {
	UpdateAttention(context.Context, UserContext) (UserContext, error)
}

// HUDRenderer renders final HUD data.
type HUDRenderer interface {
	RenderHUD(context.Context, HUDData) error
}
