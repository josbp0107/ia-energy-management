// Config variables de entorno
package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort        string
	FrontendOrigin string
	DataDir        string
	DB             DBConfig
}

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

func Load() (Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("sin archivo .env, se usan las variables de entorno del sistema")
	}

	var missing []string
	required := func(key string) string {
		value := os.Getenv(key)
		if value == "" {
			missing = append(missing, key)
		}
		return value
	}

	cfg := Config{
		AppPort:        optional("APP_PORT", "8080"),
		FrontendOrigin: optional("FRONTEND_ORIGIN", "http://localhost:3000"),
		DataDir:        optional("DATA_DIR", "../data"),
		DB: DBConfig{
			Host:     required("DB_HOST"),
			Port:     required("DB_PORT"),
			User:     required("DB_USER"),
			Password: required("DB_PASSWORD"),
			Name:     required("DB_NAME"),
			SSLMode:  optional("DB_SSLMODE", "disable"),
		},
	}

	if len(missing) > 0 {
		return Config{}, fmt.Errorf("faltan variables de entorno: %s (copia backend/.env.example a backend/.env)",
			strings.Join(missing, ", "))
	}
	return cfg, nil
}

func optional(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
