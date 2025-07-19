package config

import (
	"os"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// Config holds the application configuration
type Config struct {
	Database DatabaseConfig `yaml:"database"`
	API      APIConfig      `yaml:"api"`
	UI       UIConfig       `yaml:"ui"`
	Logging  LoggingConfig  `yaml:"logging"`
	
	// Environment variables (from .env)
	BybitAPIKey    string
	BybitSecret    string
	BybitTestnet   bool
	BitvavoAPIKey  string
	BitvavoSecret  string
	BitvavoTestnet bool
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type APIConfig struct {
	Port    int           `yaml:"port"`
	Timeout time.Duration `yaml:"timeout"`
}

type UIConfig struct {
	Port int `yaml:"port"`
}

type LoggingConfig struct {
	Level string `yaml:"level"`
}

// Load loads configuration from environment and YAML file
func Load() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	config := &Config{
		Database: DatabaseConfig{
			Path: getEnvOrDefault("DATABASE_PATH", "./data.db"),
		},
		API: APIConfig{
			Port:    getEnvIntOrDefault("API_PORT", 8080),
			Timeout: time.Duration(getEnvIntOrDefault("API_TIMEOUT_SECONDS", 30)) * time.Second,
		},
		UI: UIConfig{
			Port: getEnvIntOrDefault("UI_PORT", 8081),
		},
		Logging: LoggingConfig{
			Level: getEnvOrDefault("LOG_LEVEL", "info"),
		},
		BybitAPIKey:    os.Getenv("BYBIT_API_KEY"),
		BybitSecret:    os.Getenv("BYBIT_SECRET"),
		BybitTestnet:   getEnvOrDefault("BYBIT_TESTNET", "false") == "true",
		BitvavoAPIKey:  os.Getenv("BITVAVO_API_KEY"),
		BitvavoSecret:  os.Getenv("BITVAVO_SECRET"),
		BitvavoTestnet: getEnvOrDefault("BITVAVO_TESTNET", "false") == "true",
	}

	// Load YAML config if it exists
	if data, err := os.ReadFile("config.yaml"); err == nil {
		if err := yaml.Unmarshal(data, config); err != nil {
			return nil, err
		}
	}

	return config, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := parseIntSafe(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func parseIntSafe(s string) (int, error) {
	var result int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, &parseError{s}
		}
		result = result*10 + int(c-'0')
	}
	return result, nil
}

type parseError struct {
	value string
}

func (e *parseError) Error() string {
	return "invalid integer: " + e.value
}