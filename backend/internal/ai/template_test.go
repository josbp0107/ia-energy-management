package ai

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/josbp0107/ia-energy-management/internal/analysis"
	"github.com/josbp0107/ia-energy-management/internal/csvdata"
)

func TestTemplateExplanation(t *testing.T) {
	dataDir := filepath.Join("..", "..", "..", "data")
	readings, err := csvdata.LoadReadings(filepath.Join(dataDir, "readings.csv"))
	if err != nil {
		t.Fatalf("leer readings.csv: %v", err)
	}
	events, err := csvdata.LoadEvents(filepath.Join(dataDir, "events.csv"))
	if err != nil {
		t.Fatalf("leer events.csv: %v", err)
	}

	byMeter := map[string]analysis.Result{}
	for _, r := range analysis.Analyze(readings, events) {
		byMeter[r.MeterID] = r
	}

	tests := []struct {
		meterID      string
		reasonHas    []string
		actionHas    string
		reasonHasNot string
	}{
		{"M-109", []string{"+109,8%", "12/09 14:00", "58 h", "0,94 a 0,74", "sin un evento operativo"}, "Inspeccionar", ""},
		{"M-104", []string{"+47,3%", "New production line activated"}, "actualizar el baseline", ""},
		{"M-106", []string{"Caída", "12 h", "Scheduled maintenance outage"}, "No requiere acción", ""},
		{"M-112", []string{"16 lecturas inválidas", "0,58 / 0,72 / 0,98", "estable"}, "Revisar el medidor", "-0,0%"},
	}

	for _, tc := range tests {
		t.Run(tc.meterID, func(t *testing.T) {
			got := TemplateExplanation(byMeter[tc.meterID])
			for _, want := range tc.reasonHas {
				if !strings.Contains(got.Reason, want) {
					t.Errorf("reason no contiene %q:\n%s", want, got.Reason)
				}
			}
			if tc.reasonHasNot != "" && strings.Contains(got.Reason, tc.reasonHasNot) {
				t.Errorf("reason no debería contener %q:\n%s", tc.reasonHasNot, got.Reason)
			}
			if !strings.Contains(got.RecommendedAction, tc.actionHas) {
				t.Errorf("recommended_action no contiene %q:\n%s", tc.actionHas, got.RecommendedAction)
			}
		})
	}
}
