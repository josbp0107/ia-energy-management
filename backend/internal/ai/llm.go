package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/shared/constant"

	"github.com/josbp0107/ia-energy-management/internal/analysis"
)

const systemPrompt = `Eres un analista de energía que explica anomalías de medidores eléctricos a un jefe de mantenimiento.
Un motor estadístico ya clasificó la anomalía; NO cambies su tipo, severidad ni confianza.
Recibirás el resultado del motor en JSON. Responde en español, breve y directo:
- "reason": máximo 2 frases (unas 45 palabras). Qué cambió, desde cuándo y por qué se clasificó así.
- "recommended_action": 1 frase (unas 30 palabras) con una acción concreta.
La pantalla ya muestra la confianza, la prioridad y la tabla de variables: no las repitas, cita solo las 2 o 3 cifras que mejor lo explican.

Vocabulario (nunca escribas los códigos en inglés ni nombres de campos del JSON):
- REAL_ANOMALY = anomalía real; EXPLAINABLE_ANOMALY = anomalía explicada por un cambio operativo;
  FALSE_POSITIVE = falso positivo; DATA_QUALITY = problema de calidad de datos.
- HIGH/MEDIUM/LOW = alta/media/baja. Un evento UNKNOWN es "un evento sin causa operativa reportada".
- Traduce al español la descripción de los eventos en vez de citarla en inglés.

Reglas de cifras:
- Usa solo cifras que aparezcan en el JSON (puedes redondearlas). No calcules ni inventes números nuevos.
- Decimales con coma (109,8) y sin separador de miles (2207,6). Fechas como "12/09 14:00".

Según el tipo:
- Anomalía real: ningún evento la explica; si hay un evento sin causa reportada, di que no justifica el cambio. Acción: inspección en sitio.
- Explicada por cambio operativo: coincide con el evento; sugiere validarlo y actualizar el baseline.
- Falso positivo: coincide con una parada programada; no requiere acción.
- Calidad de datos: el problema es la medición, no el consumo; sugiere revisar el medidor.`

var outputSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"reason":             map[string]any{"type": "string"},
		"recommended_action": map[string]any{"type": "string"},
	},
	"required":             []string{"reason", "recommended_action"},
	"additionalProperties": false,
}

type LLMExplainer struct {
	client  anthropic.Client
	model   string
	timeout time.Duration
}

func NewLLMExplainer(apiKey, model string, timeout time.Duration, opts ...option.RequestOption) *LLMExplainer {
	opts = append([]option.RequestOption{option.WithAPIKey(apiKey)}, opts...)
	return &LLMExplainer{client: anthropic.NewClient(opts...), model: model, timeout: timeout}
}

func (e *LLMExplainer) Explain(ctx context.Context, r analysis.Result) Explanation {
	explanation, err := e.generate(ctx, r)
	if err != nil {
		log.Printf("explicación LLM de %s descartada, se usa la plantilla: %v", r.MeterID, err)
		return TemplateExplanation(r)
	}
	return explanation
}

func (e *LLMExplainer) generate(ctx context.Context, r analysis.Result) (Explanation, error) {
	evidence, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return Explanation{}, fmt.Errorf("serializar resultado: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	resp, err := e.client.Beta.Messages.New(ctx, anthropic.BetaMessageNewParams{
		Model:     anthropic.Model(e.model),
		MaxTokens: 16000,
		System:    []anthropic.BetaTextBlockParam{{Text: systemPrompt}},
		Messages: []anthropic.BetaMessageParam{
			anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock("Resultado del motor:\n" + string(evidence))),
		},
		OutputConfig: anthropic.BetaOutputConfigParam{
			Effort: anthropic.BetaOutputConfigEffortLow,
			Format: anthropic.BetaJSONOutputFormatParam{Schema: outputSchema},
		},
		Fallbacks: anthropic.BetaFallbacksParamUnion{OfDefault: constant.ValueOf[constant.Default]()},
		Betas:     []anthropic.AnthropicBeta{anthropic.AnthropicBetaServerSideFallback2026_07_01},
	})
	if err != nil {
		return Explanation{}, fmt.Errorf("llamada a la API: %w", err)
	}
	if resp.StopReason != anthropic.BetaStopReasonEndTurn {
		return Explanation{}, fmt.Errorf("respuesta incompleta (stop_reason=%s)", resp.StopReason)
	}

	var text strings.Builder
	for _, block := range resp.Content {
		if block.Type == "text" {
			text.WriteString(block.Text)
		}
	}

	var out struct {
		Reason            string `json:"reason"`
		RecommendedAction string `json:"recommended_action"`
	}
	if err := json.Unmarshal([]byte(text.String()), &out); err != nil {
		return Explanation{}, fmt.Errorf("JSON inválido: %w", err)
	}
	out.Reason = strings.TrimSpace(out.Reason)
	out.RecommendedAction = strings.TrimSpace(out.RecommendedAction)
	if out.Reason == "" || out.RecommendedAction == "" {
		return Explanation{}, fmt.Errorf("reason o recommended_action vacíos")
	}
	if invented := inventedNumbers(out.Reason+" "+out.RecommendedAction, r); len(invented) > 0 {
		return Explanation{}, fmt.Errorf("cita números que no están en la evidencia: %v", invented)
	}

	return Explanation{Reason: out.Reason, RecommendedAction: out.RecommendedAction, Source: SourceLLM}, nil
}
