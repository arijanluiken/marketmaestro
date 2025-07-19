package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"

	"github.com/arijanluiken/mercantile/pkg/config"
)

// Messages for API actor communication
type (
	StartServerMsg struct{}
	StopServerMsg  struct{}
	StatusMsg      struct{}
)

// APIActor provides REST API and WebSocket endpoints
type APIActor struct {
	config *config.Config
	logger zerolog.Logger
	server *http.Server
	router chi.Router
}

// New creates a new API actor
func New(cfg *config.Config, logger zerolog.Logger) *APIActor {
	return &APIActor{
		config: cfg,
		logger: logger,
	}
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
	default:
		a.logger.Debug().
			Str("message_type", fmt.Sprintf("%T", msg)).
			Msg("Received message")
	}
}

func (a *APIActor) onStarted(ctx *actor.Context) {
	a.logger.Info().Msg("API actor started")
	
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
	
	a.setupRouter()
	
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
		"server_running": a.server != nil,
		"port":          a.config.API.Port,
		"timestamp":     time.Now(),
	}
	
	ctx.Respond(status)
}

func (a *APIActor) setupRouter() {
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
			r.Get("/", a.handleGetExchanges)
			r.Get("/{exchange}/status", a.handleGetExchangeStatus)
			r.Get("/{exchange}/balances", a.handleGetBalances)
			r.Get("/{exchange}/positions", a.handleGetPositions)
		})
		
		// Strategy routes
		r.Route("/strategies", func(r chi.Router) {
			r.Get("/", a.handleGetStrategies)
			r.Post("/", a.handleCreateStrategy)
			r.Get("/{id}/status", a.handleGetStrategyStatus)
			r.Post("/{id}/start", a.handleStartStrategy)
			r.Post("/{id}/stop", a.handleStopStrategy)
		})
	})
	
	// WebSocket endpoint
	r.HandleFunc("/ws", a.handleWebSocket)
	
	a.router = r
}

func (a *APIActor) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok","timestamp":"` + time.Now().Format(time.RFC3339) + `"}`))
}

func (a *APIActor) handleOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	spec := `{
		"openapi": "3.0.0",
		"info": {
			"title": "Mercantile Trading Bot API",
			"version": "1.0.0",
			"description": "API for the Mercantile crypto trading bot"
		},
		"servers": [
			{
				"url": "http://localhost:` + fmt.Sprintf("%d", a.config.API.Port) + `/api/v1",
				"description": "Local server"
			}
		],
		"paths": {
			"/health": {
				"get": {
					"summary": "Health check",
					"responses": {
						"200": {
							"description": "Server is healthy"
						}
					}
				}
			},
			"/exchanges": {
				"get": {
					"summary": "List all exchanges",
					"responses": {
						"200": {
							"description": "List of exchanges"
						}
					}
				}
			}
		}
	}`
	
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(spec))
}

func (a *APIActor) handleGetExchanges(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"exchanges":[]}`))
}

func (a *APIActor) handleGetExchangeStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"not_implemented"}`))
}

func (a *APIActor) handleGetBalances(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"balances":[]}`))
}

func (a *APIActor) handleGetPositions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"positions":[]}`))
}

func (a *APIActor) handleGetStrategies(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"strategies":[]}`))
}

func (a *APIActor) handleCreateStrategy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"status":"created"}`))
}

func (a *APIActor) handleGetStrategyStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"not_implemented"}`))
}

func (a *APIActor) handleStartStrategy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"started"}`))
}

func (a *APIActor) handleStopStrategy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"stopped"}`))
}

func (a *APIActor) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement WebSocket handler
	http.Error(w, "WebSocket not implemented", http.StatusNotImplemented)
}