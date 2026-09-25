// go run ./cmd/seed (carpeta en DATA_DIR)
package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"gorm.io/gorm"

	"github.com/josbp0107/ia-energy-management/internal/config"
	"github.com/josbp0107/ia-energy-management/internal/db"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("configuración: %v", err)
	}

	database, err := db.Connect(cfg.DB)
	if err != nil {
		log.Fatalf("conectar a la base de datos: %v", err)
	}

	meterInfo, err := loadMeters(filepath.Join(cfg.DataDir, "meters.csv"))
	if err != nil {
		log.Fatalf("leer meters.csv: %v", err)
	}
	readings, err := loadReadings(filepath.Join(cfg.DataDir, "readings.csv"))
	if err != nil {
		log.Fatalf("leer readings.csv: %v", err)
	}
	events, err := loadEvents(filepath.Join(cfg.DataDir, "events.csv"))
	if err != nil {
		log.Fatalf("leer events.csv: %v", err)
	}

	var meters []db.Meter
	seen := map[string]bool{}
	for _, r := range readings {
		if seen[r.MeterID] {
			continue
		}
		seen[r.MeterID] = true
		meter, ok := meterInfo[r.MeterID]
		if !ok {
			log.Printf("%s no esta en meters.csv, se usa un nombre generico", r.MeterID)
			meter = db.Meter{MeterID: r.MeterID, Name: "Medidor " + r.MeterID, Location: "Sin ubicacion"}
		}
		meters = append(meters, meter)
	}

	err = database.Transaction(func(tx *gorm.DB) error {
		for _, model := range []any{&db.Reading{}, &db.Event{}, &db.Meter{}} {
			if err := tx.Where("1 = 1").Delete(model).Error; err != nil {
				return err
			}
		}
		if err := tx.Create(&meters).Error; err != nil {
			return err
		}
		if err := tx.CreateInBatches(&readings, 500).Error; err != nil {
			return err
		}
		return tx.Create(&events).Error
	})
	if err != nil {
		log.Fatalf("guardar datos: %v", err)
	}

	fmt.Printf("Seed completo: %d medidores, %d lecturas, %d eventos\n", len(meters), len(readings), len(events))
}

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

func loadMeters(path string) (map[string]db.Meter, error) {
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

func loadReadings(path string) ([]db.Reading, error) {
	rows, err := readCSV(path)
	if err != nil {
		return nil, err
	}

	readings := make([]db.Reading, 0, len(rows))
	for i, row := range rows {
		line := i + 2
		ts, err := time.Parse("2006-01-02 15:04:05", row[1])
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

func loadEvents(path string) ([]db.Event, error) {
	rows, err := readCSV(path)
	if err != nil {
		return nil, err
	}

	events := make([]db.Event, 0, len(rows))
	for i, row := range rows {
		ts, err := time.Parse("2006-01-02 15:04", row[1])
		if err != nil {
			return nil, fmt.Errorf("linea %d: event_timestamp: %w", i+2, err)
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

func parseFloats(values []string) ([]float64, error) {
	nums := make([]float64, len(values))
	for i, v := range values {
		n, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return nil, fmt.Errorf("numero invalido %q: %w", v, err)
		}
		nums[i] = n
	}
	return nums, nil
}
