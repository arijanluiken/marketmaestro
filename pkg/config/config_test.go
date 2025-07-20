package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// Save original environment variables
	originalEnvVars := map[string]string{
		"DATABASE_PATH":       os.Getenv("DATABASE_PATH"),
		"API_PORT":           os.Getenv("API_PORT"),
		"API_TIMEOUT_SECONDS": os.Getenv("API_TIMEOUT_SECONDS"),
		"UI_PORT":            os.Getenv("UI_PORT"),
		"LOG_LEVEL":          os.Getenv("LOG_LEVEL"),
		"BYBIT_API_KEY":      os.Getenv("BYBIT_API_KEY"),
		"BYBIT_SECRET":       os.Getenv("BYBIT_SECRET"),
		"BYBIT_TESTNET":      os.Getenv("BYBIT_TESTNET"),
		"BITVAVO_API_KEY":    os.Getenv("BITVAVO_API_KEY"),
		"BITVAVO_SECRET":     os.Getenv("BITVAVO_SECRET"),
		"BITVAVO_TESTNET":    os.Getenv("BITVAVO_TESTNET"),
	}

	// Clean up after test
	defer func() {
		for key, value := range originalEnvVars {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	t.Run("loads default configuration", func(t *testing.T) {
		// Clear all environment variables
		for key := range originalEnvVars {
			os.Unsetenv(key)
		}

		config, err := Load()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Check default values
		if config.Database.Path != "./mercantile.db" {
			t.Errorf("expected database path './mercantile.db', got '%s'", config.Database.Path)
		}
		if config.API.Port != 8080 {
			t.Errorf("expected API port 8080, got %d", config.API.Port)
		}
		if config.API.Timeout != 30*time.Second {
			t.Errorf("expected API timeout 30s, got %v", config.API.Timeout)
		}
		if config.UI.Port != 8081 {
			t.Errorf("expected UI port 8081, got %d", config.UI.Port)
		}
		if config.Logging.Level != "info" {
			t.Errorf("expected log level 'info', got '%s'", config.Logging.Level)
		}
		if config.Strategies.Directory != "./strategies" {
			t.Errorf("expected strategies directory './strategies', got '%s'", config.Strategies.Directory)
		}
		if config.Strategies.DefaultInterval != "1m" {
			t.Errorf("expected default interval '1m', got '%s'", config.Strategies.DefaultInterval)
		}
		if config.Strategies.MaxConcurrent != 10 {
			t.Errorf("expected max concurrent 10, got %d", config.Strategies.MaxConcurrent)
		}
		if config.Risk.MaxPositionSize != 0.1 {
			t.Errorf("expected max position size 0.1, got %f", config.Risk.MaxPositionSize)
		}
		if config.Risk.MaxDailyLoss != 1000.0 {
			t.Errorf("expected max daily loss 1000.0, got %f", config.Risk.MaxDailyLoss)
		}
		if config.Risk.MaxOpenPositions != 5 {
			t.Errorf("expected max open positions 5, got %d", config.Risk.MaxOpenPositions)
		}
		if config.BybitTestnet != false {
			t.Errorf("expected Bybit testnet false, got %t", config.BybitTestnet)
		}
		if config.BitvavoTestnet != false {
			t.Errorf("expected Bitvavo testnet false, got %t", config.BitvavoTestnet)
		}
	})

	t.Run("loads environment variables", func(t *testing.T) {
		// Set environment variables
		os.Setenv("DATABASE_PATH", "/custom/path.db")
		os.Setenv("API_PORT", "9090")
		os.Setenv("API_TIMEOUT_SECONDS", "45")
		os.Setenv("UI_PORT", "3000")
		os.Setenv("LOG_LEVEL", "debug")
		os.Setenv("BYBIT_API_KEY", "test_key")
		os.Setenv("BYBIT_SECRET", "test_secret")
		os.Setenv("BYBIT_TESTNET", "true")
		os.Setenv("BITVAVO_API_KEY", "bitvavo_key")
		os.Setenv("BITVAVO_SECRET", "bitvavo_secret")
		os.Setenv("BITVAVO_TESTNET", "true")

		config, err := Load()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Check environment variable values
		if config.Database.Path != "/custom/path.db" {
			t.Errorf("expected database path '/custom/path.db', got '%s'", config.Database.Path)
		}
		if config.API.Port != 9090 {
			t.Errorf("expected API port 9090, got %d", config.API.Port)
		}
		if config.API.Timeout != 45*time.Second {
			t.Errorf("expected API timeout 45s, got %v", config.API.Timeout)
		}
		if config.UI.Port != 3000 {
			t.Errorf("expected UI port 3000, got %d", config.UI.Port)
		}
		if config.Logging.Level != "debug" {
			t.Errorf("expected log level 'debug', got '%s'", config.Logging.Level)
		}
		if config.BybitAPIKey != "test_key" {
			t.Errorf("expected Bybit API key 'test_key', got '%s'", config.BybitAPIKey)
		}
		if config.BybitSecret != "test_secret" {
			t.Errorf("expected Bybit secret 'test_secret', got '%s'", config.BybitSecret)
		}
		if config.BybitTestnet != true {
			t.Errorf("expected Bybit testnet true, got %t", config.BybitTestnet)
		}
		if config.BitvavoAPIKey != "bitvavo_key" {
			t.Errorf("expected Bitvavo API key 'bitvavo_key', got '%s'", config.BitvavoAPIKey)
		}
		if config.BitvavoSecret != "bitvavo_secret" {
			t.Errorf("expected Bitvavo secret 'bitvavo_secret', got '%s'", config.BitvavoSecret)
		}
		if config.BitvavoTestnet != true {
			t.Errorf("expected Bitvavo testnet true, got %t", config.BitvavoTestnet)
		}
	})
}

func TestGetEnvOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		expected     string
	}{
		{
			name:         "returns environment value when set",
			key:          "TEST_KEY",
			defaultValue: "default",
			envValue:     "env_value",
			expected:     "env_value",
		},
		{
			name:         "returns default when environment not set",
			key:          "UNSET_KEY",
			defaultValue: "default",
			envValue:     "",
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original value
			original := os.Getenv(tt.key)
			defer func() {
				if original == "" {
					os.Unsetenv(tt.key)
				} else {
					os.Setenv(tt.key, original)
				}
			}()

			// Set test value
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
			} else {
				os.Unsetenv(tt.key)
			}

			result := getEnvOrDefault(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestGetEnvIntOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue int
		envValue     string
		expected     int
	}{
		{
			name:         "returns parsed integer when valid",
			key:          "TEST_INT_KEY",
			defaultValue: 42,
			envValue:     "123",
			expected:     123,
		},
		{
			name:         "returns default when invalid integer",
			key:          "TEST_INT_KEY",
			defaultValue: 42,
			envValue:     "invalid",
			expected:     42,
		},
		{
			name:         "returns default when not set",
			key:          "UNSET_INT_KEY",
			defaultValue: 42,
			envValue:     "",
			expected:     42,
		},
		{
			name:         "returns zero when environment is zero",
			key:          "TEST_INT_KEY",
			defaultValue: 42,
			envValue:     "0",
			expected:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original value
			original := os.Getenv(tt.key)
			defer func() {
				if original == "" {
					os.Unsetenv(tt.key)
				} else {
					os.Setenv(tt.key, original)
				}
			}()

			// Set test value
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
			} else {
				os.Unsetenv(tt.key)
			}

			result := getEnvIntOrDefault(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestParseIntSafe(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    int
		expectError bool
	}{
		{
			name:        "parses valid integer",
			input:       "123",
			expected:    123,
			expectError: false,
		},
		{
			name:        "parses zero",
			input:       "0",
			expected:    0,
			expectError: false,
		},
		{
			name:        "fails on negative number",
			input:       "-123",
			expected:    0,
			expectError: true,
		},
		{
			name:        "fails on floating point",
			input:       "123.45",
			expected:    0,
			expectError: true,
		},
		{
			name:        "fails on letters",
			input:       "abc",
			expected:    0,
			expectError: true,
		},
		{
			name:        "fails on empty string",
			input:       "",
			expected:    0,
			expectError: false, // Empty string results in 0, no error
		},
		{
			name:        "fails on mixed characters",
			input:       "12a3",
			expected:    0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseIntSafe(tt.input)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error for input '%s', got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error for input '%s', got %v", tt.input, err)
				}
				if result != tt.expected {
					t.Errorf("expected %d for input '%s', got %d", tt.expected, tt.input, result)
				}
			}
		})
	}
}

func TestParseError(t *testing.T) {
	err := &parseError{value: "invalid123"}
	expected := "invalid integer: invalid123"
	if err.Error() != expected {
		t.Errorf("expected error message '%s', got '%s'", expected, err.Error())
	}
}