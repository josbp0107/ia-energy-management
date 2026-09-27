package ai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/josbp0107/ia-energy-management/internal/analysis"
	"github.com/josbp0107/ia-energy-management/internal/config"
	"github.com/josbp0107/ia-energy-management/internal/csvdata"
)

func resultFor(t *testing.T, meterID string) analysis.Result {
	t.Helper()
	dataDir := filepath.Join("..", "..", "..", "data")
	readings, err := csvdata.LoadReadings(filepath.Join(dataDir, "readings.csv"))
	if err != nil {
		t.Fatalf("leer readings.csv: %v", err)
	}
	events, err := csvdata.LoadEvents(filepath.Join(dataDir, "events.csv"))
	if err != nil {
		t.Fatalf("leer events.csv: %v", err)
	}
	for _, r := range analysis.Analyze(readings, events) {
		if r.MeterID == meterID {
			return r
		}
	}
	t.Fatalf("%s no está en los resultados", meterID)
	return analysis.Result{}
}

func apiMessage(text, stopReason string) string {
	body, _ := json.Marshal(map[string]any{
		"id":          "msg_test",
		"type":        "message",
		"role":        "assistant",
		"model":       "claude-opus-5",
		"content":     []map[string]any{{"type": "text", "text": text}},
		"stop_reason": stopReason,
		"usage":       map[string]any{"input_tokens": 10, "output_tokens": 10},
	})
	return string(body)
}

func llmJSON(reason, action string) string {
	body, _ := json.Marshal(map[string]string{"reason": reason, "recommended_action": action})
	return string(body)
}

func TestLLMExplainer(t *testing.T) {
	m109 := resultFor(t, "M-109")
	const goodReason = "Consumo +109,8% sobre el baseline desde el 12/09 14:00 durante 58 h, sin evento que lo explique; el FP bajó de 0,94 a 0,74."
	const goodAction = "Inspeccionar el medidor M-109 y su instalación."

	tests := []struct {
		name       string
		status     int
		body       string
		delay      time.Duration
		wantSource string
	}{
		{"respuesta válida se usa", 200, apiMessage(llmJSON(goodReason, goodAction), "end_turn"), 0, SourceLLM},
		{"número inventado se descarta", 200, apiMessage(llmJSON("Consumo +95% sobre el baseline.", goodAction), "end_turn"), 0, SourceTemplate},
		{"JSON roto se descarta", 200, apiMessage("esto no es JSON", "end_turn"), 0, SourceTemplate},
		{"campo vacío se descarta", 200, apiMessage(llmJSON("", goodAction), "end_turn"), 0, SourceTemplate},
		{"rechazo (refusal) usa la plantilla", 200, apiMessage("", "refusal"), 0, SourceTemplate},
		{"respuesta cortada (max_tokens) usa la plantilla", 200, apiMessage(`{"reason": "Consu`, "max_tokens"), 0, SourceTemplate},
		{"error 500 usa la plantilla", 500, `{"type":"error","error":{"type":"api_error","message":"boom"}}`, 0, SourceTemplate},
		{"timeout usa la plantilla", 200, apiMessage(llmJSON(goodReason, goodAction), "end_turn"), 500 * time.Millisecond, SourceTemplate},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotRequest map[string]any
			var gotHeaders http.Header

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				gotHeaders = req.Header.Clone()
				raw, _ := io.ReadAll(req.Body)
				_ = json.Unmarshal(raw, &gotRequest)
				time.Sleep(tc.delay)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.body)
			}))
			defer server.Close()

			explainer := NewLLMExplainer("test-key", "claude-opus-5", 200*time.Millisecond,
				option.WithBaseURL(server.URL), option.WithMaxRetries(0))
			got := explainer.Explain(context.Background(), m109)

			if got.Source != tc.wantSource {
				t.Fatalf("source = %s, se esperaba %s (reason: %s)", got.Source, tc.wantSource, got.Reason)
			}
			if tc.wantSource == SourceTemplate && got != TemplateExplanation(m109) {
				t.Errorf("se esperaba exactamente la plantilla")
			}
			if tc.wantSource == SourceLLM {
				if got.Reason != goodReason || got.RecommendedAction != goodAction {
					t.Errorf("no se usó el texto del LLM: %+v", got)
				}
				checkRequest(t, gotRequest, gotHeaders)
			}
		})
	}
}

func checkRequest(t *testing.T, body map[string]any, headers http.Header) {
	t.Helper()
	if headers.Get("X-Api-Key") != "test-key" {
		t.Errorf("falta la API key en la cabecera x-api-key")
	}
	if !strings.Contains(headers.Get("Anthropic-Beta"), "server-side-fallback-2026-07-01") {
		t.Errorf("falta la cabecera beta de fallbacks: %q", headers.Get("Anthropic-Beta"))
	}
	if body["model"] != "claude-opus-5" || body["fallbacks"] != "default" {
		t.Errorf("model/fallbacks incorrectos: %v / %v", body["model"], body["fallbacks"])
	}
	if _, ok := body["temperature"]; ok {
		t.Errorf("no se debe enviar temperature: Claude Opus 5 la rechaza")
	}
	outputConfig, _ := body["output_config"].(map[string]any)
	format, _ := outputConfig["format"].(map[string]any)
	if format["type"] != "json_schema" {
		t.Errorf("falta output_config.format json_schema: %v", body["output_config"])
	}
	messages, _ := json.Marshal(body["messages"])
	if !strings.Contains(string(messages), "REAL_ANOMALY") {
		t.Errorf("la evidencia del motor no viaja en el mensaje")
	}
}

func TestInventedNumbers(t *testing.T) {
	m109 := resultFor(t, "M-109")
	tests := []struct {
		text     string
		invented bool
	}{
		{"Consumo +109,8% sobre el baseline", false},
		{"subió cerca de 110%", false},
		{"desde el 12/09 14:00 durante 58 h", false},
		{"el FP bajó de 0,94 a 0,74", false},
		{"pasó de 1052,2 a 2207,6 kWh/día", false},
		{"umbral de 30% y rango 209–231 V", false},
		{"medidor M-109", false},
		{"con una confianza del 98%", false},
		{"Consumo +95% sobre el baseline", true},
		{"pasó a 2.207,6 kWh", true},
		{"la corriente subió 250%", true},
	}
	for _, tc := range tests {
		t.Run(tc.text, func(t *testing.T) {
			invented := inventedNumbers(tc.text, m109)
			if (len(invented) > 0) != tc.invented {
				t.Errorf("inventedNumbers(%q) = %v, se esperaba inventado=%v", tc.text, invented, tc.invented)
			}
		})
	}
}

func TestNewExplainerWithoutKeyUsesTemplate(t *testing.T) {
	explainer := NewExplainer(config.AIConfig{APIKey: "", Model: "claude-opus-5", Timeout: time.Second})
	if _, ok := explainer.(TemplateExplainer); !ok {
		t.Errorf("sin API key se esperaba TemplateExplainer, se obtuvo %T", explainer)
	}
}

func TestBuildParamsPerModel(t *testing.T) {
	tests := []struct {
		model         string
		wantEffort    bool
		wantTemp      bool
		wantFallbacks bool
	}{
		{"claude-opus-5", true, false, true},
		{"claude-sonnet-5", true, false, false},
		{"claude-haiku-4-5", false, true, false},
		{"claude-sonnet-4-6", false, true, false},
		{"claude-modelo-futuro-9", false, false, false},
	}
	for _, tc := range tests {
		t.Run(tc.model, func(t *testing.T) {
			params := NewLLMExplainer("k", tc.model, time.Second).buildParams("{}")
			raw, err := json.Marshal(params)
			if err != nil {
				t.Fatalf("serializar params: %v", err)
			}
			var body map[string]any
			if err := json.Unmarshal(raw, &body); err != nil {
				t.Fatalf("leer params: %v", err)
			}
			outputConfig, _ := body["output_config"].(map[string]any)
			_, hasEffort := outputConfig["effort"]
			_, hasTemp := body["temperature"]
			_, hasFallbacks := body["fallbacks"]

			if hasEffort != tc.wantEffort || hasTemp != tc.wantTemp || hasFallbacks != tc.wantFallbacks {
				t.Errorf("effort=%v temperature=%v fallbacks=%v; se esperaba %v %v %v",
					hasEffort, hasTemp, hasFallbacks, tc.wantEffort, tc.wantTemp, tc.wantFallbacks)
			}
			if _, ok := outputConfig["format"]; !ok {
				t.Errorf("todos los modelos deben recibir el esquema JSON (output_config.format)")
			}
			if tc.wantFallbacks != (len(params.Betas) > 0) {
				t.Errorf("la cabecera beta de fallbacks debe ir solo con fallbacks: %v", params.Betas)
			}
		})
	}
}
