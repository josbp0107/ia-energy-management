// go run ./cmd/seed (carpeta en DATA_DIR)
package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"

	"gorm.io/gorm"

	"github.com/josbp0107/ia-energy-management/internal/config"
	"github.com/josbp0107/ia-energy-management/internal/csvdata"
	"github.com/josbp0107/ia-energy-management/internal/db"
)

func main() {
	ifEmpty := flag.Bool("if-empty", false, "cargar solo si no hay medidores")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("configuración: %v", err)
	}

	database, err := db.Connect(cfg.DB)
	if err != nil {
		log.Fatalf("conectar a la base de datos: %v", err)
	}

	if *ifEmpty {
		var count int64
		if err := database.Model(&db.Meter{}).Count(&count).Error; err != nil {
			log.Fatalf("contar medidores: %v", err)
		}
		if count > 0 {
			fmt.Printf("La base ya tiene %d medidores: no se recarga (-if-empty)\n", count)
			return
		}
	}

	meterInfo, err := csvdata.LoadMeters(filepath.Join(cfg.DataDir, "meters.csv"))
	if err != nil {
		log.Fatalf("leer meters.csv: %v", err)
	}
	readings, err := csvdata.LoadReadings(filepath.Join(cfg.DataDir, "readings.csv"))
	if err != nil {
		log.Fatalf("leer readings.csv: %v", err)
	}
	events, err := csvdata.LoadEvents(filepath.Join(cfg.DataDir, "events.csv"))
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
		for _, model := range []any{&db.Anomaly{}, &db.AnalysisRun{}, &db.Reading{}, &db.Event{}, &db.Meter{}} {
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

	fmt.Printf("Seed completo: %d medidores, %d lecturas, %d eventos (análisis previos borrados)\n", len(meters), len(readings), len(events))
}
