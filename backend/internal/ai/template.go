package ai

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/josbp0107/ia-energy-management/internal/analysis"
	"github.com/josbp0107/ia-energy-management/internal/db"
)

func TemplateExplanation(r analysis.Result) Explanation {
	dev := r.Evidence.Deviation
	dq := r.Evidence.DataQuality

	switch {
	case r.Type == db.TypeRealAnomaly && dev != nil:
		reason := fmt.Sprintf(
			"Consumo %s %s del baseline diario (%s → %s kWh/día) desde el %s, durante %d h, sin un evento operativo que lo explique. La corriente cambió %s",
			signedPct(r.VariationPct), aboveOrBelow(r.VariationPct), num(r.BaselineKWhDay, 1), num(r.CurrentKWhDay, 1),
			formatTime(dev.Start), dev.HoursAffected, signedPct(dev.CurrentChangePct))
		action := fmt.Sprintf("Inspeccionar en sitio el medidor %s y su instalación para identificar cargas nuevas o equipos con falla", r.MeterID)

		if dev.PowerFactorChange < -analysis.PowerFactorDrop {
			reason += fmt.Sprintf(" y el factor de potencia bajó de %s a %s", num(dev.BaselinePowerFactor, 2), num(dev.ObservedPowerFactor, 2))
			action += "; la caída del factor de potencia apunta a carga inductiva (motores, compresores)"
		}
		reason += "."
		if len(dev.RelatedEvents) > 0 {
			reason += fmt.Sprintf(" El evento registrado (%q) no justifica el cambio.", dev.RelatedEvents[0].Description)
		}
		return Explanation{Reason: reason, RecommendedAction: action + ". Validar también la medición.", Source: SourceTemplate}

	case r.Type == db.TypeExplainableAnomaly && dev != nil:
		event := firstEventDescription(dev.RelatedEvents)
		return Explanation{
			Reason: fmt.Sprintf(
				"Consumo %s %s del baseline diario desde el %s (%d h), coincide con el evento operativo %q. La corriente cambió %s con factor de potencia estable.",
				signedPct(r.VariationPct), aboveOrBelow(r.VariationPct), formatTime(dev.Start), dev.HoursAffected, event, signedPct(dev.CurrentChangePct)),
			RecommendedAction: fmt.Sprintf(
				"Confirmar con operaciones que el cambio corresponde a %q y actualizar el baseline del medidor %s para reflejar la nueva carga.", event, r.MeterID),
			Source: SourceTemplate,
		}

	case r.Type == db.TypeFalsePositive && dev != nil:
		change := "Subida"
		if dev.Direction == "DOWN" {
			change = "Caída"
		}
		return Explanation{
			Reason: fmt.Sprintf(
				"%s de consumo de %s durante %d h desde el %s, coincide con el evento %q. El consumo se recuperó (último día %s vs. baseline).",
				change, pct(math.Abs(dev.MeanHourlyDeviationPct)), dev.HoursAffected, formatTime(dev.Start),
				firstEventDescription(dev.RelatedEvents), signedPct(r.VariationPct)),
			RecommendedAction: "No requiere acción: registrar como parada programada y cerrar la alerta.",
			Source:            SourceTemplate,
		}

	case r.Type == db.TypeDataQuality && dq != nil:
		pfValues := make([]string, len(dq.PowerFactorValues))
		for i, v := range dq.PowerFactorValues {
			pfValues[i] = num(v, 2)
		}
		reason := fmt.Sprintf(
			"%d lecturas inválidas desde el %s: voltaje entre %s y %s V (rango válido %s–%s V) y factor de potencia con valores %s.",
			dq.InvalidReadings, formatTime(dq.FirstInvalid), num(dq.VoltageRange[0], 1), num(dq.VoltageRange[1], 1),
			num(analysis.VoltageMin, 0), num(analysis.VoltageMax, 0), strings.Join(pfValues, " / "))
		if dq.ConsumptionStable {
			reason += fmt.Sprintf(" El consumo se mantiene estable (%s vs. baseline): el problema es de medición, no de consumo.", signedPct(r.VariationPct))
		}
		if len(dq.CorroboratingEvents) > 0 {
			reason += fmt.Sprintf(" Lo corrobora el evento %q.", dq.CorroboratingEvents[0])
		}
		return Explanation{
			Reason:            reason,
			RecommendedAction: fmt.Sprintf("Revisar el medidor %s y su comunicación (calibración, cableado, transmisión) y no usar sus lecturas para decisiones hasta validarlo.", r.MeterID),
			Source:            SourceTemplate,
		}
	}

	return Explanation{
		Reason:            fmt.Sprintf("Consumo dentro del rango normal (%s vs. baseline).", signedPct(r.VariationPct)),
		RecommendedAction: "Sin acción requerida.",
		Source:            SourceTemplate,
	}
}

func num(v float64, decimals int) string {
	return strings.Replace(strconv.FormatFloat(v, 'f', decimals, 64), ".", ",", 1)
}

func pct(v float64) string {
	return num(v, 1) + "%"
}

func signedPct(v float64) string {
	if v > 0 {
		return "+" + pct(v)
	}
	return pct(v)
}

func aboveOrBelow(v float64) string {
	if v < 0 {
		return "por debajo"
	}
	return "por encima"
}

func formatTime(t time.Time) string {
	return t.Format("02/01 15:04")
}

func firstEventDescription(events []analysis.RelatedEvent) string {
	for _, e := range events {
		if analysis.ExplainsDeviation(e.EventType) {
			return e.Description
		}
	}
	return "evento operativo"
}
