package strategy

import (
	"math"

	"go.starlark.net/starlark"
)

// calculateSMA calculates Simple Moving Average
func (ti *TechnicalIndicators) calculateSMA(prices *starlark.List, period int) []float64 {
	length := prices.Len()
	if length < period {
		return nil
	}

	result := make([]float64, length)

	for i := 0; i < length; i++ {
		if i < period-1 {
			result[i] = math.NaN()
			continue
		}

		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			price, _ := starlark.AsFloat(prices.Index(j))
			sum += price
		}
		result[i] = sum / float64(period)
	}

	return result
}

// calculateEMA calculates Exponential Moving Average
func (ti *TechnicalIndicators) calculateEMA(prices *starlark.List, period int) []float64 {
	length := prices.Len()
	if length == 0 {
		return nil
	}

	result := make([]float64, length)
	multiplier := 2.0 / (float64(period) + 1.0)

	// First value is just the price
	firstPrice, _ := starlark.AsFloat(prices.Index(0))
	result[0] = firstPrice

	for i := 1; i < length; i++ {
		price, _ := starlark.AsFloat(prices.Index(i))
		result[i] = (price * multiplier) + (result[i-1] * (1.0 - multiplier))
	}

	return result
}

// calculateRSI calculates Relative Strength Index
func (ti *TechnicalIndicators) calculateRSI(prices *starlark.List, period int) []float64 {
	length := prices.Len()
	if length < period+1 {
		return nil
	}

	result := make([]float64, length)
	gains := make([]float64, length)
	losses := make([]float64, length)

	// Calculate price changes
	for i := 1; i < length; i++ {
		prevPrice, _ := starlark.AsFloat(prices.Index(i - 1))
		currPrice, _ := starlark.AsFloat(prices.Index(i))
		change := currPrice - prevPrice

		if change > 0 {
			gains[i] = change
		} else {
			losses[i] = -change
		}
	}

	// Calculate initial averages
	avgGain := 0.0
	avgLoss := 0.0
	for i := 1; i <= period; i++ {
		avgGain += gains[i]
		avgLoss += losses[i]
	}
	avgGain /= float64(period)
	avgLoss /= float64(period)

	// Calculate RSI
	for i := 0; i < length; i++ {
		if i < period {
			result[i] = math.NaN()
			continue
		}

		if i == period {
			// Use initial averages
		} else {
			// Use smoothed averages
			avgGain = ((avgGain * float64(period-1)) + gains[i]) / float64(period)
			avgLoss = ((avgLoss * float64(period-1)) + losses[i]) / float64(period)
		}

		if avgLoss == 0 {
			result[i] = 100
		} else {
			rs := avgGain / avgLoss
			result[i] = 100 - (100 / (1 + rs))
		}
	}

	return result
}

// calculateMACD calculates MACD (Moving Average Convergence Divergence)
func (ti *TechnicalIndicators) calculateMACD(prices *starlark.List, fastPeriod, slowPeriod, signalPeriod int) ([]float64, []float64, []float64) {
	length := prices.Len()

	// Calculate EMAs
	fastEMA := ti.calculateEMA(prices, fastPeriod)
	slowEMA := ti.calculateEMA(prices, slowPeriod)

	// Calculate MACD line
	macdLine := make([]float64, length)
	for i := 0; i < length; i++ {
		if i < slowPeriod-1 {
			macdLine[i] = math.NaN()
		} else {
			macdLine[i] = fastEMA[i] - slowEMA[i]
		}
	}

	// Convert MACD line to starlark list for signal calculation
	macdList := starlark.NewList(nil)
	for _, val := range macdLine {
		if math.IsNaN(val) {
			macdList.Append(starlark.None)
		} else {
			macdList.Append(starlark.Float(val))
		}
	}

	// Calculate signal line (EMA of MACD)
	signalLine := ti.calculateEMA(macdList, signalPeriod)

	// Calculate histogram
	histogram := make([]float64, length)
	for i := 0; i < length; i++ {
		if math.IsNaN(macdLine[i]) || math.IsNaN(signalLine[i]) {
			histogram[i] = math.NaN()
		} else {
			histogram[i] = macdLine[i] - signalLine[i]
		}
	}

	return macdLine, signalLine, histogram
}

// calculateBollinger calculates Bollinger Bands
func (ti *TechnicalIndicators) calculateBollinger(prices *starlark.List, period int, multiplier float64) ([]float64, []float64, []float64) {
	length := prices.Len()
	middle := ti.calculateSMA(prices, period)
	upper := make([]float64, length)
	lower := make([]float64, length)

	for i := 0; i < length; i++ {
		if i < period-1 {
			upper[i] = math.NaN()
			lower[i] = math.NaN()
			continue
		}

		// Calculate standard deviation
		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			price, _ := starlark.AsFloat(prices.Index(j))
			sum += math.Pow(price-middle[i], 2)
		}
		stdDev := math.Sqrt(sum / float64(period))

		upper[i] = middle[i] + (multiplier * stdDev)
		lower[i] = middle[i] - (multiplier * stdDev)
	}

	return upper, middle, lower
}

// calculateStochastic calculates Stochastic Oscillator
func (ti *TechnicalIndicators) calculateStochastic(high, low, close *starlark.List, kPeriod, dPeriod int) ([]float64, []float64) {
	length := close.Len()
	k := make([]float64, length)

	for i := 0; i < length; i++ {
		if i < kPeriod-1 {
			k[i] = math.NaN()
			continue
		}

		// Find highest high and lowest low in period
		highestHigh := math.Inf(-1)
		lowestLow := math.Inf(1)

		for j := i - kPeriod + 1; j <= i; j++ {
			h, _ := starlark.AsFloat(high.Index(j))
			l, _ := starlark.AsFloat(low.Index(j))

			if h > highestHigh {
				highestHigh = h
			}
			if l < lowestLow {
				lowestLow = l
			}
		}

		c, _ := starlark.AsFloat(close.Index(i))
		if highestHigh == lowestLow {
			k[i] = 50
		} else {
			k[i] = ((c - lowestLow) / (highestHigh - lowestLow)) * 100
		}
	}

	// Convert K to starlark list for D calculation
	kList := starlark.NewList(nil)
	for _, val := range k {
		if math.IsNaN(val) {
			kList.Append(starlark.None)
		} else {
			kList.Append(starlark.Float(val))
		}
	}

	d := ti.calculateSMA(kList, dPeriod)

	return k, d
}

// calculateHighest calculates rolling highest values
func (ti *TechnicalIndicators) calculateHighest(prices *starlark.List, period int) []float64 {
	length := prices.Len()
	result := make([]float64, length)

	for i := 0; i < length; i++ {
		if i < period-1 {
			result[i] = math.NaN()
			continue
		}

		highest := math.Inf(-1)
		for j := i - period + 1; j <= i; j++ {
			price, _ := starlark.AsFloat(prices.Index(j))
			if price > highest {
				highest = price
			}
		}
		result[i] = highest
	}

	return result
}

// calculateLowest calculates rolling lowest values
func (ti *TechnicalIndicators) calculateLowest(prices *starlark.List, period int) []float64 {
	length := prices.Len()
	result := make([]float64, length)

	for i := 0; i < length; i++ {
		if i < period-1 {
			result[i] = math.NaN()
			continue
		}

		lowest := math.Inf(1)
		for j := i - period + 1; j <= i; j++ {
			price, _ := starlark.AsFloat(prices.Index(j))
			if price < lowest {
				lowest = price
			}
		}
		result[i] = lowest
	}

	return result
}

// calculateCrossover detects when series1 crosses over series2
func (ti *TechnicalIndicators) calculateCrossover(series1, series2 *starlark.List) []bool {
	length := series1.Len()
	if length != series2.Len() {
		return nil
	}

	result := make([]bool, length)

	for i := 1; i < length; i++ {
		prev1, _ := starlark.AsFloat(series1.Index(i - 1))
		curr1, _ := starlark.AsFloat(series1.Index(i))
		prev2, _ := starlark.AsFloat(series2.Index(i - 1))
		curr2, _ := starlark.AsFloat(series2.Index(i))

		result[i] = prev1 <= prev2 && curr1 > curr2
	}

	return result
}

// calculateCrossunder detects when series1 crosses under series2
func (ti *TechnicalIndicators) calculateCrossunder(series1, series2 *starlark.List) []bool {
	length := series1.Len()
	if length != series2.Len() {
		return nil
	}

	result := make([]bool, length)

	for i := 1; i < length; i++ {
		prev1, _ := starlark.AsFloat(series1.Index(i - 1))
		curr1, _ := starlark.AsFloat(series1.Index(i))
		prev2, _ := starlark.AsFloat(series2.Index(i - 1))
		curr2, _ := starlark.AsFloat(series2.Index(i))

		result[i] = prev1 >= prev2 && curr1 < curr2
	}

	return result
}
