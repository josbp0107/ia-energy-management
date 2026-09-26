package analysis

import (
	"sort"
	"time"

	"github.com/josbp0107/ia-energy-management/internal/db"
)

const (
	BaselineDays       = 7
	DeviationThreshold = 0.30
	VoltageMin         = 209.0
	VoltageMax         = 231.0
	RatioTolerance     = 0.50
	MinInvalidReadings = 3
	EventWindow        = 3 * time.Hour
	CurrentRise        = 0.30
	PowerFactorDrop    = 0.10
	LongDurationHours  = 24
	StableVariationPct = 5.0
	LargeVariationPct  = 50.0
	MaxConfidence      = 0.99
)

var severityWeight = map[string]float64{
	db.SeverityHigh:   3,
	db.SeverityMedium: 2,
	db.SeverityLow:    1,
	db.SeverityNone:   0,
}

type Result struct {
	MeterID        string   `json:"meter_id"`
	Anomaly        bool     `json:"anomaly"`
	Type           string   `json:"type"`
	Severity       string   `json:"severity"`
	Confidence     float64  `json:"confidence"`
	VariationPct   float64  `json:"variation_pct"`
	BaselineKWhDay float64  `json:"baseline_kwh_day"`
	CurrentKWhDay  float64  `json:"current_kwh_day"`
	PriorityScore  float64  `json:"priority_score"`
	Evidence       Evidence `json:"evidence"`
}

type Evidence struct {
	DataQuality *DataQualityEvidence `json:"data_quality,omitempty"`
	Deviation   *DeviationEvidence   `json:"deviation,omitempty"`
}

type DataQualityEvidence struct {
	InvalidReadings     int        `json:"invalid_readings"`
	FirstInvalid        time.Time  `json:"first_invalid"`
	VoltageRange        [2]float64 `json:"voltage_range"`
	PowerFactorValues   []float64  `json:"power_factor_values"`
	ConsumptionStable   bool       `json:"consumption_stable"`
	CorroboratingEvents []string   `json:"corroborating_events"`
}

type DeviationEvidence struct {
	Start                  time.Time      `json:"start"`
	End                    time.Time      `json:"end"`
	HoursAffected          int            `json:"hours_affected"`
	Direction              string         `json:"direction"` // UP | DOWN
	MeanHourlyDeviationPct float64        `json:"mean_hourly_deviation_pct"`
	CurrentChangePct       float64        `json:"current_change_pct"`
	PowerFactorChange      float64        `json:"power_factor_change"`
	BaselineCurrentA       float64        `json:"baseline_current_a"`
	ObservedCurrentA       float64        `json:"observed_current_a"`
	BaselinePowerFactor    float64        `json:"baseline_power_factor"`
	ObservedPowerFactor    float64        `json:"observed_power_factor"`
	RelatedEvents          []RelatedEvent `json:"related_events"`
}

type RelatedEvent struct {
	EventType      string    `json:"event_type"`
	Description    string    `json:"description"`
	EventTimestamp time.Time `json:"event_timestamp"`
}

func Analyze(readings []db.Reading, events []db.Event) []Result {
	if len(readings) == 0 {
		return []Result{}
	}

	readingsByMeter := map[string][]db.Reading{}
	for _, r := range readings {
		readingsByMeter[r.MeterID] = append(readingsByMeter[r.MeterID], r)
	}
	eventsByMeter := map[string][]db.Event{}
	for _, e := range events {
		eventsByMeter[e.MeterID] = append(eventsByMeter[e.MeterID], e)
	}

	baselineEnd := BaselineEnd(readings)

	meterIDs := make([]string, 0, len(readingsByMeter))
	for id := range readingsByMeter {
		meterIDs = append(meterIDs, id)
	}
	sort.Strings(meterIDs)

	results := make([]Result, 0, len(meterIDs))
	for _, id := range meterIDs {
		results = append(results, analyzeMeter(id, readingsByMeter[id], eventsByMeter[id], baselineEnd))
	}

	sort.SliceStable(results, func(i, j int) bool {
		return results[i].PriorityScore > results[j].PriorityScore
	})
	return results
}
