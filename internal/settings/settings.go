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