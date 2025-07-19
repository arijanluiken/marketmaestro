package settings

import (
	"fmt"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/rs/zerolog"

	"github.com/arijanluiken/mercantile/pkg/config"
	"github.com/arijanluiken/mercantile/pkg/database"
)

// Messages for settings actor communication
type (
	GetSettingMsg struct{ Key string }
	SetSettingMsg struct{ Key, Value string }
	StatusMsg     struct{}

	// Response message for get operations
	SettingResponse struct {
		Key   string
		Value string
		Found bool
	}
)

// SettingsActor manages persistent configuration settings
type SettingsActor struct {
	exchangeName string
	config       *config.Config
	db           *database.DB
	logger       zerolog.Logger
}

// New creates a new settings actor
func New(exchangeName string, cfg *config.Config, db *database.DB, logger zerolog.Logger) *SettingsActor {
	return &SettingsActor{
		exchangeName: exchangeName,
		config:       cfg,
		db:           db,
		logger:       logger,
	}
}

// Receive handles incoming messages
func (s *SettingsActor) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Started:
		s.onStarted(ctx)
	case actor.Stopped:
		s.onStopped(ctx)
	case GetSettingMsg:
		s.onGetSetting(ctx, msg)
	case SetSettingMsg:
		s.onSetSetting(ctx, msg)
	case StatusMsg:
		s.onStatus(ctx)
	default:
		s.logger.Debug().
			Str("message_type", fmt.Sprintf("%T", msg)).
			Msg("Received message")
	}
}

func (s *SettingsActor) onStarted(ctx *actor.Context) {
	s.logger.Info().
		Str("exchange", s.exchangeName).
		Msg("Settings actor started")
}

func (s *SettingsActor) onStopped(ctx *actor.Context) {
	s.logger.Info().
		Str("exchange", s.exchangeName).
		Msg("Settings actor stopped")
}

func (s *SettingsActor) onStatus(ctx *actor.Context) {
	status := map[string]interface{}{
		"exchange":  s.exchangeName,
		"timestamp": time.Now(),
	}

	ctx.Respond(status)
}

func (s *SettingsActor) onGetSetting(ctx *actor.Context, msg GetSettingMsg) {
	// Try to get setting from database first, fallback to config
	query := `SELECT value FROM settings WHERE key = ? AND exchange = ?`
	var value string
	err := s.db.Conn().QueryRow(query, msg.Key, s.exchangeName).Scan(&value)

	if err == nil {
		// Found in database
		ctx.Respond(SettingResponse{
			Key:   msg.Key,
			Value: value,
			Found: true,
		})
		return
	}

	// Not found or error - check if it's a database error or just not found
	if err.Error() != "no rows in result set" && err.Error() != "sql: no rows in result set" {
		s.logger.Error().Err(err).Str("key", msg.Key).Msg("Error querying setting")
	}

	ctx.Respond(SettingResponse{
		Key:   msg.Key,
		Value: "",
		Found: false,
	})
}

func (s *SettingsActor) onSetSetting(ctx *actor.Context, msg SetSettingMsg) {
	// Store setting in database
	query := `INSERT OR REPLACE INTO settings (key, value, exchange, updated_at) VALUES (?, ?, ?, ?)`
	_, err := s.db.Conn().Exec(query, msg.Key, msg.Value, s.exchangeName, time.Now())

	if err != nil {
		s.logger.Error().Err(err).
			Str("key", msg.Key).
			Str("value", msg.Value).
			Msg("Failed to store setting")
		ctx.Respond(fmt.Errorf("failed to store setting: %w", err))
		return
	}

	s.logger.Info().
		Str("key", msg.Key).
		Str("value", msg.Value).
		Msg("Setting stored successfully")

	ctx.Respond("OK")
}
