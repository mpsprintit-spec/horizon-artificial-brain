package domain

import "time"

// ModuleName identifies one HC-002 AI module or supporting execution service.
type ModuleName string

const (
	ModuleAICluster    ModuleName = "ai_cluster"
	ModuleVision       ModuleName = "vision_ai"
	ModuleSensorFusion ModuleName = "sensor_fusion_ai"
	ModuleDecision     ModuleName = "decision_ai"
	ModuleLearning     ModuleName = "learning_engine"
	ModuleBehavior     ModuleName = "behavior"
	ModuleDrone        ModuleName = "drone_ai"
	ModuleHUD          ModuleName = "hud_ai"
)

// Frame is a single image frame from glasses, drone, or another camera.
type Frame struct {
	ID          string
	SourceID    string
	MissionID   string
	CapturedAt  time.Time
	ContentType string
	URI         string
	Metadata    map[string]string
}

// VideoStream is an ordered frame stream with source metadata.
type VideoStream struct {
	ID        string
	SourceID  string
	MissionID string
	StartedAt time.Time
	Metadata  map[string]string
}

// AudioSample is an audio input sample for multimodal context.
type AudioSample struct {
	ID          string
	SourceID    string
	MissionID   string
	CapturedAt  time.Time
	ContentType string
	URI         string
	Metadata    map[string]string
}

// Telemetry contains device telemetry used by Sensor Fusion AI and Drone AI.
type Telemetry struct {
	ID           string
	DeviceID     string
	MissionID    string
	CapturedAt   time.Time
	Speed        float64
	Altitude     float64
	BatteryLevel float64
	Connectivity float64
	Health       string
	Measurements map[string]float64
}

// SensorData is a raw sensor reading from IMU, GPS, environment, or wearable sensors.
type SensorData struct {
	ID           string
	SensorID     string
	DeviceID     string
	MissionID    string
	SensorType   string
	CapturedAt   time.Time
	Measurements map[string]float64
	Metadata     map[string]string
}

// LocationSignal is a GPS or derived location sample.
type LocationSignal struct {
	ID         string
	SourceID   string
	MissionID  string
	CapturedAt time.Time
	Latitude   float64
	Longitude  float64
	Altitude   float64
	Accuracy   float64
}

// UserContext carries profile, preference, command, attention, and interaction state.
type UserContext struct {
	UserID          string
	MissionID       string
	ProfileID       string
	Preferences     map[string]string
	Command         string
	AttentionState  string
	InteractionMode string
	UpdatedAt       time.Time
}

// MissionContext contains mission objective, constraints, route, priority, and lifecycle state.
type MissionContext struct {
	MissionID   string
	Objective   string
	Constraints map[string]string
	RouteID     string
	Priority    int
	State       string
	UpdatedAt   time.Time
}

// SystemStatus represents service, device, connectivity, battery, and runtime readiness state.
type SystemStatus struct {
	ID            string
	Scope         string
	Ready         bool
	Connectivity  float64
	BatteryLevel  float64
	RuntimeStatus string
	UpdatedAt     time.Time
}

// Detection is an object, person, target, or obstacle detected from visual input.
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

// SceneContext is semantic understanding of an environment and scene.
type SceneContext struct {
	ID          string
	FrameID     string
	MissionID   string
	Description string
	Labels      []string
	Confidence  float64
}

// DepthMap summarizes estimated distance or depth information from visual input.
type DepthMap struct {
	ID        string
	FrameID   string
	URI       string
	MinDepth  float64
	MaxDepth  float64
	MeanDepth float64
}

// OpticalFlow contains motion vector information across frames.
type OpticalFlow struct {
	ID           string
	FrameID      string
	PreviousID   string
	VectorURI    string
	MeanVelocity float64
}

// Obstacle is a physical or inferred object relevant to navigation or safety.
type Obstacle struct {
	ID         string
	SourceID   string
	MissionID  string
	Label      string
	Distance   float64
	Confidence float64
	DetectedAt time.Time
}

// FusedSensorState is normalized multimodal sensor state after filtering and fusion.
type FusedSensorState struct {
	ID           string
	MissionID    string
	DeviceID     string
	CapturedAt   time.Time
	Confidence   float64
	Measurements map[string]float64
}

// NavigationState contains position, velocity, orientation, heading, confidence, and route progress.
type NavigationState struct {
	ID            string
	MissionID     string
	DeviceID      string
	Position      LocationSignal
	Velocity      float64
	Orientation   map[string]float64
	Heading       float64
	RouteProgress float64
	Confidence    float64
	EstimatedAt   time.Time
}

// SituationAssessment is Decision AI's interpreted mission and environment state.
type SituationAssessment struct {
	ID         string
	MissionID  string
	Summary    string
	Confidence float64
	AssessedAt time.Time
}

// RiskAssessment contains safety and operational risk classification.
type RiskAssessment struct {
	ID          string
	MissionID   string
	Level       string
	Score       float64
	Explanation string
	AssessedAt  time.Time
}

// GoalPlan decomposes mission goals into selected objectives.
type GoalPlan struct {
	ID        string
	MissionID string
	Goals     []string
	Priority  int
	CreatedAt time.Time
}

// BehaviorSelection is the selected behavior with rationale and confidence.
type BehaviorSelection struct {
	ID         string
	MissionID  string
	Behavior   string
	Rationale  string
	Confidence float64
	CreatedAt  time.Time
}

// Decision is the final AI decision for behavior, drone, HUD, or notifications.
type Decision struct {
	ID            string
	MissionID     string
	Command       string
	Priority      int
	Confidence    float64
	Explanation   string
	RecommendedTo []string
	CreatedAt     time.Time
}

// BehaviorPlan is an executable plan derived from a decision.
type BehaviorPlan struct {
	ID         string
	DecisionID string
	MissionID  string
	Actions    []string
	CreatedAt  time.Time
}

// BehaviorExecution is runtime state and outcome of a behavior plan.
type BehaviorExecution struct {
	ID        string
	PlanID    string
	Status    string
	Outcome   string
	StartedAt time.Time
	EndedAt   time.Time
}

// DroneCommand is a command sent to drone control systems.
type DroneCommand struct {
	ID        string
	DroneID   string
	MissionID string
	Command   string
	Payload   map[string]string
	CreatedAt time.Time
}

// FlightPlan captures path, constraints, speed, altitude, and safety metadata.
type FlightPlan struct {
	ID          string
	MissionID   string
	DroneID     string
	Waypoints   []LocationSignal
	Constraints map[string]string
	CreatedAt   time.Time
}

// EnergyPlan is a battery and energy optimization strategy.
type EnergyPlan struct {
	ID             string
	DroneID        string
	MissionID      string
	BatteryLevel   float64
	Strategy       string
	ReturnRequired bool
	UpdatedAt      time.Time
}

// HUDData is an information payload for smart-glasses display.
type HUDData struct {
	ID        string
	UserID    string
	MissionID string
	Items     []string
	Priority  int
	CreatedAt time.Time
}

// AROverlay is a spatial augmented-reality placement payload.
type AROverlay struct {
	ID        string
	HUDDataID string
	TargetID  string
	Position  map[string]float64
	Label     string
}

// Alert is a user-facing warning, notification, or operational alert.
type Alert struct {
	ID        string
	MissionID string
	Severity  string
	Message   string
	CreatedAt time.Time
}

// LearningRecord is a persistent record of experience, outcome, feedback, or improvement signal.
type LearningRecord struct {
	ID        string
	MissionID string
	Source    string
	Outcome   string
	Feedback  string
	CreatedAt time.Time
}

// ExperienceMemory is aggregated learning state from historical experience.
type ExperienceMemory struct {
	ID          string
	SubjectID   string
	Summary     string
	RecordIDs   []string
	LastUpdated time.Time
}

// ModelVersion captures AI model version, runtime target, status, and deployment metadata.
type ModelVersion struct {
	ID            string
	Module        ModuleName
	Version       string
	RuntimeTarget string
	Status        string
	CreatedAt     time.Time
}

// AIWorkload is a unit of AI processing work and resource requirement.
type AIWorkload struct {
	ID        string
	Module    ModuleName
	MissionID string
	Priority  int
	Payload   map[string]string
	CreatedAt time.Time
}

// AIResult is a generic result envelope for module outputs.
type AIResult struct {
	ID         string
	WorkloadID string
	Module     ModuleName
	Status     string
	Confidence float64
	Payload    map[string]string
	CreatedAt  time.Time
}
