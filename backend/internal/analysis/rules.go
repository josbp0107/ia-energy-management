package analysis

import (
	"slices"

	"github.com/josbp0107/ia-energy-management/internal/db"
)

var explainingEvents = []string{db.EventOperationalChange, db.EventScheduledOutage}

func ExplainsDeviation(eventType string) bool {
	return slices.Contains(explainingEvents, eventType)
}

type Rules struct {
	DeviationThresholdPct float64            `json:"deviation_threshold_pct"`
	VoltageMin            float64            `json:"voltage_min"`
	VoltageMax            float64            `json:"voltage_max"`
	PowerFactorDrop       float64            `json:"power_factor_drop"`
	CurrentRisePct        float64            `json:"current_rise_pct"`
	EventWindowHours      float64            `json:"event_window_hours"`
	MinInvalidReadings    int                `json:"min_invalid_readings"`
	BaselineDays          int                `json:"baseline_days"`
	SeverityWeight        map[string]float64 `json:"severity_weight"`
	ExplainingEvents      []string           `json:"explaining_events"`
}

func CurrentRules() Rules {
	return Rules{
		DeviationThresholdPct: DeviationThreshold * 100,
		VoltageMin:            VoltageMin,
		VoltageMax:            VoltageMax,
		PowerFactorDrop:       PowerFactorDrop,
		CurrentRisePct:        CurrentRise * 100,
		EventWindowHours:      EventWindow.Hours(),
		MinInvalidReadings:    MinInvalidReadings,
		BaselineDays:          BaselineDays,
		SeverityWeight:        severityWeight,
		ExplainingEvents:      explainingEvents,
	}
}
