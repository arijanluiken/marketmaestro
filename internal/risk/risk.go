package risk

import (
	"fmt"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/rs/zerolog"

	"github.com/arijanluiken/mercantile/pkg/config"
	"github.com/arijanluiken/mercantile/pkg/database"
)

// Messages for risk manager actor communication
type (
	ValidateOrderMsg struct{}
	UpdateRiskMsg    struct{}
	StatusMsg        struct{}
)

// RiskManagerActor manages risk for trading strategies
type RiskManagerActor struct {
	exchangeName string
	config       *config.Config
	db           *database.DB
	logger       zerolog.Logger
}

// New creates a new risk manager actor
func New(exchangeName string, cfg *config.Config, db *database.DB, logger zerolog.Logger) *RiskManagerActor {
	return &RiskManagerActor{
		exchangeName: exchangeName,
		config:       cfg,
		db:           db,
		logger:       logger,
	}
}

// Receive handles incoming messages
func (r *RiskManagerActor) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Started:
		r.onStarted(ctx)
	case actor.Stopped:
		r.onStopped(ctx)
	case StatusMsg:
		r.onStatus(ctx)
	default:
		r.logger.Debug().
			Str("message_type", fmt.Sprintf("%T", msg)).
			Msg("Received message")
	}
}

func (r *RiskManagerActor) onStarted(ctx *actor.Context) {
	r.logger.Info().
		Str("exchange", r.exchangeName).
		Msg("Risk manager actor started")
}

func (r *RiskManagerActor) onStopped(ctx *actor.Context) {
	r.logger.Info().
		Str("exchange", r.exchangeName).
		Msg("Risk manager actor stopped")
}

func (r *RiskManagerActor) onStatus(ctx *actor.Context) {
	status := map[string]interface{}{
		"exchange":  r.exchangeName,
		"timestamp": time.Now(),
	}
	
	ctx.Respond(status)
}