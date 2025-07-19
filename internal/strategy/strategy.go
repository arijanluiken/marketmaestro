package strategy

import (
	"fmt"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/rs/zerolog"

	"github.com/arijanluiken/mercantile/pkg/config"
	"github.com/arijanluiken/mercantile/pkg/database"
)

// Messages for strategy actor communication
type (
	StartStrategyMsg struct{}
	StopStrategyMsg  struct{}
	KlineDataMsg     struct{}
	OrderBookDataMsg struct{}
	StatusMsg        struct{}
)

// StrategyActor executes trading strategies using Starlark
type StrategyActor struct {
	strategyName string
	symbol       string
	exchangeName string
	config       map[string]interface{}
	appConfig    *config.Config
	db           *database.DB
	logger       zerolog.Logger
	running      bool
}

// New creates a new strategy actor
func New(strategyName, symbol, exchangeName string, strategyConfig map[string]interface{}, appConfig *config.Config, db *database.DB, logger zerolog.Logger) *StrategyActor {
	return &StrategyActor{
		strategyName: strategyName,
		symbol:       symbol,
		exchangeName: exchangeName,
		config:       strategyConfig,
		appConfig:    appConfig,
		db:           db,
		logger:       logger,
	}
}

// Receive handles incoming messages
func (s *StrategyActor) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Started:
		s.onStarted(ctx)
	case actor.Stopped:
		s.onStopped(ctx)
	case StartStrategyMsg:
		s.onStartStrategy(ctx)
	case StopStrategyMsg:
		s.onStopStrategy(ctx)
	case StatusMsg:
		s.onStatus(ctx)
	default:
		s.logger.Debug().
			Str("message_type", fmt.Sprintf("%T", msg)).
			Msg("Received message")
	}
}

func (s *StrategyActor) onStarted(ctx *actor.Context) {
	s.logger.Info().
		Str("strategy", s.strategyName).
		Str("symbol", s.symbol).
		Msg("Strategy actor started")
	
	// Auto-start the strategy
	ctx.Send(ctx.PID(), StartStrategyMsg{})
}

func (s *StrategyActor) onStopped(ctx *actor.Context) {
	s.logger.Info().
		Str("strategy", s.strategyName).
		Str("symbol", s.symbol).
		Msg("Strategy actor stopped")
}

func (s *StrategyActor) onStartStrategy(ctx *actor.Context) {
	s.logger.Info().
		Str("strategy", s.strategyName).
		Str("symbol", s.symbol).
		Msg("Starting strategy execution")
	
	s.running = true
	
	// TODO: Implement Starlark strategy execution
	// This would load and execute the strategy script
}

func (s *StrategyActor) onStopStrategy(ctx *actor.Context) {
	s.logger.Info().
		Str("strategy", s.strategyName).
		Str("symbol", s.symbol).
		Msg("Stopping strategy execution")
	
	s.running = false
}

func (s *StrategyActor) onStatus(ctx *actor.Context) {
	status := map[string]interface{}{
		"strategy_name": s.strategyName,
		"symbol":        s.symbol,
		"exchange":      s.exchangeName,
		"running":       s.running,
		"timestamp":     time.Now(),
	}
	
	ctx.Respond(status)
}