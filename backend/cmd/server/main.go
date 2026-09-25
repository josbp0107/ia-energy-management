// go run ./cmd/server
package main

import (
	"log"

	"github.com/josbp0107/ia-energy-management/internal/config"
	"github.com/josbp0107/ia-energy-management/internal/db"
	"github.com/josbp0107/ia-energy-management/internal/httpapi"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Config: %v", err)
	}

	database, err := db.Connect(cfg.DB)
	if err != nil {
		log.Fatalf("Connect to database: %v", err)
	}

	router := httpapi.NewRouter(database, cfg.FrontendOrigin)

	if err := router.Run(":" + cfg.AppPort); err != nil {
		log.Fatalf("Server: %v", err)
	}
}
