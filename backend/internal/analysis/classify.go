package analysis

import (
	"math"
	"sort"
	"time"

	"github.com/josbp0107/ia-energy-management/internal/db"
)

type flaggedHour struct {
	reading   db.Reading
	deviation float64
	baseline  hourBaseline
}

func analyzeMeter(meterID string, readings []db.Reading, events []db.Event, baselineEnd time.Time) Result {
	base := buildBaseline(readings, baselineEnd)

	var baselineDays []float64
	days := dailyTotals(readings)
	for _, d := range days {
		if d.day.Before(baselineEnd) {
			baselineDays = append(baselineDays, d.kwh)
		}
	}
	baseDay := median(baselineDays)
	lastDay := 0.0
	if len(days) > 0 {
		lastDay = days[len(days)-1].kwh
	}
	variation := 0.0
	if baseDay > 0 {
		variation = (lastDay/baseDay - 1) * 100
	}

	var flagged []flaggedHour
	var invalid []db.Reading
	for _, r := range readings {
		if b, ok := base.byHour[r.Timestamp.Hour()]; ok && b.kwh > 0 {
			dev := r.ConsumptionKWh/b.kwh - 1
			if math.Abs(dev) > DeviationThreshold {
				flagged = append(flagged, flaggedHour{reading: r, deviation: dev, baseline: b})
			}
		}
		if isInvalidReading(r, base.meterRatio) {
			invalid = append(invalid, r)
		}
	}

	result := Result{
		MeterID:        meterID,
		Type:           db.TypeNormal,
		Severity:       db.SeverityNone,
		VariationPct:   round(variation, 1),
		BaselineKWhDay: round(baseDay, 1),
		CurrentKWhDay:  round(lastDay, 1),
	}

	switch {
	case len(invalid) >= MinInvalidReadings:
		classifyDataQuality(&result, invalid, events, variation)
	case len(flagged) > 0:
		classifyDeviation(&result, flagged, events, variation)
	}

	result.PriorityScore = priorityScore(result.Severity, result.Confidence, result.VariationPct)
	return result
}

func classifyDataQuality(result *Result, invalid []db.Reading, events []db.Event, variation float64) {
	voltages := make([]float64, 0, len(invalid))
	pfSet := map[float64]bool{}
	first := invalid[0].Timestamp
	for _, r := range invalid {
		voltages = append(voltages, r.VoltageV)
		pfSet[r.PowerFactor] = true
		if r.Timestamp.Before(first) {
			first = r.Timestamp
		}
	}
	pfValues := make([]float64, 0, len(pfSet))
	for pf := range pfSet {
		pfValues = append(pfValues, pf)
	}
	sort.Float64s(pfValues)

	corroborating := []string{}
	for _, e := range events {
		if e.EventType == db.EventDataQuality {
			corroborating = append(corroborating, e.Description)
		}
	}

	vMin, vMax := minMax(voltages)
	stable := math.Abs(variation) < StableVariationPct

	result.Anomaly = true
	result.Type = db.TypeDataQuality
	result.Severity = db.SeverityHigh
	result.Evidence.ConfidenceFactors = []ConfidenceFactor{
		{"Base: lecturas físicamente imposibles", 0.75, true},
		{"Consumo estable (< 5 %): el problema es la medición", 0.10, stable},
		{"Evento DATA_QUALITY que lo corrobora", 0.10, len(corroborating) > 0},
	}
	result.Confidence = confidence(sumFactors(result.Evidence.ConfidenceFactors))
	result.Evidence.DataQuality = &DataQualityEvidence{
		InvalidReadings:     len(invalid),
		FirstInvalid:        first,
		VoltageRange:        [2]float64{vMin, vMax},
		PowerFactorValues:   pfValues,
		ConsumptionStable:   stable,
		CorroboratingEvents: corroborating,
	}
}

func classifyDeviation(result *Result, flagged []flaggedHour, events []db.Event, variation float64) {
	start, end := flagged[0].reading.Timestamp, flagged[0].reading.Timestamp
	var devs, currentChanges, pfChanges, baseCurrent, obsCurrent, basePF, obsPF []float64
	for _, f := range flagged {
		ts := f.reading.Timestamp
		if ts.Before(start) {
			start = ts
		}
		if ts.After(end) {
			end = ts
		}
		devs = append(devs, f.deviation)
		currentChanges = append(currentChanges, f.reading.CurrentA/f.baseline.current-1)
		pfChanges = append(pfChanges, f.reading.PowerFactor-f.baseline.powerFactor)
		baseCurrent = append(baseCurrent, f.baseline.current)
		obsCurrent = append(obsCurrent, f.reading.CurrentA)
		basePF = append(basePF, f.baseline.powerFactor)
		obsPF = append(obsPF, f.reading.PowerFactor)
	}
	hours := len(flagged)
	currentChange := mean(currentChanges)
	pfChange := mean(pfChanges)

	direction := "UP"
	if mean(devs) <= 0 {
		direction = "DOWN"
	}

	related := []RelatedEvent{}
	explainingType := ""
	for _, e := range events {
		diff := e.EventTimestamp.Sub(start)
		if diff < 0 {
			diff = -diff
		}
		if diff > EventWindow {
			continue
		}
		related = append(related, RelatedEvent{
			EventType:      e.EventType,
			Description:    e.Description,
			EventTimestamp: e.EventTimestamp,
		})
		if ExplainsDeviation(e.EventType) && explainingType == "" {
			explainingType = e.EventType
		}
	}

	result.Anomaly = true
	switch explainingType {
	case db.EventScheduledOutage:
		result.Anomaly = false
		result.Type = db.TypeFalsePositive
		result.Severity = db.SeverityLow
		result.Evidence.ConfidenceFactors = []ConfidenceFactor{
			{"Base: coincide con una parada programada", 0.85, true},
			{"Duración ≤ 24 h, acotada a la parada", 0.10, hours <= LongDurationHours},
		}
	case db.EventOperationalChange:
		result.Type = db.TypeExplainableAnomaly
		result.Severity = db.SeverityMedium
		result.Evidence.ConfidenceFactors = []ConfidenceFactor{
			{"Base: coincide con un cambio operativo", 0.80, true},
			{"La corriente sube > 15 %: el aumento es real", 0.10, currentChange > CurrentRise/2},
		}
	default:
		result.Type = db.TypeRealAnomaly
		result.Severity = db.SeverityHigh
		result.Evidence.ConfidenceFactors = []ConfidenceFactor{
			{"Base: desviación sin evento que la explique", 0.70, true},
			{"La corriente sube > 30 %", 0.10, currentChange > CurrentRise},
			{"El factor de potencia cae > 0,10", 0.10, pfChange < -PowerFactorDrop},
			{"Dura ≥ 24 h (no es un pico pasajero)", 0.05, hours >= LongDurationHours},
			{"Variación diaria > 50 %", 0.03, math.Abs(variation) > LargeVariationPct},
		}
	}
	result.Confidence = confidence(sumFactors(result.Evidence.ConfidenceFactors))

	result.Evidence.Deviation = &DeviationEvidence{
		Start:                  start,
		End:                    end,
		HoursAffected:          hours,
		Direction:              direction,
		MeanHourlyDeviationPct: round(mean(devs)*100, 1),
		CurrentChangePct:       round(currentChange*100, 1),
		PowerFactorChange:      round(pfChange, 3),
		BaselineCurrentA:       round(mean(baseCurrent), 1),
		ObservedCurrentA:       round(mean(obsCurrent), 1),
		BaselinePowerFactor:    round(mean(basePF), 3),
		ObservedPowerFactor:    round(mean(obsPF), 3),
		RelatedEvents:          related,
	}
}

func sumFactors(factors []ConfidenceFactor) float64 {
	total := 0.0
	for _, f := range factors {
		if f.Applied {
			total += f.Points
		}
	}
	return total
}

func confidence(value float64) float64 {
	return round(math.Min(value, MaxConfidence), 2)
}

func priorityScore(severity string, confidence, variationPct float64) float64 {
	return round(severityWeight[severity]*confidence*(1+math.Abs(variationPct)/100), 2)
}
