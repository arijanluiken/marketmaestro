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
		"port":           u.config.UI.Port,
		"timestamp":      time.Now(),
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
        async function loadQuickStatus() {
            try {
                // Load health status
                const healthResponse = await fetch('http://localhost:8080/api/v1/health');
                const healthData = await healthResponse.json();
                console.log('API Health:', healthData);
                
                // Load exchanges count
                const exchangesResponse = await fetch('http://localhost:8080/api/v1/exchanges/');
                const exchangesData = await exchangesResponse.json();
                const exchangeCount = exchangesData.exchanges ? exchangesData.exchanges.length : 0;
                document.getElementById('exchange-count').textContent = exchangeCount;
                
                // Load strategies count  
                const strategiesResponse = await fetch('http://localhost:8080/api/v1/strategies/');
                const strategiesData = await strategiesResponse.json();
                const strategyCount = strategiesData.strategies ? strategiesData.strategies.length : 0;
                document.getElementById('strategy-count').textContent = strategyCount;
                
            } catch (error) {
                console.error('API Error:', error);
                document.getElementById('exchange-count').textContent = 'Error';
                document.getElementById('strategy-count').textContent = 'Error';
            }
        }
        
        // Load data on page load
        loadQuickStatus();
        
        // Refresh every 30 seconds
        setInterval(loadQuickStatus, 30000);
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
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Strategies - Mercantile Trading Bot</title>
    <link rel="stylesheet" href="https://unpkg.com/purecss@3.0.0/build/pure-min.css">
    <style>
        body { background: #f5f5f5; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; }
        .container { max-width: 1200px; margin: 0 auto; padding: 20px; }
        .header { background: white; padding: 20px; border-radius: 8px; margin-bottom: 20px; box-shadow: 0 1px 3px rgba(0,0,0,0.1); }
        .nav { background: #333; padding: 10px 0; border-radius: 5px; margin-bottom: 20px; }
        .nav ul { list-style: none; margin: 0; padding: 0; display: flex; justify-content: center; }
        .nav li { margin: 0 20px; }
        .nav a { color: white; text-decoration: none; font-weight: 500; }
        .nav a:hover { color: #ddd; }
        .active { color: #4CAF50 !important; }
        .card { background: white; padding: 20px; border-radius: 8px; margin-bottom: 20px; box-shadow: 0 1px 3px rgba(0,0,0,0.1); }
        .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 20px; }
        .strategy-item { border: 1px solid #e0e0e0; border-radius: 8px; padding: 15px; margin-bottom: 15px; background: #fafafa; }
        .status-running { color: #4CAF50; font-weight: bold; }
        .status-stopped { color: #f44336; font-weight: bold; }
        .status-error { color: #ff9800; font-weight: bold; }
        .pnl-positive { color: #4CAF50; font-weight: bold; }
        .pnl-negative { color: #f44336; font-weight: bold; }
        .btn { background: #4CAF50; color: white; padding: 8px 16px; border: none; border-radius: 4px; cursor: pointer; margin-right: 10px; }
        .btn:hover { background: #45a049; }
        .btn-stop { background: #f44336; }
        .btn-stop:hover { background: #da190b; }
        .btn-restart { background: #ff9800; }
        .btn-restart:hover { background: #e68900; }
        .metrics { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 15px; margin-bottom: 20px; }
        .metric { text-align: center; padding: 15px; background: white; border-radius: 8px; box-shadow: 0 1px 3px rgba(0,0,0,0.1); }
        .metric-value { font-size: 2em; font-weight: bold; color: #333; }
        .metric-label { color: #666; font-size: 0.9em; margin-top: 5px; }
        .error { color: #f44336; font-style: italic; }
        .loading { color: #666; font-style: italic; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Mercantile Trading Bot</h1>
            <p>Automated Trading Strategies Management</p>
        </div>

        <nav class="nav">
            <ul>
                <li><a href="/">Home</a></li>
                <li><a href="/dashboard">Dashboard</a></li>
                <li><a href="/portfolio">Portfolio</a></li>
                <li><a href="/strategies" class="active">Strategies</a></li>
                <li><a href="/settings">Settings</a></li>
            </ul>
        </nav>

        <!-- Strategy Metrics Overview -->
        <div class="metrics">
            <div class="metric">
                <div class="metric-value" id="total-strategies">-</div>
                <div class="metric-label">Total Strategies</div>
            </div>
            <div class="metric">
                <div class="metric-value" id="running-strategies">-</div>
                <div class="metric-label">Running</div>
            </div>
            <div class="metric">
                <div class="metric-value" id="total-pnl">-</div>
                <div class="metric-label">Total P&L</div>
            </div>
            <div class="metric">
                <div class="metric-value" id="avg-performance">-</div>
                <div class="metric-label">Avg Performance</div>
            </div>
        </div>

        <div class="grid">
            <!-- Active Strategies -->
            <div class="card">
                <h3>Active Strategies</h3>
                <div id="active-strategies">
                    <div class="loading">Loading strategies...</div>
                </div>
            </div>

            <!-- Strategy Performance -->
            <div class="card">
                <h3>Performance Summary</h3>
                <div id="performance-summary">
                    <div class="loading">Loading performance data...</div>
                </div>
            </div>
        </div>

        <!-- All Strategies List -->
        <div class="card">
            <h3>All Strategies</h3>
            <div id="all-strategies">
                <div class="loading">Loading strategies...</div>
            </div>
        </div>
    </div>

    <script>
        async function loadStrategies() {
            try {
                const response = await fetch('http://localhost:8080/api/v1/strategies/');
                const data = await response.json();
                
                if (data.strategies) {
                    displayStrategies(data.strategies);
                    updateMetrics(data.strategies);
                } else {
                    document.getElementById('active-strategies').innerHTML = '<div class="error">No strategies found</div>';
                    document.getElementById('all-strategies').innerHTML = '<div class="error">No strategies found</div>';
                }
            } catch (error) {
                console.error('Error loading strategies:', error);
                document.getElementById('active-strategies').innerHTML = '<div class="error">Failed to load strategies</div>';
                document.getElementById('all-strategies').innerHTML = '<div class="error">Failed to load strategies</div>';
            }
        }

        function displayStrategies(strategies) {
            const activeStrategies = strategies.filter(s => s.status === 'running');
            const allStrategies = strategies;

            // Display active strategies
            const activeContainer = document.getElementById('active-strategies');
            if (activeStrategies.length === 0) {
                activeContainer.innerHTML = '<p>No active strategies</p>';
            } else {
                activeContainer.innerHTML = activeStrategies.map(strategy => createStrategyCard(strategy, true)).join('');
            }

            // Display all strategies
            const allContainer = document.getElementById('all-strategies');
            allContainer.innerHTML = allStrategies.map(strategy => createStrategyCard(strategy, false)).join('');

            // Display performance summary
            const performanceContainer = document.getElementById('performance-summary');
            const totalPnL = strategies.reduce((sum, s) => sum + parseFloat(s.pnl.replace(/[^-\d.]/g, '')), 0);
            const runningCount = strategies.filter(s => s.status === 'running').length;
            
            performanceContainer.innerHTML = ` + "`" + `
                <p><strong>Total P&L:</strong> <span class="${totalPnL >= 0 ? 'pnl-positive' : 'pnl-negative'}">${totalPnL >= 0 ? '+' : ''}${totalPnL.toFixed(2)}</span></p>
                <p><strong>Running Strategies:</strong> ${runningCount}/${strategies.length}</p>
                <p><strong>Success Rate:</strong> ${((runningCount / strategies.length) * 100).toFixed(1)}%</p>
            ` + "`" + `;
        }

        function createStrategyCard(strategy, isActive) {
            const statusClass = 'status-' + strategy.status;
            const pnlValue = parseFloat(strategy.pnl.replace(/[^-\d.]/g, ''));
            const pnlClass = pnlValue >= 0 ? 'pnl-positive' : 'pnl-negative';
            
            const actions = isActive ? 
                '<button class="btn btn-stop" onclick="stopStrategy(\'' + strategy.id + '\')">Stop</button>' +
                '<button class="btn btn-restart" onclick="restartStrategy(\'' + strategy.id + '\')">Restart</button>'
                : '<button class="btn" onclick="startStrategy(\'' + strategy.id + '\')">Start</button>';

            return '<div class="strategy-item">' +
                '<div style="display: flex; justify-content: space-between; align-items: center;">' +
                    '<div>' +
                        '<h4>' + strategy.name + '</h4>' +
                        '<p><strong>Symbol:</strong> ' + strategy.symbol + ' | <strong>Exchange:</strong> ' + strategy.exchange + '</p>' +
                        '<p><strong>Status:</strong> <span class="' + statusClass + '">' + strategy.status + '</span></p>' +
                        '<p><strong>P&L:</strong> <span class="' + pnlClass + '">' + strategy.pnl + '</span></p>' +
                    '</div>' +
                    '<div>' + actions + '</div>' +
                '</div>' +
            '</div>';
        }

        function updateMetrics(strategies) {
            const totalStrategies = strategies.length;
            const runningStrategies = strategies.filter(s => s.status === 'running').length;
            const totalPnL = strategies.reduce((sum, s) => sum + parseFloat(s.pnl.replace(/[^-\d.]/g, '')), 0);
            const avgPerformance = totalStrategies > 0 ? (totalPnL / totalStrategies) : 0;

            document.getElementById('total-strategies').textContent = totalStrategies;
            document.getElementById('running-strategies').textContent = runningStrategies;
            document.getElementById('total-pnl').textContent = (totalPnL >= 0 ? '+' : '') + totalPnL.toFixed(2);
            document.getElementById('total-pnl').className = 'metric-value ' + (totalPnL >= 0 ? 'pnl-positive' : 'pnl-negative');
            document.getElementById('avg-performance').textContent = (avgPerformance >= 0 ? '+' : '') + avgPerformance.toFixed(2);
            document.getElementById('avg-performance').className = 'metric-value ' + (avgPerformance >= 0 ? 'pnl-positive' : 'pnl-negative');
        }

        function startStrategy(strategyId) {
            // TODO: Implement start strategy API call
            console.log('Starting strategy:', strategyId);
            alert('Start strategy functionality not yet implemented');
        }

        function stopStrategy(strategyId) {
            // TODO: Implement stop strategy API call
            console.log('Stopping strategy:', strategyId);
            alert('Stop strategy functionality not yet implemented');
        }

        function restartStrategy(strategyId) {
            // TODO: Implement restart strategy API call
            console.log('Restarting strategy:', strategyId);
            alert('Restart strategy functionality not yet implemented');
        }

        // Load strategies when page loads
        document.addEventListener('DOMContentLoaded', loadStrategies);

        // Refresh strategies every 30 seconds
        setInterval(loadStrategies, 30000);
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func (u *UIActor) handlePortfolio(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Portfolio - Mercantile Trading Bot</title>
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
        .stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 20px; margin-bottom: 20px; }
        .stat-card { background: #f8f9fa; padding: 20px; border-radius: 8px; border: 1px solid #e9ecef; }
        .stat-value { font-size: 24px; font-weight: bold; color: #2c3e50; }
        .stat-label { color: #6c757d; font-size: 14px; }
        .position-table { width: 100%; border-collapse: collapse; margin-top: 10px; }
        .position-table th, .position-table td { padding: 12px; text-align: left; border-bottom: 1px solid #e9ecef; }
        .position-table th { background: #f8f9fa; font-weight: 600; }
        .positive { color: #28a745; }
        .negative { color: #dc3545; }
        .loading { text-align: center; padding: 20px; color: #6c757d; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Portfolio Overview</h1>
            <ul class="nav">
                <li><a href="/">Home</a></li>
                <li><a href="/dashboard">Dashboard</a></li>
                <li><a href="/strategies">Strategies</a></li>
                <li><a href="/portfolio">Portfolio</a></li>
                <li><a href="/settings">Settings</a></li>
            </ul>
        </div>
        
        <div class="stats" id="portfolio-stats">
            <div class="stat-card">
                <div class="stat-value" id="total-value">Loading...</div>
                <div class="stat-label">Total Value</div>
            </div>
            <div class="stat-card">
                <div class="stat-value" id="available-cash">Loading...</div>
                <div class="stat-label">Available Cash</div>
            </div>
            <div class="stat-card">
                <div class="stat-value" id="unrealized-pnl">Loading...</div>
                <div class="stat-label">Unrealized P&L</div>
            </div>
            <div class="stat-card">
                <div class="stat-value" id="realized-pnl">Loading...</div>
                <div class="stat-label">Realized P&L</div>
            </div>
        </div>
        
        <div class="card">
            <h3>Active Positions</h3>
            <div id="positions-content">
                <div class="loading">Loading positions...</div>
            </div>
        </div>
    </div>
    
    <script>
        // Fetch portfolio data from API
        async function loadPortfolioData() {
            try {
                const response = await fetch('http://localhost:8080/api/v1/portfolio');
                const data = await response.json();
                
                document.getElementById('total-value').textContent = '$' + data.total_value.toLocaleString();
                document.getElementById('available-cash').textContent = '$' + data.available_cash.toLocaleString();
                document.getElementById('unrealized-pnl').textContent = '$' + data.unrealized_pnl.toLocaleString();
                document.getElementById('realized-pnl').textContent = '$' + data.realized_pnl.toLocaleString();
                
                // Update P&L colors
                const unrealizedEl = document.getElementById('unrealized-pnl');
                const realizedEl = document.getElementById('realized-pnl');
                
                unrealizedEl.className = 'stat-value ' + (data.unrealized_pnl >= 0 ? 'positive' : 'negative');
                realizedEl.className = 'stat-value ' + (data.realized_pnl >= 0 ? 'positive' : 'negative');
                
                // Load positions
                if (data.positions && data.positions.length > 0) {
                    renderPositions(data.positions);
                } else {
                    loadExchangePositions();
                }
            } catch (error) {
                console.error('Error loading portfolio data:', error);
                document.getElementById('positions-content').innerHTML = '<div class="loading">Error loading portfolio data</div>';
            }
        }
        
        async function loadExchangePositions() {
            try {
                const response = await fetch('http://localhost:8080/api/v1/exchanges/bybit/positions');
                const data = await response.json();
                
                if (data.positions && data.positions.length > 0) {
                    renderPositions(data.positions);
                } else {
                    document.getElementById('positions-content').innerHTML = '<div class="loading">No active positions</div>';
                }
            } catch (error) {
                console.error('Error loading exchange positions:', error);
                document.getElementById('positions-content').innerHTML = '<div class="loading">Error loading positions</div>';
            }
        }
        
        function renderPositions(positions) {
            const html = ` + "`" + `
                <table class="position-table">
                    <thead>
                        <tr>
                            <th>Symbol</th>
                            <th>Quantity</th>
                            <th>Avg Price</th>
                            <th>Current Price</th>
                            <th>Unrealized P&L</th>
                            <th>Side</th>
                        </tr>
                    </thead>
                    <tbody>
                        ${positions.map(pos => ` + "`" + `
                            <tr>
                                <td>${pos.symbol}</td>
                                <td>${pos.quantity}</td>
                                <td>$${pos.average_price.toLocaleString()}</td>
                                <td>$${pos.current_price.toLocaleString()}</td>
                                <td class="${pos.unrealized_pnl >= 0 ? 'positive' : 'negative'}">
                                    $${pos.unrealized_pnl.toLocaleString()}
                                </td>
                                <td>${pos.side || 'long'}</td>
                            </tr>
                        ` + "`" + `).join('')}
                    </tbody>
                </table>
            ` + "`" + `;
            document.getElementById('positions-content').innerHTML = html;
        }
        
        // Load data on page load
        loadPortfolioData();
        
        // Refresh every 30 seconds
        setInterval(loadPortfolioData, 30000);
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func (u *UIActor) handleSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte("<h1>Settings - Coming Soon</h1>"))
}
