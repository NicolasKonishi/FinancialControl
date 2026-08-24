package config

import (
	"os"
	"strings"
)

// Config holds runtime settings loaded from the environment.
type Config struct {
	HTTPAddr          string
	SQLitePath        string
	PythonAnalysisURL string
	FrontendURL       string
	MigrationsPath    string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() Config {
	return Config{
		HTTPAddr:          getEnv("HTTP_ADDR", ":8080"),
		SQLitePath:        getEnv("SQLITE_PATH", "./data/fluxo.db"),
		PythonAnalysisURL: getEnv("PYTHON_ANALYSIS_URL", "http://localhost:8000"),
		FrontendURL:       getEnv("FRONTEND_URL", "http://localhost:5173"),
		MigrationsPath:    getEnv("MIGRATIONS_PATH", "migrations"),
	}
}

// PublicAPIURL turns HTTP_ADDR into a browser-friendly base URL.
// Examples: ":8080" -> "http://localhost:8080", "127.0.0.1:8080" -> "http://127.0.0.1:8080"
func (c Config) PublicAPIURL() string {
	addr := strings.TrimSpace(c.HTTPAddr)
	switch {
	case addr == "":
		return "http://localhost:8080"
	case strings.HasPrefix(addr, ":"):
		return "http://localhost" + addr
	case strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://"):
		return strings.TrimRight(addr, "/")
	default:
		return "http://" + addr
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
