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

// calculateWilliamsR calculates Williams %R oscillator
func (ti *TechnicalIndicators) calculateWilliamsR(high, low, close *starlark.List, period int) []float64 {
	length := close.Len()
	result := make([]float64, length)

	for i := 0; i < length; i++ {
		if i < period-1 {
			result[i] = math.NaN()
			continue
		}

		// Find highest high and lowest low in period
		highestHigh := math.Inf(-1)
		lowestLow := math.Inf(1)

		for j := i - period + 1; j <= i; j++ {
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
			result[i] = -50 // Neutral value when no range
		} else {
			result[i] = ((highestHigh - c) / (highestHigh - lowestLow)) * -100
		}
	}

	return result
}

// calculateATR calculates Average True Range
func (ti *TechnicalIndicators) calculateATR(high, low, close *starlark.List, period int) []float64 {
	length := close.Len()
	if length < 2 {
		return nil
	}

	trueRanges := make([]float64, length)
	result := make([]float64, length)

	// Calculate True Range for each period
	for i := 0; i < length; i++ {
		if i == 0 {
			trueRanges[i] = math.NaN()
			result[i] = math.NaN()
			continue
		}

		h, _ := starlark.AsFloat(high.Index(i))
		l, _ := starlark.AsFloat(low.Index(i))
		c, _ := starlark.AsFloat(close.Index(i-1))

		tr1 := h - l
		tr2 := math.Abs(h - c)
		tr3 := math.Abs(l - c)

		trueRanges[i] = math.Max(tr1, math.Max(tr2, tr3))

		if i < period {
			result[i] = math.NaN()
			continue
		}

		// Calculate ATR using Simple Moving Average of True Range
		if i == period {
			// First ATR is simple average
			sum := 0.0
			for j := 1; j <= period; j++ {
				sum += trueRanges[j]
			}
			result[i] = sum / float64(period)
		} else {
			// Subsequent ATRs use smoothed average
			result[i] = ((result[i-1] * float64(period-1)) + trueRanges[i]) / float64(period)
		}
	}

	return result
}

// calculateCCI calculates Commodity Channel Index
func (ti *TechnicalIndicators) calculateCCI(high, low, close *starlark.List, period int) []float64 {
	length := close.Len()
	result := make([]float64, length)

	for i := 0; i < length; i++ {
		if i < period-1 {
			result[i] = math.NaN()
			continue
		}

		// Calculate typical prices for the period
		typicalPrices := make([]float64, period)
		for j := 0; j < period; j++ {
			idx := i - period + 1 + j
			h, _ := starlark.AsFloat(high.Index(idx))
			l, _ := starlark.AsFloat(low.Index(idx))
			c, _ := starlark.AsFloat(close.Index(idx))
			typicalPrices[j] = (h + l + c) / 3
		}

		// Calculate Simple Moving Average of typical prices
		smaTP := 0.0
		for _, tp := range typicalPrices {
			smaTP += tp
		}
		smaTP /= float64(period)

		// Calculate Mean Deviation
		meanDev := 0.0
		for _, tp := range typicalPrices {
			meanDev += math.Abs(tp - smaTP)
		}
		meanDev /= float64(period)

		// Calculate current typical price
		h, _ := starlark.AsFloat(high.Index(i))
		l, _ := starlark.AsFloat(low.Index(i))
		c, _ := starlark.AsFloat(close.Index(i))
		currentTP := (h + l + c) / 3

		// Calculate CCI
		if meanDev == 0 {
			result[i] = 0
		} else {
			result[i] = (currentTP - smaTP) / (0.015 * meanDev)
		}
	}

	return result
}

// calculateVWAP calculates Volume Weighted Average Price
func (ti *TechnicalIndicators) calculateVWAP(high, low, close, volume *starlark.List) []float64 {
	length := close.Len()
	result := make([]float64, length)

	cumulativeTPV := 0.0 // Cumulative Typical Price * Volume
	cumulativeVolume := 0.0

	for i := 0; i < length; i++ {
		h, _ := starlark.AsFloat(high.Index(i))
		l, _ := starlark.AsFloat(low.Index(i))
		c, _ := starlark.AsFloat(close.Index(i))
		v, _ := starlark.AsFloat(volume.Index(i))

		typicalPrice := (h + l + c) / 3
		cumulativeTPV += typicalPrice * v
		cumulativeVolume += v

		if cumulativeVolume == 0 {
			result[i] = math.NaN()
		} else {
			result[i] = cumulativeTPV / cumulativeVolume
		}
	}

	return result
}

// calculateMFI calculates Money Flow Index (volume-weighted RSI)
func (ti *TechnicalIndicators) calculateMFI(high, low, close, volume *starlark.List, period int) []float64 {
	length := close.Len()
	if length < period+1 {
		return nil
	}

	result := make([]float64, length)
	moneyFlows := make([]float64, length)
	typicalPrices := make([]float64, length)

	// Calculate typical prices and money flows
	for i := 0; i < length; i++ {
		h, _ := starlark.AsFloat(high.Index(i))
		l, _ := starlark.AsFloat(low.Index(i))
		c, _ := starlark.AsFloat(close.Index(i))
		v, _ := starlark.AsFloat(volume.Index(i))

		typicalPrices[i] = (h + l + c) / 3

		if i == 0 {
			moneyFlows[i] = 0
			result[i] = math.NaN()
		} else {
			rawMoneyFlow := typicalPrices[i] * v
			if typicalPrices[i] > typicalPrices[i-1] {
				moneyFlows[i] = rawMoneyFlow // Positive money flow
			} else if typicalPrices[i] < typicalPrices[i-1] {
				moneyFlows[i] = -rawMoneyFlow // Negative money flow
			} else {
				moneyFlows[i] = 0 // No change
			}
		}
	}

	// Calculate MFI for each period
	for i := period; i < length; i++ {
		positiveFlow := 0.0
		negativeFlow := 0.0

		for j := i - period + 1; j <= i; j++ {
			if moneyFlows[j] > 0 {
				positiveFlow += moneyFlows[j]
			} else if moneyFlows[j] < 0 {
				negativeFlow += -moneyFlows[j]
			}
		}

		if negativeFlow == 0 {
			result[i] = 100
		} else {
			moneyRatio := positiveFlow / negativeFlow
			result[i] = 100 - (100 / (1 + moneyRatio))
		}
	}

	// Fill early values with NaN
	for i := 0; i < period; i++ {
		result[i] = math.NaN()
	}

	return result
}

// calculateStdDev calculates Standard Deviation
func (ti *TechnicalIndicators) calculateStdDev(prices *starlark.List, period int) []float64 {
	length := prices.Len()
	result := make([]float64, length)

	for i := 0; i < length; i++ {
		if i < period-1 {
			result[i] = math.NaN()
			continue
		}

		// Calculate mean
		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			price, _ := starlark.AsFloat(prices.Index(j))
			sum += price
		}
		mean := sum / float64(period)

		// Calculate variance
		variance := 0.0
		for j := i - period + 1; j <= i; j++ {
			price, _ := starlark.AsFloat(prices.Index(j))
			variance += math.Pow(price-mean, 2)
		}
		variance /= float64(period)

		result[i] = math.Sqrt(variance)
	}

	return result
}

// calculateROC calculates Rate of Change
func (ti *TechnicalIndicators) calculateROC(prices *starlark.List, period int) []float64 {
	length := prices.Len()
	result := make([]float64, length)

	for i := 0; i < length; i++ {
		if i < period {
			result[i] = math.NaN()
			continue
		}

		currentPrice, _ := starlark.AsFloat(prices.Index(i))
		pastPrice, _ := starlark.AsFloat(prices.Index(i-period))

		if pastPrice == 0 {
			result[i] = math.NaN()
		} else {
			result[i] = ((currentPrice - pastPrice) / pastPrice) * 100
		}
	}

	return result
}
