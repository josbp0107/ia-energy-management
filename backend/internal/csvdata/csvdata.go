package csvdata

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/josbp0107/ia-energy-management/internal/db"
)

const (
	readingTimeLayout = "2006-01-02 15:04:05"
	eventTimeLayout   = "2006-01-02 15:04"
)

func LoadMeters(path string) (map[string]db.Meter, error) {
	rows, err := readCSV(path)
	if err != nil {
		return nil, err
	}

	meters := make(map[string]db.Meter, len(rows))
	for _, row := range rows {
		meters[row[0]] = db.Meter{MeterID: row[0], Name: row[1], Location: row[2]}
	}
	return meters, nil
}

func LoadReadings(path string) ([]db.Reading, error) {
	rows, err := readCSV(path)
	if err != nil {
		return nil, err
	}

	readings := make([]db.Reading, 0, len(rows))
	for i, row := range rows {
		line := i + 2
		ts, err := time.Parse(readingTimeLayout, row[1])
		if err != nil {
			return nil, fmt.Errorf("línea %d: timestamp: %w", line, err)
		}
		nums, err := parseFloats(row[2:6])
		if err != nil {
			return nil, fmt.Errorf("línea %d: %w", line, err)
		}
		readings = append(readings, db.Reading{
			MeterID:        row[0],
			Timestamp:      ts,
			ConsumptionKWh: nums[0],
			VoltageV:       nums[1],
			CurrentA:       nums[2],
			PowerFactor:    nums[3],
			Status:         row[6],
		})
	}
	return readings, nil
}

func LoadEvents(path string) ([]db.Event, error) {
	rows, err := readCSV(path)
	if err != nil {
		return nil, err
	}

	events := make([]db.Event, 0, len(rows))
	for i, row := range rows {
		ts, err := time.Parse(eventTimeLayout, row[1])
		if err != nil {
			return nil, fmt.Errorf("línea %d: event_timestamp: %w", i+2, err)
		}
		events = append(events, db.Event{
			MeterID:        row[0],
			EventTimestamp: ts,
			EventType:      row[2],
			Description:    row[3],
		})
	}
	return events, nil
}

// readCSV devuelve las filas del archivo sin la cabecera.
func readCSV(path string) ([][]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	rows, err := csv.NewReader(file).ReadAll()
	if err != nil {
		return nil, err
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("%s no tiene datos", path)
	}
	return rows[1:], nil
}

func parseFloats(values []string) ([]float64, error) {
	nums := make([]float64, len(values))
	for i, v := range values {
		n, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, fmt.Errorf("número inválido %q: %w", v, err)
		}
		nums[i] = n
	}
	return nums, nil
}
