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
		c, _ := starlark.AsFloat(close.Index(i - 1))

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
		pastPrice, _ := starlark.AsFloat(prices.Index(i - period))

		if pastPrice == 0 {
			result[i] = math.NaN()
		} else {
			result[i] = ((currentPrice - pastPrice) / pastPrice) * 100
		}
	}

	return result
}

// calculateOBV calculates On-Balance Volume
func (ti *TechnicalIndicators) calculateOBV(close, volume *starlark.List) []float64 {
	length := close.Len()
	if length < 2 {
		return nil
	}

	result := make([]float64, length)
	result[0] = 0

	for i := 1; i < length; i++ {
		currPrice, _ := starlark.AsFloat(close.Index(i))
		prevPrice, _ := starlark.AsFloat(close.Index(i - 1))
		vol, _ := starlark.AsFloat(volume.Index(i))

		if currPrice > prevPrice {
			result[i] = result[i-1] + vol
		} else if currPrice < prevPrice {
			result[i] = result[i-1] - vol
		} else {
			result[i] = result[i-1]
		}
	}

	return result
}

// calculateADX calculates Average Directional Index
func (ti *TechnicalIndicators) calculateADX(high, low, close *starlark.List, period int) ([]float64, []float64, []float64) {
	length := close.Len()
	if length < period+1 {
		return nil, nil, nil
	}

	// Calculate True Range and Directional Movement
	tr := make([]float64, length)
	plusDM := make([]float64, length)
	minusDM := make([]float64, length)

	for i := 1; i < length; i++ {
		h1, _ := starlark.AsFloat(high.Index(i))
		l1, _ := starlark.AsFloat(low.Index(i))
		c1, _ := starlark.AsFloat(close.Index(i - 1))
		h0, _ := starlark.AsFloat(high.Index(i - 1))
		l0, _ := starlark.AsFloat(low.Index(i - 1))

		// True Range
		tr1 := h1 - l1
		tr2 := math.Abs(h1 - c1)
		tr3 := math.Abs(l1 - c1)
		tr[i] = math.Max(tr1, math.Max(tr2, tr3))

		// Directional Movement
		hDiff := h1 - h0
		lDiff := l0 - l1

		if hDiff > lDiff && hDiff > 0 {
			plusDM[i] = hDiff
		} else {
			plusDM[i] = 0
		}

		if lDiff > hDiff && lDiff > 0 {
			minusDM[i] = lDiff
		} else {
			minusDM[i] = 0
		}
	}

	// Smooth TR, +DM, -DM
	atr := make([]float64, length)
	smoothPlusDM := make([]float64, length)
	smoothMinusDM := make([]float64, length)

	// Initialize with sum of first period
	for i := 1; i <= period; i++ {
		atr[period] += tr[i]
		smoothPlusDM[period] += plusDM[i]
		smoothMinusDM[period] += minusDM[i]
	}

	// Smooth subsequent values
	for i := period + 1; i < length; i++ {
		atr[i] = atr[i-1] - (atr[i-1] / float64(period)) + tr[i]
		smoothPlusDM[i] = smoothPlusDM[i-1] - (smoothPlusDM[i-1] / float64(period)) + plusDM[i]
		smoothMinusDM[i] = smoothMinusDM[i-1] - (smoothMinusDM[i-1] / float64(period)) + minusDM[i]
	}

	// Calculate DI+ and DI-
	plusDI := make([]float64, length)
	minusDI := make([]float64, length)
	dx := make([]float64, length)

	for i := period; i < length; i++ {
		if atr[i] != 0 {
			plusDI[i] = (smoothPlusDM[i] / atr[i]) * 100
			minusDI[i] = (smoothMinusDM[i] / atr[i]) * 100

			diSum := plusDI[i] + minusDI[i]
			if diSum != 0 {
				dx[i] = (math.Abs(plusDI[i]-minusDI[i]) / diSum) * 100
			}
		}
	}

	// Calculate ADX
	adx := make([]float64, length)
	for i := 0; i < period*2-1; i++ {
		adx[i] = math.NaN()
		plusDI[i] = math.NaN()
		minusDI[i] = math.NaN()
	}

	// First ADX value is average of first period DX values
	sum := 0.0
	count := 0
	for i := period; i < period*2-1 && i < length; i++ {
		if !math.IsNaN(dx[i]) {
			sum += dx[i]
			count++
		}
	}
	if count > 0 && period*2-1 < length {
		adx[period*2-1] = sum / float64(count)
	}

	// Smooth ADX
	for i := period * 2; i < length; i++ {
		if !math.IsNaN(adx[i-1]) && !math.IsNaN(dx[i]) {
			adx[i] = ((adx[i-1] * float64(period-1)) + dx[i]) / float64(period)
		}
	}

	return adx, plusDI, minusDI
}

// calculateParabolicSAR calculates Parabolic Stop and Reverse
func (ti *TechnicalIndicators) calculateParabolicSAR(high, low *starlark.List, step, maxStep float64) []float64 {
	length := high.Len()
	if length < 2 {
		return nil
	}

	result := make([]float64, length)
	ep := make([]float64, length) // Extreme Point
	af := make([]float64, length) // Acceleration Factor
	trend := make([]int, length)  // 1 for uptrend, -1 for downtrend

	// Initialize
	h0, _ := starlark.AsFloat(high.Index(0))
	l0, _ := starlark.AsFloat(low.Index(0))
	h1, _ := starlark.AsFloat(high.Index(1))
	l1, _ := starlark.AsFloat(low.Index(1))

	result[0] = math.NaN()
	if h1 > h0 {
		trend[1] = 1
		result[1] = l0
		ep[1] = h1
	} else {
		trend[1] = -1
		result[1] = h0
		ep[1] = l1
	}
	af[1] = step

	for i := 2; i < length; i++ {
		hi, _ := starlark.AsFloat(high.Index(i))
		li, _ := starlark.AsFloat(low.Index(i))

		// Calculate SAR
		result[i] = result[i-1] + af[i-1]*(ep[i-1]-result[i-1])

		if trend[i-1] == 1 { // Uptrend
			if li <= result[i] {
				// Trend reversal to downtrend
				trend[i] = -1
				result[i] = ep[i-1]
				ep[i] = li
				af[i] = step
			} else {
				trend[i] = 1
				if hi > ep[i-1] {
					ep[i] = hi
					af[i] = math.Min(af[i-1]+step, maxStep)
				} else {
					ep[i] = ep[i-1]
					af[i] = af[i-1]
				}
				// Ensure SAR doesn't exceed previous two lows
				l1, _ := starlark.AsFloat(low.Index(i - 1))
				l2, _ := starlark.AsFloat(low.Index(i - 2))
				result[i] = math.Min(result[i], math.Min(l1, l2))
			}
		} else { // Downtrend
			if hi >= result[i] {
				// Trend reversal to uptrend
				trend[i] = 1
				result[i] = ep[i-1]
				ep[i] = hi
				af[i] = step
			} else {
				trend[i] = -1
				if li < ep[i-1] {
					ep[i] = li
					af[i] = math.Min(af[i-1]+step, maxStep)
				} else {
					ep[i] = ep[i-1]
					af[i] = af[i-1]
				}
				// Ensure SAR doesn't exceed previous two highs
				h1, _ := starlark.AsFloat(high.Index(i - 1))
				h2, _ := starlark.AsFloat(high.Index(i - 2))
				result[i] = math.Max(result[i], math.Max(h1, h2))
			}
		}
	}

	return result
}

// calculateKeltnerChannels calculates Keltner Channels
func (ti *TechnicalIndicators) calculateKeltnerChannels(high, low, close *starlark.List, period int, multiplier float64) ([]float64, []float64, []float64) {
	length := close.Len()
	middle := ti.calculateEMA(close, period)
	atr := ti.calculateATR(high, low, close, period)

	upper := make([]float64, length)
	lower := make([]float64, length)

	for i := 0; i < length; i++ {
		if math.IsNaN(middle[i]) || math.IsNaN(atr[i]) {
			upper[i] = math.NaN()
			lower[i] = math.NaN()
		} else {
			upper[i] = middle[i] + (multiplier * atr[i])
			lower[i] = middle[i] - (multiplier * atr[i])
		}
	}

	return upper, middle, lower
}

// calculateIchimoku calculates Ichimoku Cloud components
func (ti *TechnicalIndicators) calculateIchimoku(high, low, close *starlark.List, conversionPeriod, basePeriod, spanBPeriod, displacement int) ([]float64, []float64, []float64, []float64, []float64) {
	length := close.Len()

	// Tenkan-sen (Conversion Line)
	tenkanSen := make([]float64, length)
	for i := 0; i < length; i++ {
		if i < conversionPeriod-1 {
			tenkanSen[i] = math.NaN()
			continue
		}

		highest := math.Inf(-1)
		lowest := math.Inf(1)
		for j := i - conversionPeriod + 1; j <= i; j++ {
			h, _ := starlark.AsFloat(high.Index(j))
			l, _ := starlark.AsFloat(low.Index(j))
			if h > highest {
				highest = h
			}
			if l < lowest {
				lowest = l
			}
		}
		tenkanSen[i] = (highest + lowest) / 2
	}

	// Kijun-sen (Base Line)
	kijunSen := make([]float64, length)
	for i := 0; i < length; i++ {
		if i < basePeriod-1 {
			kijunSen[i] = math.NaN()
			continue
		}

		highest := math.Inf(-1)
		lowest := math.Inf(1)
		for j := i - basePeriod + 1; j <= i; j++ {
			h, _ := starlark.AsFloat(high.Index(j))
			l, _ := starlark.AsFloat(low.Index(j))
			if h > highest {
				highest = h
			}
			if l < lowest {
				lowest = l
			}
		}
		kijunSen[i] = (highest + lowest) / 2
	}

	// Senkou Span A (Leading Span A)
	senkouSpanA := make([]float64, length)
	for i := 0; i < length; i++ {
		if math.IsNaN(tenkanSen[i]) || math.IsNaN(kijunSen[i]) {
			senkouSpanA[i] = math.NaN()
		} else {
			senkouSpanA[i] = (tenkanSen[i] + kijunSen[i]) / 2
		}
	}

	// Senkou Span B (Leading Span B)
	senkouSpanB := make([]float64, length)
	for i := 0; i < length; i++ {
		if i < spanBPeriod-1 {
			senkouSpanB[i] = math.NaN()
			continue
		}

		highest := math.Inf(-1)
		lowest := math.Inf(1)
		for j := i - spanBPeriod + 1; j <= i; j++ {
			h, _ := starlark.AsFloat(high.Index(j))
			l, _ := starlark.AsFloat(low.Index(j))
			if h > highest {
				highest = h
			}
			if l < lowest {
				lowest = l
			}
		}
		senkouSpanB[i] = (highest + lowest) / 2
	}

	// Chikou Span (Lagging Span)
	chikouSpan := make([]float64, length)
	for i := 0; i < length; i++ {
		if i < displacement {
			chikouSpan[i] = math.NaN()
		} else {
			c, _ := starlark.AsFloat(close.Index(i - displacement))
			chikouSpan[i] = c
		}
	}

	return tenkanSen, kijunSen, senkouSpanA, senkouSpanB, chikouSpan
}

// calculatePivotPoints calculates Pivot Points and support/resistance levels
func (ti *TechnicalIndicators) calculatePivotPoints(high, low, close *starlark.List) ([]float64, []float64, []float64, []float64, []float64, []float64, []float64) {
	length := close.Len()
	if length == 0 {
		return nil, nil, nil, nil, nil, nil, nil
	}

	pivot := make([]float64, length)
	r1 := make([]float64, length)
	r2 := make([]float64, length)
	r3 := make([]float64, length)
	s1 := make([]float64, length)
	s2 := make([]float64, length)
	s3 := make([]float64, length)

	for i := 0; i < length; i++ {
		if i == 0 {
			// Use current values for first calculation
			h, _ := starlark.AsFloat(high.Index(i))
			l, _ := starlark.AsFloat(low.Index(i))
			c, _ := starlark.AsFloat(close.Index(i))

			pivot[i] = (h + l + c) / 3
			r1[i] = (2 * pivot[i]) - l
			r2[i] = pivot[i] + (h - l)
			r3[i] = h + 2*(pivot[i]-l)
			s1[i] = (2 * pivot[i]) - h
			s2[i] = pivot[i] - (h - l)
			s3[i] = l - 2*(h-pivot[i])
		} else {
			// Use previous day's values
			h, _ := starlark.AsFloat(high.Index(i - 1))
			l, _ := starlark.AsFloat(low.Index(i - 1))
			c, _ := starlark.AsFloat(close.Index(i - 1))

			pivot[i] = (h + l + c) / 3
			r1[i] = (2 * pivot[i]) - l
			r2[i] = pivot[i] + (h - l)
			r3[i] = h + 2*(pivot[i]-l)
			s1[i] = (2 * pivot[i]) - h
			s2[i] = pivot[i] - (h - l)
			s3[i] = l - 2*(h-pivot[i])
		}
	}

	return pivot, r1, r2, r3, s1, s2, s3
}

// calculateFibonacciRetracement calculates Fibonacci retracement levels
func (ti *TechnicalIndicators) calculateFibonacciRetracement(high, low float64) map[string]float64 {
	diff := high - low
	levels := map[string]float64{
		"0.0":   high,
		"23.6":  high - 0.236*diff,
		"38.2":  high - 0.382*diff,
		"50.0":  high - 0.500*diff,
		"61.8":  high - 0.618*diff,
		"78.6":  high - 0.786*diff,
		"100.0": low,
	}
	return levels
}

// calculateAroon calculates Aroon Up and Aroon Down
func (ti *TechnicalIndicators) calculateAroon(high, low *starlark.List, period int) ([]float64, []float64) {
	length := high.Len()
	aroonUp := make([]float64, length)
	aroonDown := make([]float64, length)

	for i := 0; i < length; i++ {
		if i < period-1 {
			aroonUp[i] = math.NaN()
			aroonDown[i] = math.NaN()
			continue
		}

		// Find periods since highest high and lowest low
		highestIdx := i
		lowestIdx := i
		highest := math.Inf(-1)
		lowest := math.Inf(1)

		for j := i - period + 1; j <= i; j++ {
			h, _ := starlark.AsFloat(high.Index(j))
			l, _ := starlark.AsFloat(low.Index(j))

			if h > highest {
				highest = h
				highestIdx = j
			}
			if l < lowest {
				lowest = l
				lowestIdx = j
			}
		}

		periodsSinceHigh := i - highestIdx
		periodsSinceLow := i - lowestIdx

		aroonUp[i] = ((float64(period) - float64(periodsSinceHigh)) / float64(period)) * 100
		aroonDown[i] = ((float64(period) - float64(periodsSinceLow)) / float64(period)) * 100
	}

	return aroonUp, aroonDown
}
