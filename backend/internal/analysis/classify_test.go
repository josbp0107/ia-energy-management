package analysis

import (
	"math"
	"testing"
	"time"

	"github.com/josbp0107/ia-energy-management/internal/db"
)

var dataStart = time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

func at(day, hour int) time.Time {
	return dataStart.AddDate(0, 0, day-1).Add(time.Duration(hour) * time.Hour)
}

func setLoad(r *db.Reading, kwh, pf float64) {
	r.ConsumptionKWh = kwh
	r.PowerFactor = pf
	r.CurrentA = kwh * 1000 / (1.07 * r.VoltageV * pf)
}

func syntheticMeter(modify func(r *db.Reading)) []db.Reading {
	var readings []db.Reading
	for ts := dataStart; ts.Before(at(15, 0)); ts = ts.Add(time.Hour) {
		r := db.Reading{MeterID: "M-TEST", Timestamp: ts, VoltageV: 220}
		setLoad(&r, 50, 0.95)
		if modify != nil {
			modify(&r)
		}
		readings = append(readings, r)
	}
	return readings
}

func event(eventType string, ts time.Time) db.Event {
	return db.Event{MeterID: "M-TEST", EventType: eventType, EventTimestamp: ts, Description: "test"}
}

func between(r *db.Reading, from, to time.Time) bool {
	return !r.Timestamp.Before(from) && r.Timestamp.Before(to)
}

func TestClassification(t *testing.T) {
	realSpike := func(r *db.Reading) {
		if between(r, at(13, 14), at(15, 0)) {
			setLoad(r, 110, 0.74)
		}
	}

	tests := []struct {
		name           string
		modify         func(r *db.Reading)
		events         []db.Event
		wantType       string
		wantSeverity   string
		wantAnomaly    bool
		wantConfidence float64
	}{
		{
			name:         "consumo estable es NORMAL",
			wantType:     db.TypeNormal,
			wantSeverity: db.SeverityNone,
		},
		{
			name:           "subida sin evento es REAL_ANOMALY",
			modify:         realSpike,
			wantType:       db.TypeRealAnomaly,
			wantSeverity:   db.SeverityHigh,
			wantAnomaly:    true,
			wantConfidence: 0.98,
		},
		{
			name:           "un evento UNKNOWN no explica la subida (caso M-109)",
			modify:         realSpike,
			events:         []db.Event{event(db.EventUnknown, at(13, 14))},
			wantType:       db.TypeRealAnomaly,
			wantSeverity:   db.SeverityHigh,
			wantAnomaly:    true,
			wantConfidence: 0.98,
		},
		{
			name:           "un OPERATIONAL_CHANGE fuera de ±3 h no explica la subida",
			modify:         realSpike,
			events:         []db.Event{event(db.EventOperationalChange, at(13, 10))},
			wantType:       db.TypeRealAnomaly,
			wantSeverity:   db.SeverityHigh,
			wantAnomaly:    true,
			wantConfidence: 0.98,
		},
		{
			name: "escalón con OPERATIONAL_CHANGE es EXPLAINABLE_ANOMALY (caso M-104)",
			modify: func(r *db.Reading) {
				if between(r, at(11, 0), at(15, 0)) {
					setLoad(r, 75, 0.95)
				}
			},
			events:         []db.Event{event(db.EventOperationalChange, at(11, 0))},
			wantType:       db.TypeExplainableAnomaly,
			wantSeverity:   db.SeverityMedium,
			wantAnomaly:    true,
			wantConfidence: 0.90,
		},
		{
			name: "caída con SCHEDULED_OUTAGE es FALSE_POSITIVE (caso M-106)",
			modify: func(r *db.Reading) {
				if between(r, at(8, 0), at(8, 12)) {
					setLoad(r, 10, 0.95)
				}
			},
			events:         []db.Event{event(db.EventScheduledOutage, at(8, 0))},
			wantType:       db.TypeFalsePositive,
			wantSeverity:   db.SeverityLow,
			wantAnomaly:    false,
			wantConfidence: 0.95, // 0.85 + duración a 24 h
		},
		{
			name: "voltaje fuera de rango en 5 lecturas es DATA_QUALITY (caso M-112)",
			modify: func(r *db.Reading) {
				if r.Timestamp.Day() == 13 && r.Timestamp.Hour()%3 == 0 && r.Timestamp.Hour() < 15 {
					r.VoltageV = 200
				}
			},
			wantType:       db.TypeDataQuality,
			wantSeverity:   db.SeverityHigh,
			wantAnomaly:    true,
			wantConfidence: 0.85, // 0.75 + consumo estable (sin evento que lo corrobore)
		},
		{
			name: "evento DATA_QUALITY sube la confianza",
			modify: func(r *db.Reading) {
				if r.Timestamp.Day() == 13 && r.Timestamp.Hour()%3 == 0 && r.Timestamp.Hour() < 15 {
					r.VoltageV = 200
				}
			},
			events:         []db.Event{event(db.EventDataQuality, at(13, 0))},
			wantType:       db.TypeDataQuality,
			wantSeverity:   db.SeverityHigh,
			wantAnomaly:    true,
			wantConfidence: 0.95,
		},
		{
			name: "2 lecturas inválidas no alcanzan el umbral: NORMAL",
			modify: func(r *db.Reading) {
				if r.Timestamp.Equal(at(13, 0)) || r.Timestamp.Equal(at(13, 3)) {
					r.PowerFactor = 1.2
				}
			},
			wantType:     db.TypeNormal,
			wantSeverity: db.SeverityNone,
		},
		{
			name: "calidad de datos tiene prioridad sobre una desviación",
			modify: func(r *db.Reading) {
				realSpike(r)
				if r.Timestamp.Day() == 14 && r.Timestamp.Hour() < 3 {
					r.VoltageV = 245
				}
			},
			wantType:       db.TypeDataQuality,
			wantSeverity:   db.SeverityHigh,
			wantAnomaly:    true,
			wantConfidence: 0.75, // consumo no estable y sin evento
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			results := Analyze(syntheticMeter(tc.modify), tc.events)
			if len(results) != 1 {
				t.Fatalf("se esperaba 1 resultado, hay %d", len(results))
			}
			got := results[0]

			if got.Type != tc.wantType || got.Severity != tc.wantSeverity || got.Anomaly != tc.wantAnomaly {
				t.Errorf("clasificación = %s/%s/anomaly=%v, se esperaba %s/%s/anomaly=%v",
					got.Type, got.Severity, got.Anomaly, tc.wantType, tc.wantSeverity, tc.wantAnomaly)
			}
			if !almostEqual(got.Confidence, tc.wantConfidence) {
				t.Errorf("confianza = %.2f, se esperaba %.2f", got.Confidence, tc.wantConfidence)
			}
		})
	}
}

func TestPriorityScore(t *testing.T) {
	tests := []struct {
		severity   string
		confidence float64
		variation  float64
		want       float64
	}{
		{db.SeverityHigh, 0.98, 109.8, 6.17}, // M-109
		{db.SeverityHigh, 0.95, 0.0, 2.85},   // M-112
		{db.SeverityMedium, 0.90, 47.3, 2.65},
		{db.SeverityLow, 0.95, 1.0, 0.96},
		{db.SeverityNone, 0, 1.2, 0},
	}
	for _, tc := range tests {
		if got := priorityScore(tc.severity, tc.confidence, tc.variation); !almostEqual(got, tc.want) {
			t.Errorf("priorityScore(%s, %.2f, %.1f) = %.2f, se esperaba %.2f",
				tc.severity, tc.confidence, tc.variation, got, tc.want)
		}
	}
}

func TestRoundLikePython(t *testing.T) {
	tests := []struct {
		x        float64
		decimals int
		want     float64
	}{
		{6.1678, 2, 6.17},
		{2.675, 2, 2.67},
		{0.125, 2, 0.12},
		{109.84, 1, 109.8},
		{-0.04, 1, 0}, // sin "-0"
	}
	for _, tc := range tests {
		got := round(tc.x, tc.decimals)
		if got != tc.want || math.Signbit(got) != math.Signbit(tc.want) {
			t.Errorf("round(%v, %d) = %v, se esperaba %v", tc.x, tc.decimals, got, tc.want)
		}
	}
}

func TestMedian(t *testing.T) {
	tests := []struct {
		name   string
		values []float64
		want   float64
	}{
		{"impar", []float64{3, 1, 2}, 2},
		{"par promedia los centrales", []float64{4, 1, 3, 2}, 2.5},
		{"vacío", nil, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := median(tc.values); got != tc.want {
				t.Errorf("median(%v) = %v, se esperaba %v", tc.values, got, tc.want)
			}
		})
	}
}

func TestHourlyBaseline(t *testing.T) {
	hours := HourlyBaseline(syntheticMeter(func(r *db.Reading) {
		if between(r, at(13, 0), at(15, 0)) {
			setLoad(r, 500, 0.95)
		}
	}))

	if len(hours) != 24 {
		t.Fatalf("se esperaban 24 horas, hay %d", len(hours))
	}
	for i, h := range hours {
		if h.Hour != i {
			t.Fatalf("horas desordenadas: posición %d tiene la hora %d", i, h.Hour)
		}
		if h.KWh != 50 || h.LowerKWh != 35 || h.UpperKWh != 65 || h.PowerFactor != 0.95 {
			t.Errorf("hora %d: %+v, se esperaba kwh 50 con banda 35–65 y FP 0.95", h.Hour, h)
		}
	}
}

func TestCurrentRulesMirrorEngineConstants(t *testing.T) {
	r := CurrentRules()
	if r.DeviationThresholdPct != 30 || r.VoltageMin != 209 || r.VoltageMax != 231 || r.EventWindowHours != 3 {
		t.Errorf("reglas inesperadas: %+v", r)
	}
	if !ExplainsDeviation(db.EventScheduledOutage) || ExplainsDeviation(db.EventUnknown) {
		t.Errorf("solo OPERATIONAL_CHANGE y SCHEDULED_OUTAGE deben explicar una desviación")
	}
}
