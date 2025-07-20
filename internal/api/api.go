package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"

	"github.com/arijanluiken/mercantile/internal/exchange"
	"github.com/arijanluiken/mercantile/pkg/config"
)

// Messages for API actor communication
type (
	StartServerMsg       struct{}
	StopServerMsg        struct{}
	StatusMsg            struct{}
	SetPortfolioActorMsg struct {
		Exchange     string
		PortfolioPID *actor.PID
	}
	SetExchangeActorMsg struct {
		Exchange    string
		ExchangePID *actor.PID
	}
)

// APIActor provides REST API and WebSocket endpoints
type APIActor struct {
	config          *config.Config
	logger          zerolog.Logger
	server          *http.Server
	router          chi.Router
	wsUpgrader      websocket.Upgrader
	supervisorPID   *actor.PID
	portfolioPIDs   map[string]*actor.PID               // exchange name -> portfolio PID
	exchangePIDs    map[string]*actor.PID               // exchange name -> exchange PID
	strategiesCache map[string][]map[string]interface{} // exchange name -> strategies
	portfolioCache  map[string]map[string]interface{}   // exchange name -> portfolio data
	ordersCache     map[string][]map[string]interface{} // exchange name -> orders
	db              *sql.DB                             // database connection
}

// New creates a new API actor
func New(cfg *config.Config, logger zerolog.Logger) *APIActor {
	return &APIActor{
		config:          cfg,
		logger:          logger,
		portfolioPIDs:   make(map[string]*actor.PID),
		exchangePIDs:    make(map[string]*actor.PID),
		strategiesCache: make(map[string][]map[string]interface{}),
		portfolioCache:  make(map[string]map[string]interface{}),
		ordersCache:     make(map[string][]map[string]interface{}),
		wsUpgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins for development
				return true
			},
		},
	}
}

// SetSupervisorPID sets the supervisor actor PID for communication
func (a *APIActor) SetSupervisorPID(pid *actor.PID) {
	a.supervisorPID = pid
}

// SetDatabase sets the database connection for the API actor
func (a *APIActor) SetDatabase(db *sql.DB) {
	a.db = db
}

// Receive handles incoming messages
func (a *APIActor) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Started:
		a.onStarted(ctx)
	case actor.Stopped:
		a.onStopped(ctx)
	case StartServerMsg:
		a.onStartServer(ctx)
	case StopServerMsg:
		a.onStopServer(ctx)
	case StatusMsg:
		a.onStatus(ctx)
	case SetPortfolioActorMsg:
		a.onSetPortfolioActor(ctx, msg)
	case SetExchangeActorMsg:
		a.onSetExchangeActor(ctx, msg)
	case exchange.StrategyDataUpdateMsg:
		a.onStrategyDataUpdate(ctx, msg)
	case exchange.PortfolioDataUpdateMsg:
		a.onPortfolioDataUpdate(ctx, msg)
	case exchange.OrdersDataUpdateMsg:
		a.onOrdersDataUpdate(ctx, msg)
	default:
		a.logger.Debug().
			Str("message_type", fmt.Sprintf("%T", msg)).
			Msg("Received message")
	}
}

func (a *APIActor) onStarted(ctx *actor.Context) {
	a.logger.Info().Msg("API actor started")

	// Get supervisor PID from parent
	if ctx.Parent() != nil {
		a.supervisorPID = ctx.Parent()
	}

	// Auto-start the server
	ctx.Send(ctx.PID(), StartServerMsg{})

	// Start periodic strategy data refresh
	go a.startStrategyDataRefresh(ctx)
}

func (a *APIActor) onStopped(ctx *actor.Context) {
	a.logger.Info().Msg("API actor stopped")

	if a.server != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		a.server.Shutdown(shutdownCtx)
	}
}

func (a *APIActor) onStartServer(ctx *actor.Context) {
	a.logger.Info().Int("port", a.config.API.Port).Msg("Starting API server")

	a.setupRouter(ctx)

	a.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", a.config.API.Port),
		Handler:      a.router,
		ReadTimeout:  a.config.API.Timeout,
		WriteTimeout: a.config.API.Timeout,
	}

	go func() {
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.logger.Error().Err(err).Msg("API server error")
		}
	}()

	a.logger.Info().Msg("API server started successfully")
}

func (a *APIActor) onStopServer(ctx *actor.Context) {
	if a.server == nil {
		return
	}

	a.logger.Info().Msg("Stopping API server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := a.server.Shutdown(shutdownCtx); err != nil {
		a.logger.Error().Err(err).Msg("Error stopping API server")
	} else {
		a.logger.Info().Msg("API server stopped successfully")
	}
}

func (a *APIActor) onStatus(ctx *actor.Context) {
	status := map[string]interface{}{
		"server_running":  a.server != nil,
		"port":            a.config.API.Port,
		"timestamp":       time.Now(),
		"portfolio_count": len(a.portfolioPIDs),
	}

	ctx.Respond(status)
}

func (a *APIActor) onSetPortfolioActor(ctx *actor.Context, msg SetPortfolioActorMsg) {
	a.portfolioPIDs[msg.Exchange] = msg.PortfolioPID
	a.logger.Info().
		Str("exchange", msg.Exchange).
		Msg("Portfolio actor reference set")
}

func (a *APIActor) onSetExchangeActor(ctx *actor.Context, msg SetExchangeActorMsg) {
	a.exchangePIDs[msg.Exchange] = msg.ExchangePID
	a.logger.Info().
		Str("exchange", msg.Exchange).
		Msg("Exchange actor reference set")
}

func (a *APIActor) onStrategyDataUpdate(ctx *actor.Context, msg exchange.StrategyDataUpdateMsg) {
	a.strategiesCache[msg.Exchange] = msg.Strategies
	a.logger.Debug().
		Str("exchange", msg.Exchange).
		Int("strategy_count", len(msg.Strategies)).
		Msg("Strategy data cache updated")
}

func (a *APIActor) onPortfolioDataUpdate(ctx *actor.Context, msg exchange.PortfolioDataUpdateMsg) {
	// Get existing portfolio data for this exchange, or create new if it doesn't exist
	existingData, exists := a.portfolioCache[msg.Exchange]
	if !exists {
		existingData = map[string]interface{}{
			"balances":  []map[string]interface{}{},
			"positions": []map[string]interface{}{},
			"exchange":  msg.Exchange,
			"updated":   time.Now().Unix(),
		}
	}

	// Update balances if provided (don't overwrite with empty)
	if len(msg.Balances) > 0 {
		existingData["balances"] = msg.Balances
		a.logger.Debug().
			Str("exchange", msg.Exchange).
			Int("new_balance_count", len(msg.Balances)).
			Msg("Updated balances in portfolio cache")
	}

	// Update positions if provided (don't overwrite with empty)
	if len(msg.Positions) > 0 {
		existingData["positions"] = msg.Positions
		a.logger.Debug().
			Str("exchange", msg.Exchange).
			Int("new_position_count", len(msg.Positions)).
			Msg("Updated positions in portfolio cache")
	}

	// Always update timestamp
	existingData["updated"] = time.Now().Unix()

	// Store back the merged data
	a.portfolioCache[msg.Exchange] = existingData

	// Get final counts for logging
	finalBalances, _ := existingData["balances"].([]map[string]interface{})
	finalPositions, _ := existingData["positions"].([]map[string]interface{})

	a.logger.Debug().
		Str("exchange", msg.Exchange).
		Int("final_balance_count", len(finalBalances)).
		Int("final_position_count", len(finalPositions)).
		Int("incoming_balance_count", len(msg.Balances)).
		Int("incoming_position_count", len(msg.Positions)).
		Msg("Portfolio data cache updated")
}

func (a *APIActor) onOrdersDataUpdate(ctx *actor.Context, msg exchange.OrdersDataUpdateMsg) {
	a.ordersCache[msg.Exchange] = msg.Orders
	a.logger.Debug().
		Str("exchange", msg.Exchange).
		Int("order_count", len(msg.Orders)).
		Msg("Orders data cache updated")
}

func (a *APIActor) startStrategyDataRefresh(ctx *actor.Context) {
	ticker := time.NewTicker(30 * time.Second) // Refresh every 30 seconds
	defer ticker.Stop()

	// Initial refresh after 5 seconds to let exchanges start up
	time.Sleep(5 * time.Second)
	a.refreshLiveData(ctx)

	for range ticker.C {
		a.refreshLiveData(ctx)
	}
}

func (a *APIActor) refreshLiveData(ctx *actor.Context) {
	for exchangeName, exchangePID := range a.exchangePIDs {
		// Request strategies data
		ctx.Send(exchangePID, exchange.GetStrategiesMsg{})

		// Request balances and positions data
		ctx.Send(exchangePID, exchange.GetBalancesMsg{})
		ctx.Send(exchangePID, exchange.GetPositionsMsg{})

		a.logger.Debug().
			Str("exchange", exchangeName).
			Msg("Triggered periodic data refresh (strategies, balances, positions)")
	}
}

func (a *APIActor) setupRouter(ctx *actor.Context) {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.Timeout(a.config.API.Timeout))

	// CORS for development
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token")

			if r.Method == "OPTIONS" {
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	// Routes
	r.Route("/api/v1", func(r chi.Router) {
		// Health check
		r.Get("/health", a.handleHealth)

		// OpenAPI spec
		r.Get("/openapi.json", a.handleOpenAPISpec)

		// Exchange routes
		r.Route("/exchanges", func(r chi.Router) {
			r.Get("/", a.handleGetExchanges(ctx))
			r.Get("/{exchange}/status", a.handleGetExchangeStatus(ctx))
			r.Get("/{exchange}/balances", a.handleGetBalances(ctx))
			r.Get("/{exchange}/positions", a.handleGetPositions(ctx))
		})

		// Strategy routes
		r.Route("/strategies", func(r chi.Router) {
			r.Get("/", a.handleGetStrategies(ctx))
			r.Post("/", a.handleCreateStrategy(ctx))
			r.Get("/{id}", a.handleGetStrategyStatus(ctx))
			r.Get("/{id}/status", a.handleGetStrategyStatus(ctx))
			r.Post("/{id}/start", a.handleStartStrategy(ctx))
			r.Post("/{id}/stop", a.handleStopStrategy(ctx))
		})

		// Order routes
		r.Route("/orders", func(r chi.Router) {
			r.Get("/", a.handleGetOrders(ctx))
			r.Post("/", a.handlePlaceOrder(ctx))
			r.Delete("/{id}", a.handleCancelOrder(ctx))
		})

		// Portfolio routes
		r.Route("/portfolio", func(r chi.Router) {
			r.Get("/", a.handleGetPortfolio(ctx))
			r.Get("/performance", a.handleGetPerformance(ctx))
		})

		// Risk management routes
		r.Route("/risk", func(r chi.Router) {
			r.Get("/parameters", a.handleGetRiskParameters(ctx))
			r.Post("/parameters", a.handleSetRiskParameter(ctx))
			r.Get("/parameters/{parameter}", a.handleGetRiskParameter(ctx))
			r.Get("/metrics", a.handleGetRiskMetrics(ctx))
		})

		// Rebalancing routes
		r.Route("/rebalance", func(r chi.Router) {
			r.Get("/status", a.handleGetRebalanceStatus(ctx))
			r.Post("/start", a.handleStartRebalancing(ctx))
			r.Post("/stop", a.handleStopRebalancing(ctx))
			r.Post("/trigger", a.handleTriggerRebalance(ctx))
			r.Post("/load-script", a.handleLoadRebalanceScript(ctx))
		})
	})

	// WebSocket endpoint
	r.HandleFunc("/ws", a.handleWebSocket(ctx))

	a.router = r
}

// Response helpers
