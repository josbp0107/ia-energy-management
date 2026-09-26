package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort           string
	FrontendOrigin    string
	DataDir           string
	AnalysisStepDelay time.Duration
	AI                AIConfig
	Auth              AuthConfig
	DB                DBConfig
}

type AuthConfig struct {
	Email    string
	Password string
	Name     string
}

type AIConfig struct {
	APIKey  string
	Model   string
	Timeout time.Duration
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
		FrontendOrigin: optional("FRONTEND_ORIGIN", "http://localhost:5173"),
		DataDir:        optional("DATA_DIR", "../data"),
		Auth: AuthConfig{
			Email:    required("DEMO_EMAIL"),
			Password: required("DEMO_PASSWORD"),
			Name:     optional("DEMO_NAME", "Operador Demo"),
		},
		DB: DBConfig{
			Host:     required("DB_HOST"),
			Port:     required("DB_PORT"),
			User:     required("DB_USER"),
			Password: required("DB_PASSWORD"),
			Name:     required("DB_NAME"),
			SSLMode:  optional("DB_SSLMODE", "disable"),
		},
	}

	delayMs, err := strconv.Atoi(optional("ANALYSIS_STEP_DELAY_MS", "700"))
	if err != nil || delayMs < 0 {
		return Config{}, fmt.Errorf("ANALYSIS_STEP_DELAY_MS debe ser un entero >= 0")
	}
	cfg.AnalysisStepDelay = time.Duration(delayMs) * time.Millisecond

	timeoutSec, err := strconv.Atoi(optional("AI_TIMEOUT_SECONDS", "60"))
	if err != nil || timeoutSec <= 0 {
		return Config{}, fmt.Errorf("AI_TIMEOUT_SECONDS debe ser un entero > 0")
	}
	cfg.AI = AIConfig{
		APIKey:  os.Getenv("ANTHROPIC_API_KEY"),
		Model:   optional("ANTHROPIC_MODEL", "claude-opus-5"),
		Timeout: time.Duration(timeoutSec) * time.Second,
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
