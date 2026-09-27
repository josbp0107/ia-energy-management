package db

import (
	"time"

	"gorm.io/datatypes"
)

const (
	RunStatusRunning   = "RUNNING"
	RunStatusCompleted = "COMPLETED"
	RunStatusFailed    = "FAILED"

	SeverityHigh   = "HIGH"
	SeverityMedium = "MEDIUM"
	SeverityLow    = "LOW"
	SeverityNone   = "NONE"

	TypeNormal             = "NORMAL"
	TypeRealAnomaly        = "REAL_ANOMALY"
	TypeExplainableAnomaly = "EXPLAINABLE_ANOMALY"
	TypeFalsePositive      = "FALSE_POSITIVE"
	TypeDataQuality        = "DATA_QUALITY"

	EventOperationalChange = "OPERATIONAL_CHANGE"
	EventScheduledOutage   = "SCHEDULED_OUTAGE"
	EventDataQuality       = "DATA_QUALITY"
	EventUnknown           = "UNKNOWN"

	AnomalyStatusOpen          = "OPEN"
	AnomalyStatusInvestigating = "INVESTIGATING"
	AnomalyStatusResolved      = "RESOLVED"
	AnomalyStatusDismissed     = "DISMISSED"

	MeterStatusNormal   = "NORMAL"   // sin anomalía (o falso positivo descartado)
	MeterStatusAlert    = "ALERT"    // anomalía MEDIUM/LOW
	MeterStatusCritical = "CRITICAL" // anomalía HIGH
	MeterStatusPending  = "PENDING"  // todavia no hay ningun analisis completado

	StepReadings       = "READINGS"
	StepBaseline       = "BASELINE"
	StepDetection      = "DETECTION"
	StepCorrelation    = "CORRELATION"
	StepEvents         = "EVENTS"
	StepExplanation    = "EXPLANATION"
	StepRecommendation = "RECOMMENDATION"
)

var AnalysisSteps = []string{
	StepReadings, StepBaseline, StepDetection, StepCorrelation, StepEvents, StepExplanation, StepRecommendation,
}

var AnomalyStatuses = []string{
	AnomalyStatusOpen, AnomalyStatusInvestigating, AnomalyStatusResolved, AnomalyStatusDismissed,
}

type Meter struct {
	MeterID   string    `gorm:"column:meter_id;primaryKey" json:"meter_id"`
	Name      string    `gorm:"column:name;not null" json:"name"`
	Location  string    `gorm:"column:location;not null" json:"location"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
}

type Reading struct {
	ID             uint      `gorm:"column:id;primaryKey" json:"id"`
	MeterID        string    `gorm:"column:meter_id;not null;uniqueIndex:idx_readings_meter_ts" json:"meter_id"`
	Timestamp      time.Time `gorm:"column:timestamp;type:timestamp;not null;uniqueIndex:idx_readings_meter_ts" json:"timestamp"`
	ConsumptionKWh float64   `gorm:"column:consumption_kwh;not null" json:"consumption_kwh"`
	VoltageV       float64   `gorm:"column:voltage_v;not null" json:"voltage_v"`
	CurrentA       float64   `gorm:"column:current_a;not null" json:"current_a"`
	PowerFactor    float64   `gorm:"column:power_factor;not null" json:"power_factor"`
	Status         string    `gorm:"column:status" json:"status"`
}

type Event struct {
	ID             uint      `gorm:"column:id;primaryKey" json:"id"`
	MeterID        string    `gorm:"column:meter_id;not null;index" json:"meter_id"`
	EventTimestamp time.Time `gorm:"column:event_timestamp;type:timestamp;not null" json:"event_timestamp"`
	EventType      string    `gorm:"column:event_type;not null" json:"event_type"`
	Description    string    `gorm:"column:description" json:"description"`
}

type AnalysisRun struct {
	ID          uint       `gorm:"column:id;primaryKey" json:"id"`
	Status      string     `gorm:"column:status;not null;default:RUNNING" json:"status"` // RUNNING | COMPLETED | FAILED
	CurrentStep string     `gorm:"column:current_step" json:"current_step"`
	Error       string     `gorm:"column:error" json:"error,omitempty"`
	StartedAt   time.Time  `gorm:"column:started_at;not null" json:"started_at"`
	FinishedAt  *time.Time `gorm:"column:finished_at" json:"finished_at"`
}

type Anomaly struct {
	ID                uint           `gorm:"column:id;primaryKey" json:"id"`
	AnalysisRunID     uint           `gorm:"column:analysis_run_id;not null;index" json:"analysis_run_id"`
	MeterID           string         `gorm:"column:meter_id;not null;index" json:"meter_id"`
	IsAnomaly         bool           `gorm:"column:is_anomaly;not null" json:"anomaly"`
	Type              string         `gorm:"column:type;not null" json:"type"`         // REAL_ANOMALY | EXPLAINABLE_ANOMALY | FALSE_POSITIVE | DATA_QUALITY
	Severity          string         `gorm:"column:severity;not null" json:"severity"` // HIGH | MEDIUM | LOW
	Confidence        float64        `gorm:"column:confidence;not null" json:"confidence"`
	Reason            string         `gorm:"column:reason" json:"reason"`
	RecommendedAction string         `gorm:"column:recommended_action" json:"recommended_action"`
	ExplanationSource string         `gorm:"column:explanation_source" json:"explanation_source"` // llm | template
	PriorityScore     float64        `gorm:"column:priority_score;not null;index" json:"priority_score"`
	VariationPct      float64        `gorm:"column:variation_pct" json:"variation_pct"`
	BaselineKWhDay    float64        `gorm:"column:baseline_kwh_day" json:"baseline_kwh_day"`
	CurrentKWhDay     float64        `gorm:"column:current_kwh_day" json:"current_kwh_day"`
	Evidence          datatypes.JSON `gorm:"column:evidence;type:jsonb" json:"evidence"`
	Status            string         `gorm:"column:status;not null;default:OPEN" json:"status"` // OPEN | INVESTIGATING | RESOLVED | DISMISSED
	CreatedAt         time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt         time.Time      `gorm:"column:updated_at" json:"updated_at"`
}
