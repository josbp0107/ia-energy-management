package analysis

import (
	"math"
	"sort"
	"time"

	"github.com/josbp0107/ia-energy-management/internal/db"
)

type hourBaseline struct {
	kwh         float64
	current     float64
	powerFactor float64
}

type baseline struct {
	byHour     map[int]hourBaseline
	meterRatio float64
}

func buildBaseline(readings []db.Reading, baselineEnd time.Time) baseline {
	kwh := map[int][]float64{}
	current := map[int][]float64{}
	pf := map[int][]float64{}
	var ratios []float64

	for _, r := range readings {
		if !r.Timestamp.Before(baselineEnd) {
			continue
		}
		h := r.Timestamp.Hour()
		kwh[h] = append(kwh[h], r.ConsumptionKWh)
		current[h] = append(current[h], r.CurrentA)
		pf[h] = append(pf[h], r.PowerFactor)
		if ratio, ok := energyRatio(r); ok {
			ratios = append(ratios, ratio)
		}
	}

	b := baseline{byHour: map[int]hourBaseline{}, meterRatio: median(ratios)}
	for h := range kwh {
		b.byHour[h] = hourBaseline{
			kwh:         median(kwh[h]),
			current:     median(current[h]),
			powerFactor: median(pf[h]),
		}
	}
	return b
}

func BaselineEnd(readings []db.Reading) time.Time {
	if len(readings) == 0 {
		return time.Time{}
	}
	first := readings[0].Timestamp
	for _, r := range readings {
		if r.Timestamp.Before(first) {
			first = r.Timestamp
		}
	}
	return first.Truncate(24*time.Hour).AddDate(0, 0, BaselineDays)
}

type HourBaseline struct {
	Hour        int     `json:"hour"`
	KWh         float64 `json:"kwh"`
	LowerKWh    float64 `json:"lower_kwh"`
	UpperKWh    float64 `json:"upper_kwh"`
	CurrentA    float64 `json:"current_a"`
	PowerFactor float64 `json:"power_factor"`
}

func HourlyBaseline(readings []db.Reading) []HourBaseline {
	base := buildBaseline(readings, BaselineEnd(readings))
	hours := make([]HourBaseline, 0, len(base.byHour))
	for h, b := range base.byHour {
		hours = append(hours, HourBaseline{
			Hour:        h,
			KWh:         round(b.kwh, 2),
			LowerKWh:    round(b.kwh*(1-DeviationThreshold), 2),
			UpperKWh:    round(b.kwh*(1+DeviationThreshold), 2),
			CurrentA:    round(b.current, 2),
			PowerFactor: round(b.powerFactor, 3),
		})
	}
	sort.Slice(hours, func(i, j int) bool { return hours[i].Hour < hours[j].Hour })
	return hours
}

func energyRatio(r db.Reading) (ratio float64, ok bool) {
	power := r.VoltageV * r.CurrentA * r.PowerFactor / 1000
	if power <= 0 {
		return 0, false
	}
	return r.ConsumptionKWh / power, true
}

func isInvalidReading(r db.Reading, meterRatio float64) bool {
	if r.VoltageV < VoltageMin || r.VoltageV > VoltageMax {
		return true
	}
	if r.PowerFactor > 1 || r.PowerFactor <= 0 {
		return true
	}
	ratio, ok := energyRatio(r)
	if !ok {
		return r.ConsumptionKWh > 0
	}
	if meterRatio == 0 {
		return false
	}
	return math.Abs(ratio/meterRatio-1) > RatioTolerance
}

type dayTotal struct {
	day time.Time
	kwh float64
}

func dailyTotals(readings []db.Reading) []dayTotal {
	sums := map[time.Time]float64{}
	for _, r := range readings {
		day := r.Timestamp.Truncate(24 * time.Hour)
		sums[day] += r.ConsumptionKWh
	}

	days := make([]dayTotal, 0, len(sums))
	for day, kwh := range sums {
		days = append(days, dayTotal{day: day, kwh: kwh})
	}
	sort.Slice(days, func(i, j int) bool { return days[i].day.Before(days[j].day) })
	return days
}
