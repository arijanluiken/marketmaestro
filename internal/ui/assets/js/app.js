/**
 * Mercantile Trading Bot - Shared JavaScript Components
 */

// API Configuration
const API_BASE_URL = 'http://localhost:8080/api/v1';

// Utility Functions
const Utils = {
    // Format currency values
    formatCurrency: (value, currency = 'USD') => {
        const formatter = new Intl.NumberFormat('en-US', {
            style: 'currency',
            currency: currency,
            minimumFractionDigits: 2,
            maximumFractionDigits: 2
        });
        return formatter.format(value);
    },

    // Format percentage values
    formatPercentage: (value, decimals = 2) => {
        return (value >= 0 ? '+' : '') + value.toFixed(decimals) + '%';
    },

    // Format large numbers
    formatNumber: (value, decimals = 2) => {
        return new Intl.NumberFormat('en-US', {
            minimumFractionDigits: decimals,
            maximumFractionDigits: decimals
        }).format(value);
    },

    // Get status class for styling
    getStatusClass: (status) => {
        switch (status.toLowerCase()) {
            case 'running':
            case 'active':
            case 'online':
                return 'status-running';
            case 'stopped':
            case 'inactive':
            case 'offline':
                return 'status-stopped';
            case 'error':
            case 'failed':
                return 'status-error';
            default:
                return 'text-muted';
        }
    },

    // Get P&L class for styling
    getPnLClass: (value) => {
        return value >= 0 ? 'pnl-positive' : 'pnl-negative';
    },

    // Debounce function for API calls
    debounce: (func, wait) => {
        let timeout;
        return function executedFunction(...args) {
            const later = () => {
                clearTimeout(timeout);
                func(...args);
            };
            clearTimeout(timeout);
            timeout = setTimeout(later, wait);
        };
    }
};

// API Client
const API = {
    // Generic fetch wrapper with error handling
    async fetch(endpoint, options = {}) {
        try {
            const response = await fetch(`${API_BASE_URL}${endpoint}`, {
                headers: {
                    'Content-Type': 'application/json',
                    ...options.headers
                },
                ...options
            });

            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }

            return await response.json();
        } catch (error) {
            console.error(`API Error (${endpoint}):`, error);
            throw error;
        }
    },

    // Health check
    health: () => API.fetch('/health'),

    // Exchanges
    exchanges: {
        list: () => API.fetch('/exchanges/'),
        get: (exchange) => API.fetch(`/exchanges/${exchange}`),
        positions: (exchange) => API.fetch(`/exchanges/${exchange}/positions`)
    },

    // Strategies
    strategies: {
        list: () => API.fetch('/strategies/'),
        get: (id) => API.fetch(`/strategies/${id}`),
        start: (id) => API.fetch(`/strategies/${id}/start`, { method: 'POST' }),
        stop: (id) => API.fetch(`/strategies/${id}/stop`, { method: 'POST' }),
        restart: (id) => API.fetch(`/strategies/${id}/restart`, { method: 'POST' })
    },

    // Portfolio
    portfolio: {
        get: () => API.fetch('/portfolio'),
        summary: () => API.fetch('/portfolio/summary')
    }
};

// Shared Components
const Components = {
    // Create loading indicator
    loading: (message = 'Loading...') => {
        return `<div class="loading">${message}</div>`;
    },

    // Create error message
    error: (message = 'An error occurred') => {
        return `<div class="error">${message}</div>`;
    },

    // Create empty state
    empty: (message = 'No data available') => {
        return `<div class="empty-state">${message}</div>`;
    },

    // Create navigation bar
    navbar: (currentPage = '') => {
        const navItems = [
            { path: '/', label: 'Home', icon: '🏠' },
            { path: '/dashboard', label: 'Dashboard', icon: '📊' },
            { path: '/strategies', label: 'Strategies', icon: '🤖' },
            { path: '/portfolio', label: 'Portfolio', icon: '💰' },
            { path: '/settings', label: 'Settings', icon: '⚙️' }
        ];

        const navHtml = navItems.map(item => {
            const isActive = currentPage === item.path ? 'active' : '';
            return `<li class="nav-item">
                <a href="${item.path}" class="nav-link ${isActive}">
                    <span class="nav-icon">${item.icon}</span>
                    <span class="nav-label">${item.label}</span>
                </a>
            </li>`;
        }).join('');

        return `
            <nav class="app-navigation">
                <ul class="nav-list">
                    ${navHtml}
                </ul>
            </nav>
        `;
    },

    // Create header component
    header: (title, subtitle = '', showStatus = true) => {
        const statusHtml = showStatus ? `
            <div class="header-status">
                <div class="status-indicator" id="connection-status">
                    <div class="status-dot"></div>
                    <span>Checking...</span>
                </div>
            </div>
        ` : '';

        return `
            <header class="app-header">
                <div class="header-content">
                    <div class="header-brand">
                        <h1>Mercantile</h1>
                        <div class="subtitle">${subtitle || 'Trading Bot'}</div>
                    </div>
                    ${statusHtml}
                </div>
            </header>
        `;
    },

    // Create metric card
    metricCard: (value, label, trend = null, format = 'number') => {
        let formattedValue = value;
        let trendClass = '';

        switch (format) {
            case 'currency':
                formattedValue = Utils.formatCurrency(value);
                break;
            case 'percentage':
                formattedValue = Utils.formatPercentage(value);
                break;
            case 'number':
                formattedValue = Utils.formatNumber(value);
                break;
        }

        if (trend !== null) {
            trendClass = Utils.getPnLClass(trend);
        }

        return `
            <div class="metric-card">
                <div class="metric-value ${trendClass}">${formattedValue}</div>
                <div class="metric-label">${label}</div>
            </div>
        `;
    },

    // Create strategy card
    strategyCard: (strategy) => {
        const statusClass = Utils.getStatusClass(strategy.status);
        const pnlClass = Utils.getPnLClass(parseFloat(strategy.pnl?.replace(/[^-\d.]/g, '') || 0));
        
        const actions = strategy.status === 'running' 
            ? `<button class="btn btn-warning btn-sm" onclick="Strategy.stop('${strategy.id}')">Stop</button>
               <button class="btn btn-secondary btn-sm" onclick="Strategy.restart('${strategy.id}')">Restart</button>`
            : `<button class="btn btn-success btn-sm" onclick="Strategy.start('${strategy.id}')">Start</button>`;

        return `
            <div class="card">
                <div class="d-flex justify-content-between align-items-center">
                    <div>
                        <h4 class="mb-1">${strategy.name}</h4>
                        <p class="text-muted mb-2">${strategy.symbol} • ${strategy.exchange}</p>
                        <div class="d-flex gap-2">
                            <span class="badge ${statusClass}">${strategy.status}</span>
                            <span class="badge ${pnlClass}">${strategy.pnl || '$0.00'}</span>
                        </div>
                    </div>
                    <div class="text-right">
                        ${actions}
                    </div>
                </div>
            </div>
        `;
    }
};

// Strategy Management
const Strategy = {
    async start(strategyId) {
        try {
            await API.strategies.start(strategyId);
            this.showNotification('Strategy started successfully', 'success');
            // Refresh the page data
            if (typeof loadStrategies === 'function') {
                loadStrategies();
            }
        } catch (error) {
            this.showNotification('Failed to start strategy: ' + error.message, 'error');
        }
    },

    async stop(strategyId) {
        try {
            await API.strategies.stop(strategyId);
            this.showNotification('Strategy stopped successfully', 'success');
            // Refresh the page data
            if (typeof loadStrategies === 'function') {
                loadStrategies();
            }
        } catch (error) {
            this.showNotification('Failed to stop strategy: ' + error.message, 'error');
        }
    },

    async restart(strategyId) {
        try {
            await API.strategies.restart(strategyId);
            this.showNotification('Strategy restarted successfully', 'success');
            // Refresh the page data
            if (typeof loadStrategies === 'function') {
                loadStrategies();
            }
        } catch (error) {
            this.showNotification('Failed to restart strategy: ' + error.message, 'error');
        }
    },

    showNotification(message, type = 'info') {
        // Simple notification system
        const notification = document.createElement('div');
        notification.className = `notification notification-${type}`;
        notification.textContent = message;
        notification.style.cssText = `
            position: fixed;
            top: 20px;
            right: 20px;
            padding: 12px 20px;
            border-radius: 6px;
            color: white;
            font-weight: 500;
            z-index: 1000;
            background: ${type === 'success' ? '#28a745' : type === 'error' ? '#dc3545' : '#007bff'};
        `;

        document.body.appendChild(notification);

        setTimeout(() => {
            notification.remove();
        }, 3000);
    }
};

// Connection Status Management
const ConnectionStatus = {
    async check() {
        try {
            await API.health();
            this.updateStatus('online', 'Connected');
        } catch (error) {
            this.updateStatus('offline', 'Disconnected');
        }
    },

    updateStatus(status, text) {
        const statusElement = document.getElementById('connection-status');
        if (statusElement) {
            statusElement.className = `status-indicator ${status}`;
            statusElement.querySelector('span').textContent = text;
        }
    },

    init() {
        // Check status immediately
        this.check();
        
        // Check every 30 seconds
        setInterval(() => this.check(), 30000);
    }
};

// Auto-refresh Management
const AutoRefresh = {
    intervals: new Map(),

    start(key, callback, interval = 30000) {
        // Clear existing interval if any
        this.stop(key);
        
        // Run callback immediately
        callback();
        
        // Set up interval
        const intervalId = setInterval(callback, interval);
        this.intervals.set(key, intervalId);
    },

    stop(key) {
        const intervalId = this.intervals.get(key);
        if (intervalId) {
            clearInterval(intervalId);
            this.intervals.delete(key);
        }
    },

    stopAll() {
        this.intervals.forEach(intervalId => clearInterval(intervalId));
        this.intervals.clear();
    }
};

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    // Initialize connection status checking
    ConnectionStatus.init();
    
    // Clean up intervals when page unloads
    window.addEventListener('beforeunload', () => {
        AutoRefresh.stopAll();
    });
});

// Global error handler
window.addEventListener('error', (event) => {
    console.error('Global error:', event.error);
});

// Export for use in other scripts
window.MercantileUI = {
    Utils,
    API,
    Components,
    Strategy,
    ConnectionStatus,
    AutoRefresh
};