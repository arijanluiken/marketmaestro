package settings

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/arijanluiken/mercantile/pkg/config"
	"github.com/arijanluiken/mercantile/pkg/database"
)

func setupTestDatabase(t *testing.T) *database.DB {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := database.New(dbPath)
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	return db
}

func setupTestActor(t *testing.T) (*SettingsActor, *database.DB) {
	db := setupTestDatabase(t)
	cfg := &config.Config{}
	logger := zerolog.New(nil)

	actor := New("test_exchange", cfg, db, logger)
	return actor, db
}

func TestNew(t *testing.T) {
	cfg := &config.Config{}
	db := setupTestDatabase(t)
	defer db.Close()
	logger := zerolog.New(nil)

	actor := New("bybit", cfg, db, logger)

	if actor == nil {
		t.Error("expected non-nil actor")
	}
	if actor.exchangeName != "bybit" {
		t.Errorf("expected exchange name 'bybit', got '%s'", actor.exchangeName)
	}
	if actor.config != cfg {
		t.Error("expected config to be set")
	}
	if actor.db != db {
		t.Error("expected database to be set")
	}
}

func TestSettingResponse(t *testing.T) {
	response := SettingResponse{
		Key:   "test_key",
		Value: "test_value",
		Found: true,
	}

	if response.Key != "test_key" {
		t.Errorf("expected key 'test_key', got '%s'", response.Key)
	}
	if response.Value != "test_value" {
		t.Errorf("expected value 'test_value', got '%s'", response.Value)
	}
	if !response.Found {
		t.Error("expected Found to be true")
	}
}

func TestMessageStructs(t *testing.T) {
	t.Run("GetSettingMsg", func(t *testing.T) {
		msg := GetSettingMsg{Key: "test_key"}
		if msg.Key != "test_key" {
			t.Errorf("expected key 'test_key', got '%s'", msg.Key)
		}
	})

	t.Run("SetSettingMsg", func(t *testing.T) {
		msg := SetSettingMsg{Key: "test_key", Value: "test_value"}
		if msg.Key != "test_key" {
			t.Errorf("expected key 'test_key', got '%s'", msg.Key)
		}
		if msg.Value != "test_value" {
			t.Errorf("expected value 'test_value', got '%s'", msg.Value)
		}
	})

	t.Run("StatusMsg", func(t *testing.T) {
		msg := StatusMsg{}
		// StatusMsg is empty, just test it can be created
		_ = msg
	})
}

func TestSettingsActorDirectDatabase(t *testing.T) {
	settingsActor, db := setupTestActor(t)
	defer db.Close()

	t.Run("sets and gets setting via database", func(t *testing.T) {
		// Use the proper schema that matches the migration
		query := `INSERT INTO settings (actor_type, actor_id, key, value, exchange, updated_at) VALUES (?, ?, ?, ?, ?, ?)`
		_, err := db.Conn().Exec(query, "settings", "test_exchange", "direct_key", "direct_value", "test_exchange", time.Now())
		if err != nil {
			t.Fatalf("failed to insert test setting: %v", err)
		}

		// Verify we can query it back
		var value string
		err = db.Conn().QueryRow("SELECT value FROM settings WHERE key = ? AND exchange = ?", "direct_key", "test_exchange").Scan(&value)
		if err != nil {
			t.Fatalf("failed to query setting: %v", err)
		}

		if value != "direct_value" {
			t.Errorf("expected value 'direct_value', got '%s'", value)
		}
	})

	t.Run("verifies actor fields are set correctly", func(t *testing.T) {
		if settingsActor.exchangeName != "test_exchange" {
			t.Errorf("expected exchange name 'test_exchange', got '%s'", settingsActor.exchangeName)
		}
		if settingsActor.db == nil {
			t.Error("expected database to be set")
		}
		if settingsActor.config == nil {
			t.Error("expected config to be set")
		}
	})
}

func TestDatabaseOperations(t *testing.T) {
	db := setupTestDatabase(t)
	defer db.Close()

	t.Run("handles database settings table operations", func(t *testing.T) {
		// Test UPSERT behavior using the actual schema
		query := `INSERT OR REPLACE INTO settings (actor_type, actor_id, key, value, exchange, updated_at) VALUES (?, ?, ?, ?, ?, ?)`
		
		// First insert
		_, err := db.Conn().Exec(query, "settings", "test_exchange", "upsert_key", "value1", "test_exchange", time.Now())
		if err != nil {
			t.Fatalf("failed to insert setting: %v", err)
		}

		// Update same key
		_, err = db.Conn().Exec(query, "settings", "test_exchange", "upsert_key", "value2", "test_exchange", time.Now())
		if err != nil {
			t.Fatalf("failed to update setting: %v", err)
		}

		// Verify only one record exists with updated value
		var count int
		var value string
		err = db.Conn().QueryRow("SELECT COUNT(*), value FROM settings WHERE key = ? AND exchange = ?", "upsert_key", "test_exchange").Scan(&count, &value)
		if err != nil {
			t.Fatalf("failed to query setting: %v", err)
		}

		if count != 1 {
			t.Errorf("expected 1 record, got %d", count)
		}
		if value != "value2" {
			t.Errorf("expected value 'value2', got '%s'", value)
		}
	})

	t.Run("handles different exchanges", func(t *testing.T) {
		// Insert settings for different exchanges using proper schema
		query := `INSERT INTO settings (actor_type, actor_id, key, value, exchange, updated_at) VALUES (?, ?, ?, ?, ?, ?)`
		
		_, err := db.Conn().Exec(query, "settings", "bybit", "same_key", "bybit_value", "bybit", time.Now())
		if err != nil {
			t.Fatalf("failed to insert bybit setting: %v", err)
		}

		_, err = db.Conn().Exec(query, "settings", "bitvavo", "same_key", "bitvavo_value", "bitvavo", time.Now())
		if err != nil {
			t.Fatalf("failed to insert bitvavo setting: %v", err)
		}

		// Verify both exist with different values
		var bybitValue, bitvavoValue string
		
		err = db.Conn().QueryRow("SELECT value FROM settings WHERE key = ? AND exchange = ?", "same_key", "bybit").Scan(&bybitValue)
		if err != nil {
			t.Fatalf("failed to query bybit setting: %v", err)
		}

		err = db.Conn().QueryRow("SELECT value FROM settings WHERE key = ? AND exchange = ?", "same_key", "bitvavo").Scan(&bitvavoValue)
		if err != nil {
			t.Fatalf("failed to query bitvavo setting: %v", err)
		}

		if bybitValue != "bybit_value" {
			t.Errorf("expected bybit value 'bybit_value', got '%s'", bybitValue)
		}
		if bitvavoValue != "bitvavo_value" {
			t.Errorf("expected bitvavo value 'bitvavo_value', got '%s'", bitvavoValue)
		}
	})

	t.Run("handles non-existent settings", func(t *testing.T) {
		var value string
		err := db.Conn().QueryRow("SELECT value FROM settings WHERE key = ? AND exchange = ?", "nonexistent", "test_exchange").Scan(&value)
		
		// Should get sql.ErrNoRows
		if err == nil {
			t.Error("expected error for non-existent setting")
		}
		
		// Check for specific "no rows" error
		if err.Error() != "sql: no rows in result set" {
			t.Errorf("expected 'sql: no rows in result set', got '%s'", err.Error())
		}
	})
}