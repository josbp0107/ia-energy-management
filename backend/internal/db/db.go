package db

import (
	"fmt"
	"net/url"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/josbp0107/ia-energy-management/internal/config"
)

func Connect(cfg config.DBConfig) (*gorm.DB, error) {
	dsn := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(cfg.User, cfg.Password),
		Host:     cfg.Host + ":" + cfg.Port,
		Path:     cfg.Name,
		RawQuery: url.Values{"sslmode": {cfg.SSLMode}, "TimeZone": {"UTC"}}.Encode(),
	}

	database, err := gorm.Open(postgres.Open(dsn.String()), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("abrir conexión: %w", err)
	}

	err = database.AutoMigrate(&Meter{}, &Reading{}, &Event{}, &AnalysisRun{}, &Anomaly{})
	if err != nil {
		return nil, fmt.Errorf("migrar esquema: %w", err)
	}

	return database, nil
}

func FailInterruptedRuns(database *gorm.DB) error {
	return database.Model(&AnalysisRun{}).
		Where("status = ?", RunStatusRunning).
		Updates(map[string]any{
			"status":      RunStatusFailed,
			"error":       "interrumpido por reinicio del servidor",
			"finished_at": time.Now().UTC(),
		}).Error
}
