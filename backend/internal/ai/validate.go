package ai

import (
	"encoding/json"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/josbp0107/ia-energy-management/internal/analysis"
)

var numberPattern = regexp.MustCompile(`\d+(?:[.,]\d+)?`)

func inventedNumbers(text string, r analysis.Result) []string {
	allowed := allowedNumbers(r)
	var invented []string

	for _, raw := range numberPattern.FindAllString(text, -1) {
		normalized := strings.Replace(raw, ",", ".", 1)
		value, err := strconv.ParseFloat(normalized, 64)
		if err != nil {
			invented = append(invented, raw)
			continue
		}
		decimals := 0
		if i := strings.Index(normalized, "."); i >= 0 {
			decimals = len(normalized) - i - 1
		}
		if !matchesAny(value, decimals, allowed) {
			invented = append(invented, raw)
		}
	}
	return invented
}

func matchesAny(value float64, decimals int, allowed []float64) bool {
	factor := math.Pow(10, float64(decimals))
	for _, a := range allowed {
		if math.Abs(math.Round(a*factor)/factor-value) < 1e-9 {
			return true
		}
	}
	return false
}

func allowedNumbers(r analysis.Result) []float64 {
	allowed := []float64{
		0,
		analysis.DeviationThreshold * 100,
		analysis.VoltageMin,
		analysis.VoltageMax,
		analysis.EventWindow.Hours(),
		analysis.LongDurationHours,
		analysis.BaselineDays,
		analysis.MinInvalidReadings,
	}

	data, err := json.Marshal(r)
	if err != nil {
		return allowed
	}
	var tree any
	if err := json.Unmarshal(data, &tree); err != nil {
		return allowed
	}

	var walk func(node any)
	walk = func(node any) {
		switch v := node.(type) {
		case float64:
			allowed = append(allowed, math.Abs(v))
			if math.Abs(v) > 0 && math.Abs(v) <= 1 {
				allowed = append(allowed, math.Abs(v)*100) // confianza 0,95 -> "95%"
			}
		case string:
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				allowed = append(allowed, float64(t.Year()), float64(t.Month()), float64(t.Day()),
					float64(t.Hour()), float64(t.Minute()))
				return
			}
			for _, raw := range numberPattern.FindAllString(v, -1) {
				if n, err := strconv.ParseFloat(strings.Replace(raw, ",", ".", 1), 64); err == nil {
					allowed = append(allowed, n)
				}
			}
		case []any:
			for _, item := range v {
				walk(item)
			}
		case map[string]any:
			for _, item := range v {
				walk(item)
			}
		}
	}
	walk(tree)
	return allowed
}
