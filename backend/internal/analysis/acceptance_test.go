package analysis

import (
	"math"
	"path/filepath"
	"testing"

	"github.com/josbp0107/ia-energy-management/internal/csvdata"
	"github.com/josbp0107/ia-energy-management/internal/db"
)

func TestAcceptance(t *testing.T) {
	dataDir := filepath.Join("..", "..", "..", "data")

	readings, err := csvdata.LoadReadings(filepath.Join(dataDir, "readings.csv"))
	if err != nil {
		t.Fatalf("leer readings.csv: %v", err)
	}
	events, err := csvdata.LoadEvents(filepath.Join(dataDir, "events.csv"))
	if err != nil {
		t.Fatalf("leer events.csv: %v", err)
	}

	results := Analyze(readings, events)
	if len(results) != 12 {
		t.Fatalf("se esperaban 12 medidores, hay %d", len(results))
	}

	byMeter := map[string]Result{}
	for _, r := range results {
		byMeter[r.MeterID] = r
	}

	expected := []struct {
		meterID    string
		typ        string
		severity   string
		anomaly    bool
		confidence float64
		variation  float64
		priority   float64
	}{
		{"M-109", db.TypeRealAnomaly, db.SeverityHigh, true, 0.98, 109.8, 6.17},
		{"M-112", db.TypeDataQuality, db.SeverityHigh, true, 0.95, 0.0, 2.85},
		{"M-104", db.TypeExplainableAnomaly, db.SeverityMedium, true, 0.90, 47.3, 2.65},
		{"M-106", db.TypeFalsePositive, db.SeverityLow, false, 0.95, 1.0, 0.96},
	}

	for _, want := range expected {
		t.Run(want.meterID, func(t *testing.T) {
			got, ok := byMeter[want.meterID]
			if !ok {
				t.Fatalf("%s no aparece en los resultados", want.meterID)
			}
			if got.Type != want.typ || got.Severity != want.severity || got.Anomaly != want.anomaly {
				t.Errorf("clasificación = %s/%s/anomaly=%v, se esperaba %s/%s/anomaly=%v",
					got.Type, got.Severity, got.Anomaly, want.typ, want.severity, want.anomaly)
			}
			if !almostEqual(got.Confidence, want.confidence) {
				t.Errorf("confianza = %.2f, se esperaba %.2f", got.Confidence, want.confidence)
			}
			if !almostEqual(math.Abs(got.VariationPct), math.Abs(want.variation)) {
				t.Errorf("variación = %.1f%%, se esperaba %.1f%%", got.VariationPct, want.variation)
			}
			if !almostEqual(got.PriorityScore, want.priority) {
				t.Errorf("prioridad = %.2f, se esperaba %.2f", got.PriorityScore, want.priority)
			}
		})
	}

	t.Run("M-109 es la prioridad 1", func(t *testing.T) {
		if results[0].MeterID != "M-109" {
			t.Errorf("el primero es %s, se esperaba M-109", results[0].MeterID)
		}
	})

	t.Run("los otros 8 son NORMAL", func(t *testing.T) {
		normals := 0
		for _, r := range results {
			if r.Type != db.TypeNormal {
				continue
			}
			normals++
			if r.PriorityScore != 0 || math.Abs(r.VariationPct) > 1.2 {
				t.Errorf("%s: NORMAL con prioridad %.2f y variación %.1f%%", r.MeterID, r.PriorityScore, r.VariationPct)
			}
		}
		if normals != 8 {
			t.Errorf("hay %d medidores NORMAL, se esperaban 8", normals)
		}
	})

	t.Run("resumen: 4 anomalías, 2 de severidad alta", func(t *testing.T) {
		anomalies, high := 0, 0
		for _, r := range results {
			if r.Type == db.TypeNormal {
				continue
			}
			anomalies++
			if r.Severity == db.SeverityHigh {
				high++
			}
		}
		if anomalies != 4 || high != 2 {
			t.Errorf("anomalías = %d, alta = %d; se esperaban 4 y 2", anomalies, high)
		}
	})
}

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}
