package supervisor

import (
	"context"
	"fmt"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/arijanluiken/mercantile/internal/api"
	"github.com/arijanluiken/mercantile/internal/exchange"
	"github.com/arijanluiken/mercantile/internal/ui"
	"github.com/arijanluiken/mercantile/pkg/config"
	"github.com/arijanluiken/mercantile/pkg/database"
)

// Messages for supervisor actor communication
type (
	StartMessage     struct{}
	StopMessage      struct{}
	StatusMessage    struct{}
	ErrorMessage     struct{ Error error }
	RegisterExchange struct {
		Name   string
		Config map[string]interface{}
	}
)

// Supervisor manages all other actors in the system
type Supervisor struct {
	config         *config.Config
	logger         zerolog.Logger
	exchangeActors map[string]*actor.PID
	apiActor       *actor.PID
	uiActor        *actor.PID
	db             *database.DB
}

// New creates a new supervisor actor
func New() *Supervisor {
	return &Supervisor{
		logger:         log.With().Str("actor", "supervisor").Logger(),
		exchangeActors: make(map[string]*actor.PID),
	}
}

// Start initializes and starts the supervisor actor system
func (s *Supervisor) Start(ctx context.Context) error {
	s.logger.Info().Msg("Starting supervisor actor system")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	s.config = cfg

	// Initialize database
	db, err := database.New(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	s.db = db

	// Create actor engine
	config := actor.NewEngineConfig()
	engine, err := actor.NewEngine(config)
	if err != nil {
		return fmt.Errorf("failed to create actor engine: %w", err)
	}

	// Spawn supervisor actor
	supervisorPID := engine.Spawn(func() actor.Receiver {
		return s
	}, "supervisor")

	// Send start message to supervisor
	engine.Send(supervisorPID, StartMessage{})

	s.logger.Info().Msg("Supervisor actor system started successfully")
	return nil
}

// Receive handles incoming messages
func (s *Supervisor) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Started:
		s.onStarted(ctx)
	case actor.Stopped:
		s.onStopped(ctx)
	case actor.Initialized:
		s.onInitialized(ctx)
	case StartMessage:
		s.onStart(ctx)
	case StopMessage:
		s.onStop(ctx)
	case StatusMessage:
		s.onStatus(ctx)
	case ErrorMessage:
		s.onError(ctx, msg)
	case RegisterExchange:
		s.onRegisterExchange(ctx, msg)
	case exchange.PortfolioActorCreatedMsg:
		s.onPortfolioActorCreated(ctx, msg)
	default:
		s.logger.Warn().
			Str("message_type", fmt.Sprintf("%T", msg)).
			Msg("Received unknown message")
	}
}

func (s *Supervisor) onStarted(ctx *actor.Context) {
	s.logger.Info().Msg("Supervisor actor started")
}

func (s *Supervisor) onStopped(ctx *actor.Context) {
	s.logger.Info().Msg("Supervisor actor stopped")
	if s.db != nil {
		s.db.Close()
	}
}

func (s *Supervisor) onInitialized(ctx *actor.Context) {
	s.logger.Debug().Msg("Supervisor actor initialized")
}

func (s *Supervisor) onStart(ctx *actor.Context) {
	s.logger.Info().Msg("Starting child actors")

	// Start API actor
	apiActorPID := ctx.SpawnChild(func() actor.Receiver {
		return api.New(s.config, s.logger.With().Str("actor", "api").Logger())
	}, "api")
	s.apiActor = apiActorPID

	// Start UI actor
	uiActorPID := ctx.SpawnChild(func() actor.Receiver {
		return ui.New(s.config, s.logger.With().Str("actor", "ui").Logger())
	}, "ui")
	s.uiActor = uiActorPID

	// Start exchange actors based on configuration
	for exchangeName, exchangeConfig := range s.config.Exchanges {
		if !exchangeConfig.Enabled {
			s.logger.Info().Str("exchange", exchangeName).Msg("Exchange disabled in configuration")
			continue
		}

		// Check if we have API credentials for this exchange
		var hasCredentials bool
		switch exchangeName {
		case "bybit":
			hasCredentials = s.config.BybitAPIKey != "" && s.config.BybitSecret != ""
		case "bitvavo":
			hasCredentials = s.config.BitvavoAPIKey != "" && s.config.BitvavoSecret != ""
		}

		if hasCredentials {
			s.startExchangeActor(ctx, exchangeName, map[string]interface{}{
				"enabled": exchangeConfig.Enabled,
			})
		} else {
			s.logger.Warn().Str("exchange", exchangeName).Msg("Exchange enabled but missing API credentials")
		}
	}
}

func (s *Supervisor) onStop(ctx *actor.Context) {
	s.logger.Info().Msg("Stopping child actors")

	// Stop all exchange actors
	for name, pid := range s.exchangeActors {
		s.logger.Info().Str("exchange", name).Msg("Stopping exchange actor")
		ctx.Engine().Stop(pid)
	}

	// Stop API actor
	if s.apiActor != nil {
		ctx.Engine().Stop(s.apiActor)
	}

	// Stop UI actor
	if s.uiActor != nil {
		ctx.Engine().Stop(s.uiActor)
	}
}

func (s *Supervisor) onStatus(ctx *actor.Context) {
	status := map[string]interface{}{
		"timestamp":       time.Now(),
		"exchange_actors": len(s.exchangeActors),
		"api_actor_alive": s.apiActor != nil,
		"ui_actor_alive":  s.uiActor != nil,
	}

	s.logger.Info().Interface("status", status).Msg("Supervisor status")
	ctx.Respond(status)
}

func (s *Supervisor) onError(ctx *actor.Context, msg ErrorMessage) {
	s.logger.Error().Err(msg.Error).Msg("Received error from child actor")
}

func (s *Supervisor) onRegisterExchange(ctx *actor.Context, msg RegisterExchange) {
	s.startExchangeActor(ctx, msg.Name, msg.Config)
}

func (s *Supervisor) startExchangeActor(ctx *actor.Context, exchangeName string, config map[string]interface{}) {
	s.logger.Info().Str("exchange", exchangeName).Msg("Starting exchange actor")

	exchangeActorPID := ctx.SpawnChild(func() actor.Receiver {
		return exchange.New(
			exchangeName,
			config,
			s.config,
			s.db,
			s.logger.With().Str("actor", "exchange").Str("exchange", exchangeName).Logger(),
		)
	}, "exchange_"+exchangeName)

	s.exchangeActors[exchangeName] = exchangeActorPID
	s.logger.Info().Str("exchange", exchangeName).Msg("Exchange actor started successfully")
}

func (s *Supervisor) onPortfolioActorCreated(ctx *actor.Context, msg exchange.PortfolioActorCreatedMsg) {
	s.logger.Info().
		Str("exchange", msg.Exchange).
		Msg("Portfolio actor created, notifying API actor")

	// Notify API actor about the new portfolio actor
	if s.apiActor != nil {
		ctx.Send(s.apiActor, api.SetPortfolioActorMsg{
			Exchange:     msg.Exchange,
			PortfolioPID: msg.PortfolioPID,
		})
	}
}
