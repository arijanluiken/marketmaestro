package ui

import (
	"context"
	"embed"
	"fmt"
	"net/http"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"

	"github.com/arijanluiken/mercantile/pkg/config"
)

//go:embed assets/*
var assets embed.FS

// Messages for UI actor communication
type (
	StartServerMsg struct{}
	StopServerMsg  struct{}
	StatusMsg      struct{}
)

// UIActor provides the web interface
type UIActor struct {
	config *config.Config
	logger zerolog.Logger
	server *http.Server
	router chi.Router
}

// New creates a new UI actor
func New(cfg *config.Config, logger zerolog.Logger) *UIActor {
	return &UIActor{
		config: cfg,
		logger: logger,
	}
}

// Receive handles incoming messages
func (u *UIActor) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Started:
		u.onStarted(ctx)
	case actor.Stopped:
		u.onStopped(ctx)
	case StartServerMsg:
		u.onStartServer(ctx)
	case StopServerMsg:
		u.onStopServer(ctx)
	case StatusMsg:
		u.onStatus(ctx)
	default:
		u.logger.Debug().
			Str("message_type", fmt.Sprintf("%T", msg)).
			Msg("Received message")
	}
}

func (u *UIActor) onStarted(ctx *actor.Context) {
	u.logger.Info().Msg("UI actor started")
	
	// Auto-start the server
	ctx.Send(ctx.PID(), StartServerMsg{})
}

func (u *UIActor) onStopped(ctx *actor.Context) {
	u.logger.Info().Msg("UI actor stopped")
	
	if u.server != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		u.server.Shutdown(shutdownCtx)
	}
}

func (u *UIActor) onStartServer(ctx *actor.Context) {
	u.logger.Info().Int("port", u.config.UI.Port).Msg("Starting UI server")
	
	u.setupRouter()
	
	u.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", u.config.UI.Port),
		Handler: u.router,
	}
	
	go func() {
		if err := u.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			u.logger.Error().Err(err).Msg("UI server error")
		}
	}()
	
	u.logger.Info().Msg("UI server started successfully")
}

func (u *UIActor) onStopServer(ctx *actor.Context) {
	if u.server == nil {
		return
	}
	
	u.logger.Info().Msg("Stopping UI server")
	
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := u.server.Shutdown(shutdownCtx); err != nil {
		u.logger.Error().Err(err).Msg("Error stopping UI server")
	} else {
		u.logger.Info().Msg("UI server stopped successfully")
	}
}

func (u *UIActor) onStatus(ctx *actor.Context) {
	status := map[string]interface{}{
		"server_running": u.server != nil,
		"port":          u.config.UI.Port,
		"timestamp":     time.Now(),
	}
	
	ctx.Respond(status)
}

func (u *UIActor) setupRouter() {
	r := chi.NewRouter()
	
	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	
	// Serve embedded static assets
	r.Handle("/assets/*", http.FileServer(http.FS(assets)))
	
	// Routes
	r.Get("/", u.handleIndex)
	r.Get("/dashboard", u.handleDashboard)
	r.Get("/strategies", u.handleStrategies)
	r.Get("/portfolio", u.handlePortfolio)
	r.Get("/settings", u.handleSettings)
	
	u.router = r
}

func (u *UIActor) handleIndex(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Mercantile Trading Bot</title>
    <link rel="stylesheet" href="https://unpkg.com/purecss@3.0.0/build/pure-min.css">
    <style>
        body { background: #f5f5f5; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; }
        .container { max-width: 1200px; margin: 0 auto; padding: 20px; }
        .header { background: white; padding: 20px; margin-bottom: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .card { background: white; padding: 20px; margin-bottom: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .nav { list-style: none; padding: 0; margin: 0; display: flex; gap: 20px; }
        .nav a { text-decoration: none; color: #333; padding: 10px 15px; border-radius: 4px; }
        .nav a:hover { background: #eee; }
        .status { display: inline-block; padding: 4px 8px; border-radius: 4px; font-size: 12px; font-weight: bold; }
        .status.running { background: #d4edda; color: #155724; }
        .status.stopped { background: #f8d7da; color: #721c24; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Mercantile Trading Bot</h1>
            <ul class="nav">
                <li><a href="/">Home</a></li>
                <li><a href="/dashboard">Dashboard</a></li>
                <li><a href="/strategies">Strategies</a></li>
                <li><a href="/portfolio">Portfolio</a></li>
                <li><a href="/settings">Settings</a></li>
            </ul>
        </div>
        
        <div class="card">
            <h2>Welcome to Mercantile</h2>
            <p>Your advanced crypto trading bot built with the actor model.</p>
            <p>Use the navigation above to explore different sections of the application.</p>
        </div>
        
        <div class="card">
            <h3>Quick Status</h3>
            <p>Bot Status: <span class="status running">Running</span></p>
            <p>Connected Exchanges: <span id="exchange-count">Loading...</span></p>
            <p>Active Strategies: <span id="strategy-count">Loading...</span></p>
        </div>
    </div>
    
    <script>
        // Load status data from API
        fetch('http://localhost:8080/api/v1/health')
            .then(response => response.json())
            .then(data => {
                console.log('API Health:', data);
            })
            .catch(error => {
                console.error('API Error:', error);
            });
    </script>
</body>
</html>`
	
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func (u *UIActor) handleDashboard(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Dashboard - Mercantile Trading Bot</title>
    <link rel="stylesheet" href="https://unpkg.com/purecss@3.0.0/build/pure-min.css">
    <style>
        body { background: #f5f5f5; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; }
        .container { max-width: 1200px; margin: 0 auto; padding: 20px; }
        .header { background: white; padding: 20px; margin-bottom: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .card { background: white; padding: 20px; margin-bottom: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .nav { list-style: none; padding: 0; margin: 0; display: flex; gap: 20px; }
        .nav a { text-decoration: none; color: #333; padding: 10px 15px; border-radius: 4px; }
        .nav a:hover { background: #eee; }
        .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 20px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Dashboard</h1>
            <ul class="nav">
                <li><a href="/">Home</a></li>
                <li><a href="/dashboard">Dashboard</a></li>
                <li><a href="/strategies">Strategies</a></li>
                <li><a href="/portfolio">Portfolio</a></li>
                <li><a href="/settings">Settings</a></li>
            </ul>
        </div>
        
        <div class="grid">
            <div class="card">
                <h3>Exchange Status</h3>
                <p>Real-time exchange connection status and data feeds.</p>
                <div id="exchange-status">Loading...</div>
            </div>
            
            <div class="card">
                <h3>Active Strategies</h3>
                <p>Currently running trading strategies.</p>
                <div id="strategy-status">Loading...</div>
            </div>
            
            <div class="card">
                <h3>Portfolio Overview</h3>
                <p>Account balances and P&L summary.</p>
                <div id="portfolio-status">Loading...</div>
            </div>
        </div>
    </div>
</body>
</html>`
	
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func (u *UIActor) handleStrategies(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte("<h1>Strategies - Coming Soon</h1>"))
}

func (u *UIActor) handlePortfolio(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte("<h1>Portfolio - Coming Soon</h1>"))
}

func (u *UIActor) handleSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte("<h1>Settings - Coming Soon</h1>"))
}