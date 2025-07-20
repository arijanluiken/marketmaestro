package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/arijanluiken/mercantile/pkg/config"
	"github.com/arijanluiken/mercantile/pkg/database"
)

func setupTestAPI(t *testing.T) *APIActor {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"
	
	db, err := database.New(dbPath)
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	cfg := &config.Config{
		API: config.APIConfig{
			Port:    8080,
			Timeout: 30 * time.Second,
		},
	}
	logger := zerolog.New(nil)

	api := New(cfg, logger)
	api.SetDatabase(db.Conn())
	return api
}

func TestWriteJSON(t *testing.T) {
	api := setupTestAPI(t)
	
	// Create test data
	testData := map[string]interface{}{
		"message": "test response",
		"status":  "success",
		"count":   42,
	}

	// Create test response recorder
	w := httptest.NewRecorder()

	// Call writeJSON
	api.writeJSON(w, testData)

	// Check status code
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Check content type
	expectedContentType := "application/json"
	if w.Header().Get("Content-Type") != expectedContentType {
		t.Errorf("expected content type %s, got %s", expectedContentType, w.Header().Get("Content-Type"))
	}

	// Check response body
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["message"] != "test response" {
		t.Errorf("expected message 'test response', got '%v'", response["message"])
	}
	if response["status"] != "success" {
		t.Errorf("expected status 'success', got '%v'", response["status"])
	}

	// JSON numbers are parsed as float64
	if response["count"].(float64) != 42 {
		t.Errorf("expected count 42, got %v", response["count"])
	}
}

func TestWriteError(t *testing.T) {
	api := setupTestAPI(t)
	
	// Create test response recorder
	w := httptest.NewRecorder()

	// Call writeError
	api.writeError(w, "test error message", http.StatusBadRequest)

	// Check status code
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	// Check content type
	expectedContentType := "application/json"
	if w.Header().Get("Content-Type") != expectedContentType {
		t.Errorf("expected content type %s, got %s", expectedContentType, w.Header().Get("Content-Type"))
	}

	// Check response body
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}

	if response["error"] != "test error message" {
		t.Errorf("expected error 'test error message', got '%s'", response["error"])
	}
}

func TestHandleHealth(t *testing.T) {
	api := setupTestAPI(t)
	
	// Create test request
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	// Call handleHealth
	api.handleHealth(w, req)

	// Check status code
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Check content type
	expectedContentType := "application/json"
	if w.Header().Get("Content-Type") != expectedContentType {
		t.Errorf("expected content type %s, got %s", expectedContentType, w.Header().Get("Content-Type"))
	}

	// Check response body
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("failed to unmarshal health response: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("expected status 'ok', got '%v'", response["status"])
	}
	if response["version"] != "1.0.0" {
		t.Errorf("expected version '1.0.0', got '%v'", response["version"])
	}

	// Check timestamp is present and is a string
	if _, ok := response["timestamp"].(string); !ok {
		t.Error("expected timestamp to be a string")
	}

	// Verify timestamp format (should be RFC3339)
	timestampStr := response["timestamp"].(string)
	_, err = time.Parse(time.RFC3339, timestampStr)
	if err != nil {
		t.Errorf("expected timestamp in RFC3339 format, got parse error: %v", err)
	}
}

func TestHandleOpenAPISpec(t *testing.T) {
	api := setupTestAPI(t)
	
	// Create test request
	req := httptest.NewRequest("GET", "/openapi.yaml", nil)
	w := httptest.NewRecorder()

	// Call handleOpenAPISpec
	api.handleOpenAPISpec(w, req)

	// Check status code
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Check content type
	expectedContentType := "application/json"
	if w.Header().Get("Content-Type") != expectedContentType {
		t.Errorf("expected content type %s, got %s", expectedContentType, w.Header().Get("Content-Type"))
	}

	// Check response body structure
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("failed to unmarshal OpenAPI spec response: %v", err)
	}

	if response["openapi"] != "3.0.0" {
		t.Errorf("expected openapi '3.0.0', got '%v'", response["openapi"])
	}

	// Check info section
	info, ok := response["info"].(map[string]interface{})
	if !ok {
		t.Error("expected info to be a map")
	} else {
		if info["title"] != "Mercantile Trading Bot API" {
			t.Errorf("expected title 'Mercantile Trading Bot API', got '%v'", info["title"])
		}
		if info["version"] != "1.0.0" {
			t.Errorf("expected version '1.0.0', got '%v'", info["version"])
		}
		if info["description"] != "API for the Mercantile crypto trading bot" {
			t.Errorf("expected specific description, got '%v'", info["description"])
		}
	}

	// Check servers section exists
	if _, ok := response["servers"]; !ok {
		t.Error("expected servers section to be present")
	}
}

func TestNew(t *testing.T) {
	cfg := &config.Config{
		API: config.APIConfig{
			Port:    8080,
			Timeout: 30 * time.Second,
		},
	}
	logger := zerolog.New(nil)

	api := New(cfg, logger)

	if api == nil {
		t.Error("expected non-nil API actor")
	}
	if api.config == nil {
		t.Error("expected config to be set")
	}
	if api.config.API.Port != 8080 {
		t.Errorf("expected API port 8080, got %d", api.config.API.Port)
	}
	if api.config.API.Timeout != 30*time.Second {
		t.Errorf("expected API timeout 30s, got %v", api.config.API.Timeout)
	}
}