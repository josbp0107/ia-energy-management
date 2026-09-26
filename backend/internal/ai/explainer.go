package ai

import (
	"context"
	"log"

	"github.com/josbp0107/ia-energy-management/internal/analysis"
	"github.com/josbp0107/ia-energy-management/internal/config"
)

const (
	SourceLLM      = "llm"
	SourceTemplate = "template"
)

type Explanation struct {
	Reason            string `json:"reason"`
	RecommendedAction string `json:"recommended_action"`
	Source            string `json:"source"` // llm | template
}

type Explainer interface {
	Explain(ctx context.Context, r analysis.Result) Explanation
}

type TemplateExplainer struct{}

func (TemplateExplainer) Explain(_ context.Context, r analysis.Result) Explanation {
	return TemplateExplanation(r)
}

func NewExplainer(cfg config.AIConfig) Explainer {
	if cfg.APIKey == "" {
		log.Println("ANTHROPIC_API_KEY vacía: las explicaciones usarán la plantilla")
		return TemplateExplainer{}
	}
	return NewLLMExplainer(cfg.APIKey, cfg.Model, cfg.Timeout)
}
