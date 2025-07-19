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

		// Check if exchange is connected
		_, exists := a.exchangePIDs[exchangeName]
		if !exists {
			a.writeJSON(w, map[string]interface{}{
				"error":    "Exchange not found",
				"exchange": exchangeName,
				"balances": []interface{}{},
			})
			return
		}

		// Get cached portfolio data for this exchange
		portfolioData, hasData := a.portfolioCache[exchangeName]
		if hasData {
			balances, hasBalances := portfolioData["balances"].([]map[string]interface{})
			if hasBalances {
				a.writeJSON(w, map[string]interface{}{
					"exchange": exchangeName,
					"status":   "live_data",
					"balances": balances,
				})
				return
			}
		}

		// If no cached data, trigger refresh and return loading state
		if exchangePID, exists := a.exchangePIDs[exchangeName]; exists {
			ctx.Send(exchangePID, exchange.GetBalancesMsg{})
			a.logger.Debug().
				Str("exchange", exchangeName).
				Msg("Triggered balance refresh")
		}

		a.writeJSON(w, map[string]interface{}{
			"exchange": exchangeName,
			"status":   "loading",
			"balances": []interface{}{},
			"message":  "Fetching live balance data...",
		})
	}
}

func (a *APIActor) handleGetPositions(ctx *actor.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		exchangeName := chi.URLParam(r, "exchange")

		// Check if exchange is connected
		_, exists := a.exchangePIDs[exchangeName]
		if !exists {
			a.writeJSON(w, map[string]interface{}{
				"error":     "Exchange not found",
				"exchange":  exchangeName,
				"positions": []interface{}{},
			})
			return
		}

		// Get cached portfolio data for this exchange
		portfolioData, hasData := a.portfolioCache[exchangeName]
		if hasData {
			positions, hasPositions := portfolioData["positions"].([]map[string]interface{})
			if hasPositions {
				a.writeJSON(w, map[string]interface{}{
					"exchange":  exchangeName,
					"status":    "live_data",
					"positions": positions,
				})
				return
			}
		}

		// If no cached data, trigger refresh and return loading state
		if exchangePID, exists := a.exchangePIDs[exchangeName]; exists {
			ctx.Send(exchangePID, exchange.GetPositionsMsg{})
			a.logger.Debug().
				Str("exchange", exchangeName).
				Msg("Triggered position refresh")
		}

		a.writeJSON(w, map[string]interface{}{
			"exchange":  exchangeName,
			"status":    "loading",
			"positions": []interface{}{},
			"message":   "Fetching live position data...",
		})
	}
}

func (a *APIActor) handleGetStrategies(ctx *actor.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		allStrategies := make([]map[string]interface{}, 0)

		// Collect strategies from cache (populated by periodic updates from exchange actors)
		for exchangeName, strategies := range a.strategiesCache {
			a.logger.Debug().
				Str("exchange", exchangeName).
				Int("strategy_count", len(strategies)).
				Msg("Adding strategies from cache")
			allStrategies = append(allStrategies, strategies...)
		}

		// If no cached data available, trigger a refresh and return current state
		if len(allStrategies) == 0 && len(a.exchangePIDs) > 0 {
			// Trigger strategy data refresh from all exchanges
			for exchangeName, exchangePID := range a.exchangePIDs {
				ctx.Send(exchangePID, exchange.GetStrategiesMsg{})
				a.logger.Debug().
					Str("exchange", exchangeName).
					Msg("Triggered strategy refresh")
			}

			// Return a loading state since we just triggered refresh
			allStrategies = []map[string]interface{}{
				{
					"id":       "system:loading",
					"name":     "Loading Strategies",
					"symbol":   "N/A",
					"exchange": "system",
					"status":   "loading",
					"pnl":      "$0.00",
					"note":     fmt.Sprintf("Fetching live data from %d exchange(s)...", len(a.exchangePIDs)),
				},
			}
		} else if len(a.exchangePIDs) == 0 {
			// No exchanges configured
			allStrategies = []map[string]interface{}{
				{
					"id":       "system:no_exchanges",
					"name":     "No Exchanges",
					"symbol":   "N/A",
					"exchange": "system",
					"status":   "info",
					"pnl":      "$0.00",
					"note":     "No exchanges are currently configured",
				},
			}
		}

		a.writeJSON(w, map[string]interface{}{"strategies": allStrategies})
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
		// TODO: Implement GetOrders method in exchange interface
		// For now, return empty orders with a note about implementation status
		a.writeJSON(w, map[string]interface{}{
			"orders":  []interface{}{},
			"status":  "not_implemented",
			"message": "Order history retrieval not yet implemented",
		})
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
		// Aggregate portfolio data from all exchanges
		allBalances := make([]map[string]interface{}, 0)
		allPositions := make([]map[string]interface{}, 0)

		connectedExchanges := len(a.exchangePIDs)
		totalValue := 0.0
		totalUnrealizedPnL := 0.0
		availableCash := 0.0

		// Collect data from all exchange caches
		for exchangeName, portfolioData := range a.portfolioCache {
			if balances, ok := portfolioData["balances"].([]map[string]interface{}); ok {
				for _, balance := range balances {
					// Add exchange information to balance
					balance["exchange"] = exchangeName
					allBalances = append(allBalances, balance)

					// Calculate available cash (USDT/USD balances)
					if asset, ok := balance["asset"].(string); ok {
						if asset == "USDT" || asset == "USD" {
							if available, ok := balance["available"].(float64); ok {
								availableCash += available
							}
						}
					}
				}
			}

			if positions, ok := portfolioData["positions"].([]map[string]interface{}); ok {
				for _, position := range positions {
					// Add exchange information to position
					position["exchange"] = exchangeName
					allPositions = append(allPositions, position)

					// Calculate unrealized PnL
					if unrealizedPnl, ok := position["unrealized_pnl"].(float64); ok {
						totalUnrealizedPnL += unrealizedPnl
					}
				}
			}
		}

		// If no cached data, trigger refresh
		if len(a.portfolioCache) == 0 && connectedExchanges > 0 {
			for exchangeName, exchangePID := range a.exchangePIDs {
				ctx.Send(exchangePID, exchange.GetBalancesMsg{})
				ctx.Send(exchangePID, exchange.GetPositionsMsg{})
				a.logger.Debug().
					Str("exchange", exchangeName).
					Msg("Triggered portfolio data refresh")
			}
		}

		status := "connected"
		if connectedExchanges == 0 {
			status = "no_exchanges"
		} else if len(a.portfolioCache) == 0 {
			status = "loading"
		}

		a.writeJSON(w, map[string]interface{}{
			"total_value":         totalValue, // TODO: Calculate based on positions and prices
			"available_cash":      availableCash,
			"unrealized_pnl":      totalUnrealizedPnL,
			"realized_pnl":        0.0, // TODO: Get from trading history
			"connected_exchanges": connectedExchanges,
			"status":              status,
			"positions":           allPositions,
			"balances":            allBalances,
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

// Risk management handlers

func (a *APIActor) handleGetRiskParameters(ctx *actor.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		a.logger.Info().Msg("Getting all risk parameters")

		// Get all risk parameters
		parameters := []string{
			"max_position_size", "max_daily_loss", "max_portfolio_risk",
			"max_correlation", "max_leverage", "max_daily_trades",
			"max_hourly_trades", "var_limit", "max_drawdown_limit",
			"concentration_limit",
		}

		result := make(map[string]interface{})

		// For now, get from the first available exchange's risk manager
		for _, exchangePID := range a.exchangePIDs {
			for _, param := range parameters {
				// Send request to exchange actor to get risk parameter
				// Exchange actor will forward to risk manager
				msg := map[string]interface{}{
					"type":      "get_risk_parameter",
					"parameter": param,
				}

				response, err := ctx.Request(exchangePID, msg, 5*time.Second).Result()
				if err != nil {
					a.logger.Error().Err(err).Str("parameter", param).Msg("Failed to get risk parameter")
					continue
				}

				result[param] = response
			}
			break // Only use first exchange for now
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"exchange":   "all", // or specific exchange
			"parameters": result,
		})
	}
}

func (a *APIActor) handleSetRiskParameter(ctx *actor.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Parameter string `json:"parameter"`
			Value     string `json:"value"`
			Exchange  string `json:"exchange,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if req.Parameter == "" || req.Value == "" {
			http.Error(w, "Parameter and value are required", http.StatusBadRequest)
			return
		}

		a.logger.Info().
			Str("parameter", req.Parameter).
			Str("value", req.Value).
			Str("exchange", req.Exchange).
			Msg("Setting risk parameter")

		// If no exchange specified, apply to all exchanges
		exchanges := []string{}
		if req.Exchange != "" {
			exchanges = append(exchanges, req.Exchange)
		} else {
			for exchangeName := range a.exchangePIDs {
				exchanges = append(exchanges, exchangeName)
			}
		}

		results := make(map[string]interface{})
		for _, exchangeName := range exchanges {
			if exchangePID, exists := a.exchangePIDs[exchangeName]; exists {
				msg := map[string]interface{}{
					"type":      "set_risk_parameter",
					"parameter": req.Parameter,
					"value":     req.Value,
				}

				response, err := ctx.Request(exchangePID, msg, 5*time.Second).Result()
				if err != nil {
					a.logger.Error().
						Err(err).
						Str("exchange", exchangeName).
						Str("parameter", req.Parameter).
						Msg("Failed to set risk parameter")
					results[exchangeName] = map[string]interface{}{
						"success": false,
						"error":   err.Error(),
					}
				} else {
					results[exchangeName] = map[string]interface{}{
						"success": true,
						"result":  response,
					}
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
	}
}

func (a *APIActor) handleGetRiskParameter(ctx *actor.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		parameter := chi.URLParam(r, "parameter")
		if parameter == "" {
			http.Error(w, "Parameter name is required", http.StatusBadRequest)
			return
		}

		a.logger.Info().Str("parameter", parameter).Msg("Getting risk parameter")

		// Get from first available exchange
		for exchangeName, exchangePID := range a.exchangePIDs {
			msg := map[string]interface{}{
				"type":      "get_risk_parameter",
				"parameter": parameter,
			}

			response, err := ctx.Request(exchangePID, msg, 5*time.Second).Result()
			if err != nil {
				a.logger.Error().Err(err).Str("parameter", parameter).Msg("Failed to get risk parameter")
				http.Error(w, "Failed to get risk parameter", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"exchange":  exchangeName,
				"parameter": parameter,
				"result":    response,
			})
			return
		}

		http.Error(w, "No exchanges available", http.StatusServiceUnavailable)
	}
}

func (a *APIActor) handleGetRiskMetrics(ctx *actor.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		a.logger.Info().Msg("Getting risk metrics")

		results := make(map[string]interface{})

		// Get risk metrics from all exchanges
		for exchangeName, exchangePID := range a.exchangePIDs {
			msg := map[string]interface{}{
				"type": "get_risk_metrics",
			}

			response, err := ctx.Request(exchangePID, msg, 5*time.Second).Result()
			if err != nil {
				a.logger.Error().
					Err(err).
					Str("exchange", exchangeName).
					Msg("Failed to get risk metrics")
				results[exchangeName] = map[string]interface{}{
					"error": err.Error(),
				}
			} else {
				results[exchangeName] = response
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
	}
}

// Rebalance handlers
func (a *APIActor) handleGetRebalanceStatus(ctx *actor.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		exchangeName := chi.URLParam(r, "exchange")
		a.logger.Info().Str("exchange", exchangeName).Msg("Getting rebalance status")

		exchangePID, exists := a.exchangePIDs[exchangeName]
		if !exists {
			http.Error(w, "Exchange not found", http.StatusNotFound)
			return
		}

		msg := map[string]interface{}{
			"type": "get_rebalance_status",
		}

		response, err := ctx.Request(exchangePID, msg, 5*time.Second).Result()
		if err != nil {
			a.logger.Error().Err(err).Str("exchange", exchangeName).Msg("Failed to get rebalance status")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func (a *APIActor) handleStartRebalancing(ctx *actor.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		exchangeName := chi.URLParam(r, "exchange")
		a.logger.Info().Str("exchange", exchangeName).Msg("Starting rebalancing")

		exchangePID, exists := a.exchangePIDs[exchangeName]
		if !exists {
			http.Error(w, "Exchange not found", http.StatusNotFound)
			return
		}

		msg := map[string]interface{}{
			"type": "start_rebalancing",
		}

		response, err := ctx.Request(exchangePID, msg, 5*time.Second).Result()
		if err != nil {
			a.logger.Error().Err(err).Str("exchange", exchangeName).Msg("Failed to start rebalancing")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func (a *APIActor) handleStopRebalancing(ctx *actor.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		exchangeName := chi.URLParam(r, "exchange")
		a.logger.Info().Str("exchange", exchangeName).Msg("Stopping rebalancing")

		exchangePID, exists := a.exchangePIDs[exchangeName]
		if !exists {
			http.Error(w, "Exchange not found", http.StatusNotFound)
			return
		}

		msg := map[string]interface{}{
			"type": "stop_rebalancing",
		}

		response, err := ctx.Request(exchangePID, msg, 5*time.Second).Result()
		if err != nil {
			a.logger.Error().Err(err).Str("exchange", exchangeName).Msg("Failed to stop rebalancing")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func (a *APIActor) handleTriggerRebalance(ctx *actor.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		exchangeName := chi.URLParam(r, "exchange")
		a.logger.Info().Str("exchange", exchangeName).Msg("Triggering manual rebalance")

		exchangePID, exists := a.exchangePIDs[exchangeName]
		if !exists {
			http.Error(w, "Exchange not found", http.StatusNotFound)
			return
		}

		msg := map[string]interface{}{
			"type": "trigger_rebalance",
		}

		response, err := ctx.Request(exchangePID, msg, 5*time.Second).Result()
		if err != nil {
			a.logger.Error().Err(err).Str("exchange", exchangeName).Msg("Failed to trigger rebalance")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func (a *APIActor) handleLoadRebalanceScript(ctx *actor.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		exchangeName := chi.URLParam(r, "exchange")
		a.logger.Info().Str("exchange", exchangeName).Msg("Loading rebalance script")

		exchangePID, exists := a.exchangePIDs[exchangeName]
		if !exists {
			http.Error(w, "Exchange not found", http.StatusNotFound)
			return
		}

		var request struct {
			ScriptPath string `json:"script_path"`
		}

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		msg := map[string]interface{}{
			"type":        "load_rebalance_script",
			"script_path": request.ScriptPath,
		}

		response, err := ctx.Request(exchangePID, msg, 5*time.Second).Result()
		if err != nil {
			a.logger.Error().Err(err).Str("exchange", exchangeName).Msg("Failed to load rebalance script")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}
