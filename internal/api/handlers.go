package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/go-chi/chi/v5"

	"github.com/arijanluiken/mercantile/internal/exchange"
)

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

		// Find strategy in cache
		var strategy map[string]interface{}
		found := false

		for _, strategies := range a.strategiesCache {
			for _, s := range strategies {
				if s["id"] == strategyID {
					strategy = s
					found = true
					break
				}
			}
			if found {
				break
			}
		}

		if !found {
			a.writeError(w, "Strategy not found", http.StatusNotFound)
			return
		}

		// Get additional details from database
		details := a.getStrategyDetails(strategyID)

		// Merge strategy data with details
		for k, v := range details {
			strategy[k] = v
		}

		a.writeJSON(w, strategy)
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

		_, exists := a.exchangePIDs[exchangeName]
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

		// TODO: Implement actual script loading
		response := map[string]interface{}{
			"status":  "success",
			"message": "Script loading not yet implemented",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// getStrategyDetails retrieves additional strategy details from database
func (a *APIActor) getStrategyDetails(strategyID string) map[string]interface{} {
	details := make(map[string]interface{})

	if a.db == nil {
		a.logger.Warn().Msg("Database not available for strategy details")
		return details
	}

	// Get recent orders for this strategy
	orders, err := a.getStrategyOrders(strategyID, 10)
	if err != nil {
		a.logger.Error().Err(err).Str("strategy_id", strategyID).Msg("Failed to get strategy orders")
	} else {
		details["recent_orders"] = orders
	}

	// Get strategy statistics
	stats, err := a.getStrategyStats(strategyID)
	if err != nil {
		a.logger.Error().Err(err).Str("strategy_id", strategyID).Msg("Failed to get strategy stats")
	} else {
		details["stats"] = stats
	}

	// Get recent logs from database
	logs, err := a.getStrategyLogs(strategyID, 10)
	if err != nil {
		a.logger.Error().Err(err).Str("strategy_id", strategyID).Msg("Failed to get strategy logs")
		details["recent_logs"] = []map[string]interface{}{} // Return empty array instead of mock data
	} else {
		details["recent_logs"] = logs
	}

	return details
}

// getStrategyOrders retrieves recent orders for a strategy
func (a *APIActor) getStrategyOrders(strategyID string, limit int) ([]map[string]interface{}, error) {
	// Extract symbol from strategy ID for now (in a real implementation, you'd have a proper mapping)
	// Strategy ID format might be like "strategy:bybit:BTCUSDT:simple_sma"

	query := `
		SELECT order_id, symbol, side, type, quantity, price, status, created_at
		FROM orders 
		WHERE symbol LIKE '%' || ? || '%'
		ORDER BY created_at DESC 
		LIMIT ?
	`

	rows, err := a.db.Query(query, strategyID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query orders: %w", err)
	}
	defer rows.Close()

	var orders []map[string]interface{}
	for rows.Next() {
		var orderID, symbol, side, orderType, status, createdAt string
		var quantity, price float64

		if err := rows.Scan(&orderID, &symbol, &side, &orderType, &quantity, &price, &status, &createdAt); err != nil {
			continue
		}

		orders = append(orders, map[string]interface{}{
			"order_id":   orderID,
			"symbol":     symbol,
			"side":       side,
			"type":       orderType,
			"quantity":   quantity,
			"price":      price,
			"status":     status,
			"created_at": createdAt,
		})
	}

	return orders, nil
}

// getStrategyStats calculates strategy statistics
func (a *APIActor) getStrategyStats(strategyID string) (map[string]interface{}, error) {
	stats := map[string]interface{}{
		"total_orders":   0,
		"buy_orders":     0,
		"sell_orders":    0,
		"success_rate":   0.0,
		"total_volume":   0.0,
		"avg_order_size": 0.0,
	}

	// Count total orders
	query := `
		SELECT 
			COUNT(*) as total_orders,
			SUM(CASE WHEN side = 'buy' THEN 1 ELSE 0 END) as buy_orders,
			SUM(CASE WHEN side = 'sell' THEN 1 ELSE 0 END) as sell_orders,
			SUM(quantity * price) as total_volume,
			AVG(quantity) as avg_order_size
		FROM orders 
		WHERE symbol LIKE '%' || ? || '%'
	`

	row := a.db.QueryRow(query, strategyID)

	var totalOrders, buyOrders, sellOrders int
	var totalVolume, avgOrderSize sql.NullFloat64

	if err := row.Scan(&totalOrders, &buyOrders, &sellOrders, &totalVolume, &avgOrderSize); err != nil {
		return stats, fmt.Errorf("failed to get strategy stats: %w", err)
	}

	stats["total_orders"] = totalOrders
	stats["buy_orders"] = buyOrders
	stats["sell_orders"] = sellOrders

	if totalVolume.Valid {
		stats["total_volume"] = totalVolume.Float64
	}
	if avgOrderSize.Valid {
		stats["avg_order_size"] = avgOrderSize.Float64
	}

	// Calculate success rate (filled orders / total orders)
	successQuery := `
		SELECT COUNT(*) FROM orders 
		WHERE symbol LIKE '%' || ? || '%' AND status = 'filled'
	`
	var filledOrders int
	if err := a.db.QueryRow(successQuery, strategyID).Scan(&filledOrders); err == nil && totalOrders > 0 {
		stats["success_rate"] = float64(filledOrders) / float64(totalOrders) * 100
	}

	return stats, nil
}

// getStrategyLogs retrieves recent logs for a strategy from cache
func (a *APIActor) getStrategyLogs(strategyID string, limit int) ([]map[string]interface{}, error) {
	// Check if we have logs in cache for this strategy
	logs, exists := a.logsCache[strategyID]
	if !exists {
		// No logs found in cache
		a.logger.Debug().
			Str("strategy_id", strategyID).
			Msg("No logs found in cache for strategy")
		return []map[string]interface{}{}, nil
	}

	// Apply limit if needed
	if limit > 0 && len(logs) > limit {
		// Return the most recent logs (last 'limit' entries)
		start := len(logs) - limit
		return logs[start:], nil
	}

	return logs, nil
}
