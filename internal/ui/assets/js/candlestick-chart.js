// Candlestick Chart Implementation for Mercantile Trading Bot
// Uses Chart.js with custom drawing plugin for candlestick visualization

// Function to create a candlestick chart
function createCandlestickChart(ctx, klines, symbol) {
    if (!klines || klines.length === 0) {
        console.error('No kline data provided');
        return null;
    }

    // Prepare data for line chart
    const chartData = klines.map(kline => ({
        x: new Date(kline.timestamp),
        y: kline.close
    }));

    // Store original kline data for candlestick drawing
    const candlestickData = klines.map(kline => ({
        x: new Date(kline.timestamp),
        open: parseFloat(kline.open),
        high: parseFloat(kline.high),
        low: parseFloat(kline.low),
        close: parseFloat(kline.close),
        volume: parseFloat(kline.volume || 0)
    }));

    const chart = new Chart(ctx, {
        type: 'line',
        data: {
            datasets: [{
                label: symbol,
                data: chartData,
                borderColor: '#00d4aa',
                backgroundColor: 'rgba(0, 212, 170, 0.1)',
                borderWidth: 1,
                pointRadius: 0,
                tension: 0
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            interaction: {
                mode: 'index',
                intersect: false
            },
            scales: {
                x: {
                    type: 'time',
                    time: {
                        unit: 'minute',
                        displayFormats: {
                            minute: 'HH:mm',
                            hour: 'HH:mm',
                            day: 'MMM dd'
                        }
                    },
                    title: {
                        display: true,
                        text: 'Time',
                        color: '#212529'
                    },
                    grid: {
                        color: 'rgba(0, 0, 0, 0.1)'
                    },
                    ticks: {
                        color: '#212529'
                    }
                },
                y: {
                    title: {
                        display: true,
                        text: 'Price',
                        color: '#212529'
                    },
                    grid: {
                        color: 'rgba(0, 0, 0, 0.1)'
                    },
                    ticks: {
                        color: '#212529',
                        callback: function(value) {
                            return '$' + value.toFixed(2);
                        }
                    }
                }
            },
            plugins: {
                legend: {
                    labels: {
                        color: '#212529'
                    }
                },
                tooltip: {
                    backgroundColor: 'rgba(0, 0, 0, 0.8)',
                    titleColor: 'white',
                    bodyColor: 'white',
                    borderColor: '#ccc',
                    borderWidth: 1,
                    callbacks: {
                        title: function(context) {
                            return new Date(context[0].parsed.x).toLocaleString();
                        },
                        label: function(context) {
                            const dataIndex = context.dataIndex;
                            const candle = candlestickData[dataIndex];
                            if (candle) {
                                return [
                                    'Open: $' + candle.open.toFixed(2),
                                    'High: $' + candle.high.toFixed(2),
                                    'Low: $' + candle.low.toFixed(2),
                                    'Close: $' + candle.close.toFixed(2),
                                    'Volume: ' + candle.volume.toFixed(2)
                                ];
                            }
                            return 'Price: $' + context.parsed.y.toFixed(2);
                        }
                    }
                }
            }
        },
        plugins: [{
            id: 'candlestickOverlay',
            afterDatasetsDraw: function(chart) {
                const ctx = chart.ctx;
                const chartArea = chart.chartArea;
                
                if (!candlestickData || candlestickData.length === 0) return;
                
                ctx.save();
                
                candlestickData.forEach((candle, index) => {
                    const meta = chart.getDatasetMeta(0);
                    const point = meta.data[index];
                    
                    if (!point || point.x < chartArea.left || point.x > chartArea.right) return;
                    
                    const yScale = chart.scales.y;
                    
                    const x = point.x;
                    const openY = yScale.getPixelForValue(candle.open);
                    const highY = yScale.getPixelForValue(candle.high);
                    const lowY = yScale.getPixelForValue(candle.low);
                    const closeY = yScale.getPixelForValue(candle.close);
                    
                    const candleWidth = Math.max(2, (chartArea.right - chartArea.left) / candlestickData.length * 0.8);
                    
                    // Determine colors
                    const isGreen = candle.close >= candle.open;
                    const bodyColor = isGreen ? '#00d4aa' : '#ff4757';
                    const wickColor = isGreen ? '#00d4aa' : '#ff4757';
                    
                    // Draw wick (high-low line)
                    ctx.strokeStyle = wickColor;
                    ctx.lineWidth = 1;
                    ctx.beginPath();
                    ctx.moveTo(x, highY);
                    ctx.lineTo(x, lowY);
                    ctx.stroke();
                    
                    // Draw body (open-close rectangle)
                    ctx.fillStyle = bodyColor;
                    ctx.strokeStyle = bodyColor;
                    ctx.lineWidth = 1;
                    
                    const bodyTop = Math.min(openY, closeY);
                    const bodyHeight = Math.abs(closeY - openY);
                    
                    if (bodyHeight < 2) {
                        // For very small bodies (flat candles), draw a thicker line to make them visible
                        ctx.lineWidth = 2;
                        ctx.beginPath();
                        ctx.moveTo(x - candleWidth/2, openY);
                        ctx.lineTo(x + candleWidth/2, openY);
                        ctx.stroke();
                    } else {
                        // Draw rectangle body
                        ctx.fillRect(x - candleWidth/2, bodyTop, candleWidth, bodyHeight);
                        ctx.strokeRect(x - candleWidth/2, bodyTop, candleWidth, bodyHeight);
                    }
                });
                
                ctx.restore();
            }
        }]
    });

    // Store candlestick data on chart for updates
    chart.candlestickData = candlestickData;
    
    return chart;
}

// Function to update candlestick chart with new data
function updateCandlestickChart(chart, klines) {
    if (!chart || !klines || klines.length === 0) {
        console.error('Invalid chart or kline data for update');
        return;
    }

    // Update line chart data
    const chartData = klines.map(kline => ({
        x: new Date(kline.timestamp),
        y: parseFloat(kline.close)
    }));

    // Update candlestick data
    const candlestickData = klines.map(kline => ({
        x: new Date(kline.timestamp),
        open: parseFloat(kline.open),
        high: parseFloat(kline.high),
        low: parseFloat(kline.low),
        close: parseFloat(kline.close),
        volume: parseFloat(kline.volume || 0)
    }));

    chart.data.datasets[0].data = chartData;
    chart.candlestickData = candlestickData;
    chart.update('none'); // Update without animation for better performance
}

// Export functions for global use
window.createCandlestickChart = createCandlestickChart;
window.updateCandlestickChart = updateCandlestickChart;
