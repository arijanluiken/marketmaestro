package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"

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
)

// APIActor provides REST API and WebSocket endpoints
type APIActor struct {
	config        *config.Config
	logger        zerolog.Logger
	server        *http.Server
	router        chi.Router
	wsUpgrader    websocket.Upgrader
	supervisorPID *actor.PID
	portfolioPIDs map[string]*actor.PID // exchange name -> portfolio PID
}

// New creates a new API actor
func New(cfg *config.Config, logger zerolog.Logger) *APIActor {
	return &APIActor{
		config:        cfg,
		logger:        logger,
		portfolioPIDs: make(map[string]*actor.PID),
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
	})

	// WebSocket endpoint
	r.HandleFunc("/ws", a.handleWebSocket(ctx))

	a.router = r
}

// Response helpers
func (a *APIActor) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (a *APIActor) writeError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// Basic handlers
func (a *APIActor) handleHealth(w http.ResponseWriter, r *http.Request) {
	a.writeJSON(w, map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
		"version":   "1.0.0",
	})
}

func (a *APIActor) handleOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	spec := map[string]interface{}{
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title":       "Mercantile Trading Bot API",
			"version":     "1.0.0",
			"description": "API for the Mercantile crypto trading bot",
		},
		"servers": []map[string]interface{}{
			{
				"url":         fmt.Sprintf("http://localhost:%d/api/v1", a.config.API.Port),
				"description": "Local server",
			},
		},
		"paths": map[string]interface{}{
			"/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "Health check",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Server is healthy",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"status":    map[string]string{"type": "string"},
											"timestamp": map[string]string{"type": "string"},
										},
									},
								},
							},
						},
					},
				},
			},
			"/exchanges": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "List all exchanges",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "List of exchanges",
						},
					},
				},
			},
			"/strategies": map[string]interface{}{
				"get": map[string]interface{}{
					"summary": "List all strategies",
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "List of strategies",
						},
					},
				},
			},
		},
	}

	a.writeJSON(w, spec)
}

// Handler generators that capture context
func (a *APIActor) handleGetExchanges(ctx *actor.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Get actual exchange status from supervisor
		exchanges := []map[string]interface{}{
			{
				"name":    "bybit",
				"status":  "connected",
				"enabled": true,
				"testnet": true,
			},
		}
		a.writeJSON(w, map[string]interface{}{"exchanges": exchanges})
	}
}

func (a *APIActor) handleGetExchangeStatus(ctx *actor.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		exchangeName := chi.URLParam(r, "exchange")
		// TODO: Get actual status from exchange actor
		a.writeJSON(w, map[string]interface{}{
			"exchange": exchangeName,
			"status":   "connected",
			"uptime":   "5m",
		})
	}
}

func (a *APIActor) handleGetBalances(ctx *actor.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		exchangeName := chi.URLParam(r, "exchange")
		// TODO: Get actual balances from exchange actor
		balances := []map[string]interface{}{
			{
				"asset":     "BTC",
				"available": 1.0,
				"locked":    0.0,
				"total":     1.0,
			},
			{
				"asset":     "USDT",
				"available": 50000.0,
				"locked":    0.0,
				"total":     50000.0,
			},
		}
		a.writeJSON(w, map[string]interface{}{
			"exchange": exchangeName,
			"balances": balances,
		})
	}
}

func (a *APIActor) handleGetPositions(ctx *actor.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		exchangeName := chi.URLParam(r, "exchange")

		// Check if portfolio is connected for this exchange
		_, exists := a.portfolioPIDs[exchangeName]
		if !exists {
			a.writeJSON(w, map[string]interface{}{
				"error":     "Exchange portfolio not found",
				"exchange":  exchangeName,
				"positions": []interface{}{},
			})
			return
		}

		// For now, return sample data to show portfolio integration is working
		// TODO: Implement proper async request-response pattern
		a.writeJSON(w, map[string]interface{}{
			"exchange": exchangeName,
			"status":   "portfolio_connected",
			"positions": []map[string]interface{}{
				{
					"symbol":         "BTCUSDT",
					"quantity":       0.5,
					"average_price":  45000.0,
					"current_price":  46000.0,
					"unrealized_pnl": 500.0,
					"side":           "long",
				},
			},
		})
	}
}

func (a *APIActor) handleGetStrategies(ctx *actor.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Get actual strategies from exchange actors
		strategies := []map[string]interface{}{
			{
				"id":       "bybit:BTCUSDT:simple_sma",
				"name":     "simple_sma",
				"symbol":   "BTCUSDT",
				"exchange": "bybit",
				"status":   "running",
				"pnl":      "+$125.50",
			},
		}
		a.writeJSON(w, map[string]interface{}{"strategies": strategies})
	}
}

func (a *APIActor) handleCreateStrategy(ctx *actor.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement strategy creation
		a.writeJSON(w, map[string]interface{}{
			"status":  "created",
			"message": "Strategy creation not yet implemented",
		})
	}
}

func (a *APIActor) handleGetStrategyStatus(ctx *actor.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		strategyID := chi.URLParam(r, "id")
		// TODO: Get actual strategy status
		a.writeJSON(w, map[string]interface{}{
			"id":     strategyID,
			"status": "running",
		})
	}
}

func (a *APIActor) handleStartStrategy(ctx *actor.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		strategyID := chi.URLParam(r, "id")
		// TODO: Send start message to strategy actor
		a.writeJSON(w, map[string]interface{}{
			"id":     strategyID,
			"status": "started",
		})
	}
}

func (a *APIActor) handleStopStrategy(ctx *actor.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		strategyID := chi.URLParam(r, "id")
		// TODO: Send stop message to strategy actor
		a.writeJSON(w, map[string]interface{}{
			"id":     strategyID,
			"status": "stopped",
		})
	}
}

func (a *APIActor) handleGetOrders(ctx *actor.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Get actual orders from order manager
		orders := []map[string]interface{}{
			{
				"id":       "order_123",
				"symbol":   "BTCUSDT",
				"side":     "buy",
				"quantity": 0.01,
				"price":    50000.0,
				"status":   "filled",
			},
		}
		a.writeJSON(w, map[string]interface{}{"orders": orders})
	}
}

func (a *APIActor) handlePlaceOrder(ctx *actor.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement order placement
		a.writeJSON(w, map[string]interface{}{
			"status":  "placed",
			"message": "Order placement not yet implemented",
		})
	}
}

func (a *APIActor) handleCancelOrder(ctx *actor.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orderID := chi.URLParam(r, "id")
		// TODO: Send cancel message to order manager
		a.writeJSON(w, map[string]interface{}{
			"id":     orderID,
			"status": "cancelled",
		})
	}
}

func (a *APIActor) handleGetPortfolio(ctx *actor.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if we have any portfolio connections
		portfolioCount := len(a.portfolioPIDs)

		// Enhanced portfolio data with connection status
		a.writeJSON(w, map[string]interface{}{
			"total_value":         51250.50,
			"available_cash":      1250.50,
			"unrealized_pnl":      125.50,
			"realized_pnl":        500.25,
			"connected_exchanges": portfolioCount,
			"status":              "connected",
			"positions": []map[string]interface{}{
				{
					"symbol":         "BTCUSDT",
					"quantity":       0.5,
					"average_price":  45000.0,
					"current_price":  46000.0,
					"unrealized_pnl": 500.0,
					"exchange":       "bybit",
				},
			},
		})
	}
}

func (a *APIActor) handleGetPerformance(ctx *actor.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Get actual performance data
		a.writeJSON(w, map[string]interface{}{
			"daily_pnl":   125.50,
			"weekly_pnl":  625.75,
			"monthly_pnl": 2150.25,
			"total_pnl":   5000.00,
		})
	}
}

func (a *APIActor) handleWebSocket(ctx *actor.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := a.wsUpgrader.Upgrade(w, r, nil)
		if err != nil {
			a.logger.Error().Err(err).Msg("WebSocket upgrade failed")
			return
		}
		defer conn.Close()

		a.logger.Info().Str("remote", r.RemoteAddr).Msg("WebSocket connection established")

		// Handle WebSocket connection
		for {
			messageType, p, err := conn.ReadMessage()
			if err != nil {
				a.logger.Debug().Err(err).Msg("WebSocket read error")
				break
			}

			// Echo back for now (TODO: implement real-time updates)
			if err := conn.WriteMessage(messageType, p); err != nil {
				a.logger.Debug().Err(err).Msg("WebSocket write error")
				break
			}
		}

		a.logger.Info().Str("remote", r.RemoteAddr).Msg("WebSocket connection closed")
	}
}
