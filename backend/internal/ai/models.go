package ai

import "strings"

type modelFeatures struct {
	effort      bool
	temperature bool
	fallbacks   bool
}

var effortModels = []string{
	"claude-opus-5", "claude-fable-5", "claude-mythos-5", "claude-sonnet-5",
	"claude-opus-4-8", "claude-opus-4-7",
}

var temperatureModels = []string{
	"claude-haiku-4", "claude-sonnet-4-6", "claude-opus-4-6", "claude-sonnet-4-5", "claude-opus-4-5",
}

var fallbackModels = []string{"claude-opus-5", "claude-fable-5"}

func featuresFor(model string) modelFeatures {
	return modelFeatures{
		effort:      hasAnyPrefix(model, effortModels),
		temperature: hasAnyPrefix(model, temperatureModels),
		fallbacks:   hasAnyPrefix(model, fallbackModels),
	}
}

func hasAnyPrefix(model string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(model, p) {
			return true
		}
	}
	return false
}
