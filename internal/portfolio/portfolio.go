package portfolio

import (
	"fmt"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/rs/zerolog"

	"github.com/arijanluiken/mercantile/pkg/config"
	"github.com/arijanluiken/mercantile/pkg/database"
)

// Messages for portfolio actor communication
type (
	UpdatePortfolioMsg struct{}
	GetPortfolioMsg    struct{}
	StatusMsg          struct{}
)

// PortfolioActor tracks portfolio and P&L information
type PortfolioActor struct {
	exchangeName string
	config       *config.Config
	db           *database.DB
	logger       zerolog.Logger
}

// New creates a new portfolio actor
func New(exchangeName string, cfg *config.Config, db *database.DB, logger zerolog.Logger) *PortfolioActor {
	return &PortfolioActor{
		exchangeName: exchangeName,
		config:       cfg,
		db:           db,
		logger:       logger,
	}
}

// Receive handles incoming messages
func (p *PortfolioActor) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Started:
		p.onStarted(ctx)
	case actor.Stopped:
		p.onStopped(ctx)
	case StatusMsg:
		p.onStatus(ctx)
	default:
		p.logger.Debug().
			Str("message_type", fmt.Sprintf("%T", msg)).
			Msg("Received message")
	}
}

func (p *PortfolioActor) onStarted(ctx *actor.Context) {
	p.logger.Info().
		Str("exchange", p.exchangeName).
		Msg("Portfolio actor started")
}

func (p *PortfolioActor) onStopped(ctx *actor.Context) {
	p.logger.Info().
		Str("exchange", p.exchangeName).
		Msg("Portfolio actor stopped")
}

func (p *PortfolioActor) onStatus(ctx *actor.Context) {
	status := map[string]interface{}{
		"exchange":  p.exchangeName,
		"timestamp": time.Now(),
	}
	
	ctx.Respond(status)
}