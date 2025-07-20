package ui

import (
	"context"
	"embed"
	"fmt"
	"html/template"
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

// Shared template functions and components
var templateFuncs = template.FuncMap{
	"add": func(a, b int) int { return a + b },
	"sub": func(a, b int) int { return a - b },
}

// BasePageData contains common data for all pages
type BasePageData struct {
	Title       string
	CurrentPage string
	Subtitle    string
}

// Shared HTML templates as constants
const baseTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}} - Mercantile Trading Bot</title>
    <link rel="stylesheet" href="/assets/css/style.css">
</head>
<body>
    <div class="app-container">
        <header class="app-header">
            <div class="header-content">
                <div class="header-brand">
                    <h1>Mercantile</h1>
                    <div class="subtitle">{{.Subtitle}}</div>
                </div>
                <div class="header-status">
                    <div class="status-indicator" id="connection-status">
                        <div class="status-dot"></div>
                        <span>Checking...</span>
                    </div>
                </div>
            </div>
        </header>

        <nav class="app-navigation">
            <ul class="nav-list">
                <li class="nav-item">
                    <a href="/" class="nav-link {{if eq .CurrentPage "/"}}active{{end}}">
                        <span class="nav-icon">🏠</span>
                        <span class="nav-label">Home</span>
                    </a>
                </li>
                <li class="nav-item">
                    <a href="/dashboard" class="nav-link {{if eq .CurrentPage "/dashboard"}}active{{end}}">
                        <span class="nav-icon">📊</span>
                        <span class="nav-label">Dashboard</span>
                    </a>
                </li>
                <li class="nav-item">
                    <a href="/strategies" class="nav-link {{if eq .CurrentPage "/strategies"}}active{{end}}">
                        <span class="nav-icon">🤖</span>
                        <span class="nav-label">Strategies</span>
                    </a>
                </li>
                <li class="nav-item">
                    <a href="/portfolio" class="nav-link {{if eq .CurrentPage "/portfolio"}}active{{end}}">
                        <span class="nav-icon">💰</span>
                        <span class="nav-label">Portfolio</span>
                    </a>
                </li>
                <li class="nav-item">
                    <a href="/settings" class="nav-link {{if eq .CurrentPage "/settings"}}active{{end}}">
                        <span class="nav-icon">⚙️</span>
                        <span class="nav-label">Settings</span>
                    </a>
                </li>
            </ul>
        </nav>

        <main class="app-main">
            {{block "content" .}}{{end}}
        </main>
    </div>

    <script src="/assets/js/app.js"></script>
    {{block "scripts" .}}{{end}}
</body>
</html>
`

// Page templates
const indexTemplate = baseTemplate + `
{{define "content"}}
<div class="page-content">
    <div class="card">
        <h2>Welcome to Mercantile</h2>
        <p>Your advanced crypto trading bot built with the actor model.</p>
        <p>Use the navigation above to explore different sections of the application.</p>
    </div>
    
    <div class="metrics-grid">
        <div class="metric-card">
            <div class="metric-value" id="bot-status">Running</div>
            <div class="metric-label">Bot Status</div>
        </div>
        <div class="metric-card">
            <div class="metric-value" id="exchange-count">Loading...</div>
            <div class="metric-label">Connected Exchanges</div>
        </div>
        <div class="metric-card">
            <div class="metric-value" id="strategy-count">Loading...</div>
            <div class="metric-label">Active Strategies</div>
        </div>
    </div>
</div>
{{end}}

{{define "scripts"}}
<script>
// Load status data from API
async function loadQuickStatus() {
    try {
        // Load health status
        const healthResponse = await MercantileUI.API.health();
        console.log('API Health:', healthResponse);
        
        // Load exchanges count
        const exchangesData = await MercantileUI.API.exchanges.list();
        const exchangeCount = exchangesData.exchanges ? exchangesData.exchanges.length : 0;
        document.getElementById('exchange-count').textContent = exchangeCount;
        
        // Load strategies count  
        const strategiesData = await MercantileUI.API.strategies.list();
        const strategyCount = strategiesData.strategies ? strategiesData.strategies.length : 0;
        document.getElementById('strategy-count').textContent = strategyCount;
        
    } catch (error) {
        console.error('API Error:', error);
        document.getElementById('exchange-count').textContent = 'Error';
        document.getElementById('strategy-count').textContent = 'Error';
    }
}

// Initialize auto-refresh for home page
MercantileUI.AutoRefresh.start('quickStatus', loadQuickStatus, 30000);
</script>
{{end}}
`

const dashboardTemplate = baseTemplate + `
{{define "content"}}
<div class="page-content">
    <div class="metrics-grid">
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
{{end}}

{{define "scripts"}}
<script>
async function loadDashboardData() {
    try {
        // Load exchange status
        const exchangesData = await MercantileUI.API.exchanges.list();
        const exchangeStatusEl = document.getElementById('exchange-status');
        exchangeStatusEl.innerHTML = exchangesData.exchanges ? 
            exchangesData.exchanges.map(ex => '<div class="status-running">' + ex.name + ': Connected</div>').join('') :
            '<div class="text-muted">No exchanges configured</div>';
        
        // Load strategy status
        const strategiesData = await MercantileUI.API.strategies.list();
        const strategyStatusEl = document.getElementById('strategy-status');
        const runningStrategies = strategiesData.strategies ? 
            strategiesData.strategies.filter(s => s.status === 'running') : [];
        strategyStatusEl.innerHTML = runningStrategies.length > 0 ?
            runningStrategies.map(s => '<div class="status-running">' + s.name + ': ' + s.status + '</div>').join('') :
            '<div class="text-muted">No active strategies</div>';
            
        // Load portfolio summary
        const portfolioData = await MercantileUI.API.portfolio.summary();
        const portfolioStatusEl = document.getElementById('portfolio-status');
        portfolioStatusEl.innerHTML = portfolioData ? 
            '<div>Total Value: ' + MercantileUI.Utils.formatCurrency(portfolioData.total_value) + '</div>' +
            '<div class="' + MercantileUI.Utils.getPnLClass(portfolioData.total_pnl) + '">' +
                'P&L: ' + MercantileUI.Utils.formatCurrency(portfolioData.total_pnl) +
            '</div>' :
            '<div class="text-muted">No portfolio data available</div>';
            
    } catch (error) {
        console.error('Dashboard Error:', error);
    }
}

// Initialize auto-refresh for dashboard
MercantileUI.AutoRefresh.start('dashboard', loadDashboardData, 30000);
</script>
{{end}}
`

const strategiesTemplate = baseTemplate + `
{{define "content"}}
<div class="page-content">
    <!-- Strategy Metrics Overview -->
    <div class="metrics-grid">
        <div class="metric-card">
            <div class="metric-value" id="total-strategies">-</div>
            <div class="metric-label">Total Strategies</div>
        </div>
        <div class="metric-card">
            <div class="metric-value" id="running-strategies">-</div>
            <div class="metric-label">Running</div>
        </div>
        <div class="metric-card">
            <div class="metric-value" id="total-pnl">-</div>
            <div class="metric-label">Total P&L</div>
        </div>
        <div class="metric-card">
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
{{end}}

{{define "scripts"}}
<script>
async function loadStrategies() {
    try {
        const data = await MercantileUI.API.strategies.list();
        
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
        activeContainer.innerHTML = activeStrategies.map(strategy => MercantileUI.Components.strategyCard(strategy)).join('');
    }

    // Display all strategies
    const allContainer = document.getElementById('all-strategies');
    allContainer.innerHTML = allStrategies.map(strategy => MercantileUI.Components.strategyCard(strategy)).join('');

    // Display performance summary
    const performanceContainer = document.getElementById('performance-summary');
    const totalPnL = strategies.reduce((sum, s) => sum + parseFloat(s.pnl?.replace(/[^-\d.]/g, '') || '0'), 0);
    const runningCount = strategies.filter(s => s.status === 'running').length;
    
    performanceContainer.innerHTML = '<p><strong>Total P&L:</strong> <span class="' + MercantileUI.Utils.getPnLClass(totalPnL) + '">' + MercantileUI.Utils.formatCurrency(totalPnL) + '</span></p>' +
        '<p><strong>Running Strategies:</strong> ' + runningCount + '/' + strategies.length + '</p>' +
        '<p><strong>Success Rate:</strong> ' + ((runningCount / strategies.length) * 100).toFixed(1) + '%</p>';
}

function updateMetrics(strategies) {
    const totalStrategies = strategies.length;
    const runningStrategies = strategies.filter(s => s.status === 'running').length;
    const totalPnL = strategies.reduce((sum, s) => sum + parseFloat(s.pnl?.replace(/[^-\d.]/g, '') || '0'), 0);
    const avgPerformance = totalStrategies > 0 ? (totalPnL / totalStrategies) : 0;

    document.getElementById('total-strategies').textContent = totalStrategies;
    document.getElementById('running-strategies').textContent = runningStrategies;
    document.getElementById('total-pnl').textContent = MercantileUI.Utils.formatCurrency(totalPnL);
    document.getElementById('total-pnl').className = 'metric-value ' + MercantileUI.Utils.getPnLClass(totalPnL);
    document.getElementById('avg-performance').textContent = MercantileUI.Utils.formatCurrency(avgPerformance);
    document.getElementById('avg-performance').className = 'metric-value ' + MercantileUI.Utils.getPnLClass(avgPerformance);
}

// Initialize strategies page
MercantileUI.AutoRefresh.start('strategies', loadStrategies, 30000);
</script>
{{end}}
`

const portfolioTemplate = baseTemplate + `
{{define "content"}}
<div class="page-content">
    <div class="metrics-grid">
        <div class="metric-card">
            <div class="metric-value" id="total-value">Loading...</div>
            <div class="metric-label">Total Value</div>
        </div>
        <div class="metric-card">
            <div class="metric-value" id="available-cash">Loading...</div>
            <div class="metric-label">Available Cash</div>
        </div>
        <div class="metric-card">
            <div class="metric-value" id="unrealized-pnl">Loading...</div>
            <div class="metric-label">Unrealized P&L</div>
        </div>
        <div class="metric-card">
            <div class="metric-value" id="realized-pnl">Loading...</div>
            <div class="metric-label">Realized P&L</div>
        </div>
    </div>
    
    <div class="card">
        <h3>Active Positions</h3>
        <div id="positions-content">
            <div class="loading">Loading positions...</div>
        </div>
    </div>
</div>
{{end}}

{{define "scripts"}}
<script>
async function loadPortfolioData() {
    try {
        const data = await MercantileUI.API.portfolio.get();
        
        document.getElementById('total-value').textContent = MercantileUI.Utils.formatCurrency(data.total_value);
        document.getElementById('available-cash').textContent = MercantileUI.Utils.formatCurrency(data.available_cash);
        document.getElementById('unrealized-pnl').textContent = MercantileUI.Utils.formatCurrency(data.unrealized_pnl);
        document.getElementById('realized-pnl').textContent = MercantileUI.Utils.formatCurrency(data.realized_pnl);
        
        // Update P&L colors
        const unrealizedEl = document.getElementById('unrealized-pnl');
        const realizedEl = document.getElementById('realized-pnl');
        
        unrealizedEl.className = 'metric-value ' + MercantileUI.Utils.getPnLClass(data.unrealized_pnl);
        realizedEl.className = 'metric-value ' + MercantileUI.Utils.getPnLClass(data.realized_pnl);
        
        // Load positions
        if (data.positions && data.positions.length > 0) {
            renderPositions(data.positions);
        } else {
            loadExchangePositions();
        }
    } catch (error) {
        console.error('Error loading portfolio data:', error);
        document.getElementById('positions-content').innerHTML = '<div class="error">Error loading portfolio data</div>';
    }
}

async function loadExchangePositions() {
    try {
        const data = await MercantileUI.API.exchanges.positions('bybit');
        
        if (data.positions && data.positions.length > 0) {
            renderPositions(data.positions);
        } else {
            document.getElementById('positions-content').innerHTML = '<div class="empty-state">No active positions</div>';
        }
    } catch (error) {
        console.error('Error loading exchange positions:', error);
        document.getElementById('positions-content').innerHTML = '<div class="error">Error loading positions</div>';
    }
}

function renderPositions(positions) {
    const tableHtml = '<table class="data-table">' +
        '<thead>' +
            '<tr>' +
                '<th>Symbol</th>' +
                '<th>Quantity</th>' +
                '<th>Avg Price</th>' +
                '<th>Current Price</th>' +
                '<th>Unrealized P&L</th>' +
                '<th>Side</th>' +
            '</tr>' +
        '</thead>' +
        '<tbody>' +
            positions.map(pos => '<tr>' +
                '<td>' + pos.symbol + '</td>' +
                '<td>' + MercantileUI.Utils.formatNumber(pos.quantity) + '</td>' +
                '<td>' + MercantileUI.Utils.formatCurrency(pos.average_price) + '</td>' +
                '<td>' + MercantileUI.Utils.formatCurrency(pos.current_price) + '</td>' +
                '<td class="' + MercantileUI.Utils.getPnLClass(pos.unrealized_pnl) + '">' +
                    MercantileUI.Utils.formatCurrency(pos.unrealized_pnl) +
                '</td>' +
                '<td>' + (pos.side || 'long') + '</td>' +
            '</tr>').join('') +
        '</tbody>' +
    '</table>';
    
    document.getElementById('positions-content').innerHTML = tableHtml;
}

// Initialize portfolio page
MercantileUI.AutoRefresh.start('portfolio', loadPortfolioData, 30000);
</script>
{{end}}
`

const settingsTemplate = baseTemplate + `
{{define "content"}}
<div class="page-content">
    <div class="card">
        <h2>Settings</h2>
        <p>Configuration options coming soon...</p>
    </div>
</div>
{{end}}
`

const strategyDetailsTemplate = baseTemplate + `
{{define "content"}}
<div class="page-content">
    <!-- Back Button -->
    <div class="mb-3">
        <a href="/strategies" class="btn btn-secondary">
            ← Back to Strategies
        </a>
    </div>

    <!-- Strategy Header -->
    <div class="card">
        <div class="d-flex justify-content-between align-items-center">
            <div>
                <h2 id="strategy-name">Loading...</h2>
                <p class="text-muted" id="strategy-info">Loading strategy details...</p>
            </div>
            <div class="strategy-actions" id="strategy-actions">
                <div class="loading">Loading...</div>
            </div>
        </div>
    </div>

    <!-- Strategy Metrics -->
    <div class="metrics-grid">
        <div class="metric-card">
            <div class="metric-value" id="strategy-status">-</div>
            <div class="metric-label">Status</div>
        </div>
        <div class="metric-card">
            <div class="metric-value" id="strategy-pnl">-</div>
            <div class="metric-label">P&L</div>
        </div>
        <div class="metric-card">
            <div class="metric-value" id="total-orders">-</div>
            <div class="metric-label">Total Orders</div>
        </div>
        <div class="metric-card">
            <div class="metric-value" id="success-rate">-</div>
            <div class="metric-label">Success Rate</div>
        </div>
    </div>

    <div class="grid">
        <!-- Strategy Statistics -->
        <div class="card">
            <h3>Statistics</h3>
            <div id="strategy-stats">
                <div class="loading">Loading statistics...</div>
            </div>
        </div>

        <!-- Recent Orders -->
        <div class="card">
            <h3>Recent Orders</h3>
            <div id="recent-orders">
                <div class="loading">Loading orders...</div>
            </div>
        </div>
    </div>

    <!-- Recent Logs -->
    <div class="card">
        <h3>Recent Logs</h3>
        <div id="recent-logs">
            <div class="loading">Loading logs...</div>
        </div>
    </div>
</div>
{{end}}

{{define "scripts"}}
<script>
let strategyId = null;

async function loadStrategyDetails() {
    // Get strategy ID from URL
    const pathParts = window.location.pathname.split('/');
    strategyId = pathParts[pathParts.length - 1];
    
    if (!strategyId) {
        document.getElementById('strategy-name').textContent = 'Strategy Not Found';
        document.getElementById('strategy-info').textContent = 'Invalid strategy ID';
        return;
    }

    try {
        const data = await MercantileUI.API.strategies.get(strategyId);
        displayStrategyDetails(data);
    } catch (error) {
        console.error('Error loading strategy details:', error);
        document.getElementById('strategy-name').textContent = 'Error Loading Strategy';
        document.getElementById('strategy-info').textContent = 'Failed to load strategy details: ' + error.message;
    }
}

function displayStrategyDetails(strategy) {
    // Update header
    document.getElementById('strategy-name').textContent = strategy.name || 'Unknown Strategy';
    document.getElementById('strategy-info').textContent = 
        (strategy.symbol || 'N/A') + ' • ' + (strategy.exchange || 'N/A');

    // Update metrics
    const statusEl = document.getElementById('strategy-status');
    statusEl.textContent = strategy.status || 'Unknown';
    statusEl.className = 'metric-value ' + MercantileUI.Utils.getStatusClass(strategy.status);

    const pnlEl = document.getElementById('strategy-pnl');
    const pnlValue = parseFloat(strategy.pnl?.replace(/[^-\d.]/g, '') || '0');
    pnlEl.textContent = strategy.pnl || '$0.00';
    pnlEl.className = 'metric-value ' + MercantileUI.Utils.getPnLClass(pnlValue);

    // Update statistics
    if (strategy.stats) {
        document.getElementById('total-orders').textContent = strategy.stats.total_orders || 0;
        document.getElementById('success-rate').textContent = 
            (strategy.stats.success_rate || 0).toFixed(1) + '%';
        
        displayStatistics(strategy.stats);
    }

    // Update actions
    const actions = strategy.status === 'running' 
        ? '<button class="btn btn-warning" onclick="stopStrategy()">Stop</button> ' +
          '<button class="btn btn-secondary" onclick="restartStrategy()">Restart</button>'
        : '<button class="btn btn-success" onclick="startStrategy()">Start</button>';
    
    document.getElementById('strategy-actions').innerHTML = actions;

    // Display recent orders
    if (strategy.recent_orders) {
        displayRecentOrders(strategy.recent_orders);
    } else {
        document.getElementById('recent-orders').innerHTML = '<div class="empty-state">No recent orders</div>';
    }

    // Display recent logs
    if (strategy.recent_logs) {
        displayRecentLogs(strategy.recent_logs);
    } else {
        document.getElementById('recent-logs').innerHTML = '<div class="empty-state">No recent logs</div>';
    }
}

function displayStatistics(stats) {
    const statsHtml = '<div class="stats-grid">' +
        '<div class="stat-item">' +
            '<div class="stat-value">' + (stats.buy_orders || 0) + '</div>' +
            '<div class="stat-label">Buy Orders</div>' +
        '</div>' +
        '<div class="stat-item">' +
            '<div class="stat-value">' + (stats.sell_orders || 0) + '</div>' +
            '<div class="stat-label">Sell Orders</div>' +
        '</div>' +
        '<div class="stat-item">' +
            '<div class="stat-value">' + MercantileUI.Utils.formatCurrency(stats.total_volume || 0) + '</div>' +
            '<div class="stat-label">Total Volume</div>' +
        '</div>' +
        '<div class="stat-item">' +
            '<div class="stat-value">' + MercantileUI.Utils.formatNumber(stats.avg_order_size || 0, 4) + '</div>' +
            '<div class="stat-label">Avg Order Size</div>' +
        '</div>' +
    '</div>';
    
    document.getElementById('strategy-stats').innerHTML = statsHtml;
}

function displayRecentOrders(orders) {
    if (!orders || orders.length === 0) {
        document.getElementById('recent-orders').innerHTML = '<div class="empty-state">No recent orders</div>';
        return;
    }

    const tableHtml = '<table class="data-table">' +
        '<thead>' +
            '<tr>' +
                '<th>Time</th>' +
                '<th>Side</th>' +
                '<th>Type</th>' +
                '<th>Quantity</th>' +
                '<th>Price</th>' +
                '<th>Status</th>' +
            '</tr>' +
        '</thead>' +
        '<tbody>' +
            orders.map(order => {
                const time = new Date(order.created_at).toLocaleTimeString();
                const sideClass = order.side === 'buy' ? 'text-success' : 'text-danger';
                const statusClass = MercantileUI.Utils.getStatusClass(order.status);
                
                return '<tr>' +
                    '<td>' + time + '</td>' +
                    '<td class="' + sideClass + '">' + order.side.toUpperCase() + '</td>' +
                    '<td>' + order.type + '</td>' +
                    '<td>' + MercantileUI.Utils.formatNumber(order.quantity, 4) + '</td>' +
                    '<td>' + MercantileUI.Utils.formatCurrency(order.price) + '</td>' +
                    '<td class="' + statusClass + '">' + order.status + '</td>' +
                '</tr>';
            }).join('') +
        '</tbody>' +
    '</table>';
    
    document.getElementById('recent-orders').innerHTML = tableHtml;
}

function displayRecentLogs(logs) {
    if (!logs || logs.length === 0) {
        document.getElementById('recent-logs').innerHTML = '<div class="empty-state">No recent logs</div>';
        return;
    }

    const logsHtml = '<div class="logs-container">' +
        logs.map(log => {
            const time = new Date(log.timestamp).toLocaleTimeString();
            const levelClass = 'log-' + log.level.toLowerCase();
            
            return '<div class="log-entry">' +
                '<span class="log-time">' + time + '</span> ' +
                '<span class="log-level ' + levelClass + '">' + log.level + '</span> ' +
                '<span class="log-message">' + log.message + '</span>' +
            '</div>';
        }).join('') +
    '</div>';
    
    document.getElementById('recent-logs').innerHTML = logsHtml;
}

async function startStrategy() {
    try {
        await MercantileUI.API.strategies.start(strategyId);
        MercantileUI.Strategy.showNotification('Strategy started successfully', 'success');
        loadStrategyDetails(); // Refresh
    } catch (error) {
        MercantileUI.Strategy.showNotification('Failed to start strategy: ' + error.message, 'error');
    }
}

async function stopStrategy() {
    try {
        await MercantileUI.API.strategies.stop(strategyId);
        MercantileUI.Strategy.showNotification('Strategy stopped successfully', 'success');
        loadStrategyDetails(); // Refresh
    } catch (error) {
        MercantileUI.Strategy.showNotification('Failed to stop strategy: ' + error.message, 'error');
    }
}

async function restartStrategy() {
    try {
        await MercantileUI.API.strategies.restart(strategyId);
        MercantileUI.Strategy.showNotification('Strategy restarted successfully', 'success');
        loadStrategyDetails(); // Refresh
    } catch (error) {
        MercantileUI.Strategy.showNotification('Failed to restart strategy: ' + error.message, 'error');
    }
}

// Initialize strategy details page
MercantileUI.AutoRefresh.start('strategyDetails', loadStrategyDetails, 30000);
</script>
{{end}}
`

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
	r.Get("/strategies/{id}", u.handleStrategyDetails)
	r.Get("/portfolio", u.handlePortfolio)
	r.Get("/settings", u.handleSettings)

	u.router = r
}

// Helper function to render templates
func (u *UIActor) renderTemplate(w http.ResponseWriter, templateStr string, data BasePageData) {
	tmpl, err := template.New("page").Funcs(templateFuncs).Parse(templateStr)
	if err != nil {
		u.logger.Error().Err(err).Msg("Template parsing error")
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if err := tmpl.Execute(w, data); err != nil {
		u.logger.Error().Err(err).Msg("Template execution error")
		http.Error(w, "Template error", http.StatusInternalServerError)
	}
}

func (u *UIActor) handleIndex(w http.ResponseWriter, r *http.Request) {
	data := BasePageData{
		Title:       "Home",
		CurrentPage: "/",
		Subtitle:    "Trading Bot",
	}
	u.renderTemplate(w, indexTemplate, data)
}

func (u *UIActor) handleDashboard(w http.ResponseWriter, r *http.Request) {
	data := BasePageData{
		Title:       "Dashboard",
		CurrentPage: "/dashboard",
		Subtitle:    "Trading Dashboard",
	}
	u.renderTemplate(w, dashboardTemplate, data)
}

func (u *UIActor) handleStrategies(w http.ResponseWriter, r *http.Request) {
	data := BasePageData{
		Title:       "Strategies",
		CurrentPage: "/strategies",
		Subtitle:    "Strategy Management",
	}
	u.renderTemplate(w, strategiesTemplate, data)
}

func (u *UIActor) handleStrategyDetails(w http.ResponseWriter, r *http.Request) {
	data := BasePageData{
		Title:       "Strategy Details",
		CurrentPage: "/strategies",
		Subtitle:    "Strategy Details",
	}
	u.renderTemplate(w, strategyDetailsTemplate, data)
}

func (u *UIActor) handlePortfolio(w http.ResponseWriter, r *http.Request) {
	data := BasePageData{
		Title:       "Portfolio",
		CurrentPage: "/portfolio",
		Subtitle:    "Portfolio Overview",
	}
	u.renderTemplate(w, portfolioTemplate, data)
}

func (u *UIActor) handleSettings(w http.ResponseWriter, r *http.Request) {
	data := BasePageData{
		Title:       "Settings",
		CurrentPage: "/settings",
		Subtitle:    "Configuration",
	}
	u.renderTemplate(w, settingsTemplate, data)
}
