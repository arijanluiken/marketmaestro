package strategy

import (
	"math"

	"go.starlark.net/starlark"
)

// calculateTSI calculates True Strength Index
func (ti *TechnicalIndicators) calculateTSI(prices *starlark.List, longPeriod, shortPeriod int) []float64 {
	length := prices.Len()
	if length < longPeriod {
		return nil
	}

	// Calculate price changes
	changes := make([]float64, length)
	absChanges := make([]float64, length)

	changes[0] = 0
	absChanges[0] = 0

	for i := 1; i < length; i++ {
		current, _ := starlark.AsFloat(prices.Index(i))
		previous, _ := starlark.AsFloat(prices.Index(i - 1))
		change := current - previous
		changes[i] = change
		absChanges[i] = math.Abs(change)
	}

	// First smoothing
	firstSmoothedChanges := ti.smoothArray(changes, longPeriod)
	firstSmoothedAbsChanges := ti.smoothArray(absChanges, longPeriod)

	// Second smoothing
	secondSmoothedChanges := ti.smoothArray(firstSmoothedChanges, shortPeriod)
	secondSmoothedAbsChanges := ti.smoothArray(firstSmoothedAbsChanges, shortPeriod)

	// Calculate TSI
	result := make([]float64, length)
	for i := 0; i < length; i++ {
		if secondSmoothedAbsChanges[i] == 0 {
			result[i] = 0
		} else {
			result[i] = 100 * (secondSmoothedChanges[i] / secondSmoothedAbsChanges[i])
		}
	}

	return result
}

// calculateDonchianChannels calculates Donchian Channels
func (ti *TechnicalIndicators) calculateDonchianChannels(high, low *starlark.List, period int) ([]float64, []float64, []float64) {
	length := high.Len()
	if length < period {
		return nil, nil, nil
	}

	upper := make([]float64, length)
	lower := make([]float64, length)
	middle := make([]float64, length)

	for i := 0; i < length; i++ {
		if i < period-1 {
			upper[i] = math.NaN()
			lower[i] = math.NaN()
			middle[i] = math.NaN()
			continue
		}

		maxHigh := -math.MaxFloat64
		minLow := math.MaxFloat64

		for j := i - period + 1; j <= i; j++ {
			h, _ := starlark.AsFloat(high.Index(j))
			l, _ := starlark.AsFloat(low.Index(j))

			if h > maxHigh {
				maxHigh = h
			}
			if l < minLow {
				minLow = l
			}
		}

		upper[i] = maxHigh
		lower[i] = minLow
		middle[i] = (maxHigh + minLow) / 2
	}

	return upper, middle, lower
}

// calculateCommodityChannelIndex calculates advanced CCI with smoothing
func (ti *TechnicalIndicators) calculateAdvancedCCI(high, low, close *starlark.List, period int, smoothPeriod int) []float64 {
	basicCCI := ti.calculateCCI(high, low, close, period)
	if basicCCI == nil {
		return nil
	}

	// Convert to starlark list for smoothing
	cciList := starlark.NewList(nil)
	for _, val := range basicCCI {
		if !math.IsNaN(val) {
			cciList.Append(starlark.Float(val))
		}
	}

	// Apply smoothing
	return ti.calculateSMA(cciList, smoothPeriod)
}

// calculateElderRay calculates Elder Ray Index (Bull Power and Bear Power)
func (ti *TechnicalIndicators) calculateElderRay(high, low, close *starlark.List, period int) ([]float64, []float64) {
	length := close.Len()
	if length < period {
		return nil, nil
	}

	// Calculate EMA
	ema := ti.calculateEMA(close, period)

	bullPower := make([]float64, length)
	bearPower := make([]float64, length)

	for i := 0; i < length; i++ {
		if i < period-1 {
			bullPower[i] = math.NaN()
			bearPower[i] = math.NaN()
			continue
		}

		h, _ := starlark.AsFloat(high.Index(i))
		l, _ := starlark.AsFloat(low.Index(i))

		bullPower[i] = h - ema[i]
		bearPower[i] = l - ema[i]
	}

	return bullPower, bearPower
}

// calculateDetrended calculates Detrended Price Oscillator
func (ti *TechnicalIndicators) calculateDetrended(prices *starlark.List, period int) []float64 {
	length := prices.Len()
	if length < period {
		return nil
	}

	sma := ti.calculateSMA(prices, period)
	result := make([]float64, length)
	offset := (period / 2) + 1

	for i := 0; i < length; i++ {
		if i < period-1 || i-offset < 0 {
			result[i] = math.NaN()
			continue
		}

		price, _ := starlark.AsFloat(prices.Index(i))
		result[i] = price - sma[i-offset]
	}

	return result
}

// calculateKaufmanAMA calculates Kaufman Adaptive Moving Average
func (ti *TechnicalIndicators) calculateKaufmanAMA(prices *starlark.List, period, fastSC, slowSC int) []float64 {
	length := prices.Len()
	if length < period {
		return nil
	}

	result := make([]float64, length)
	fastConstant := 2.0 / (float64(fastSC) + 1.0)
	slowConstant := 2.0 / (float64(slowSC) + 1.0)

	// Initialize first value
	firstPrice, _ := starlark.AsFloat(prices.Index(period - 1))
	result[period-1] = firstPrice

	for i := period; i < length; i++ {
		current, _ := starlark.AsFloat(prices.Index(i))
		oldest, _ := starlark.AsFloat(prices.Index(i - period))

		// Calculate change and volatility
		change := math.Abs(current - oldest)
		volatility := 0.0

		for j := i - period + 1; j <= i; j++ {
			p1, _ := starlark.AsFloat(prices.Index(j))
			p2, _ := starlark.AsFloat(prices.Index(j - 1))
			volatility += math.Abs(p1 - p2)
		}

		// Calculate efficiency ratio
		var efficiencyRatio float64
		if volatility != 0 {
			efficiencyRatio = change / volatility
		} else {
			efficiencyRatio = 0
		}

		// Calculate smoothing constant
		smoothingConstant := math.Pow(efficiencyRatio*(fastConstant-slowConstant)+slowConstant, 2)

		// Calculate AMA
		result[i] = result[i-1] + smoothingConstant*(current-result[i-1])
	}

	// Fill initial values with NaN
	for i := 0; i < period-1; i++ {
		result[i] = math.NaN()
	}

	return result
}

// calculateChaikinOscillator calculates Chaikin Oscillator
func (ti *TechnicalIndicators) calculateChaikinOscillator(high, low, close, volume *starlark.List, fastPeriod, slowPeriod int) []float64 {
	length := close.Len()
	if length == 0 {
		return nil
	}

	// Calculate Accumulation/Distribution Line
	adLine := make([]float64, length)
	adLine[0] = 0

	for i := 1; i < length; i++ {
		h, _ := starlark.AsFloat(high.Index(i))
		l, _ := starlark.AsFloat(low.Index(i))
		c, _ := starlark.AsFloat(close.Index(i))
		v, _ := starlark.AsFloat(volume.Index(i))

		var moneyFlowMultiplier float64
		if h != l {
			moneyFlowMultiplier = ((c - l) - (h - c)) / (h - l)
		} else {
			moneyFlowMultiplier = 0
		}

		moneyFlowVolume := moneyFlowMultiplier * v
		adLine[i] = adLine[i-1] + moneyFlowVolume
	}

	// Convert to starlark list for EMA calculation
	adList := starlark.NewList(nil)
	for _, val := range adLine {
		adList.Append(starlark.Float(val))
	}

	// Calculate fast and slow EMAs
	fastEMA := ti.calculateEMA(adList, fastPeriod)
	slowEMA := ti.calculateEMA(adList, slowPeriod)

	// Calculate oscillator
	result := make([]float64, length)
	for i := 0; i < length; i++ {
		if math.IsNaN(fastEMA[i]) || math.IsNaN(slowEMA[i]) {
			result[i] = math.NaN()
		} else {
			result[i] = fastEMA[i] - slowEMA[i]
		}
	}

	return result
}

// calculateUltimateOscillator calculates Ultimate Oscillator
func (ti *TechnicalIndicators) calculateUltimateOscillator(high, low, close *starlark.List, period1, period2, period3 int) []float64 {
	length := close.Len()
	if length < period3 {
		return nil
	}

	// Calculate True Range and Buying Pressure
	tr := make([]float64, length)
	bp := make([]float64, length)

	tr[0] = 0
	bp[0] = 0

	for i := 1; i < length; i++ {
		h, _ := starlark.AsFloat(high.Index(i))
		l, _ := starlark.AsFloat(low.Index(i))
		c, _ := starlark.AsFloat(close.Index(i))
		prevC, _ := starlark.AsFloat(close.Index(i - 1))

		// True Range
		tr1 := h - l
		tr2 := math.Abs(h - prevC)
		tr3 := math.Abs(l - prevC)
		tr[i] = math.Max(tr1, math.Max(tr2, tr3))

		// Buying Pressure
		bp[i] = c - math.Min(l, prevC)
	}

	result := make([]float64, length)

	for i := period3 - 1; i < length; i++ {
		// Sum for each period
		var sum1BP, sum1TR, sum2BP, sum2TR, sum3BP, sum3TR float64

		// Period 1
		for j := i - period1 + 1; j <= i; j++ {
			sum1BP += bp[j]
			sum1TR += tr[j]
		}

		// Period 2
		for j := i - period2 + 1; j <= i; j++ {
			sum2BP += bp[j]
			sum2TR += tr[j]
		}

		// Period 3
		for j := i - period3 + 1; j <= i; j++ {
			sum3BP += bp[j]
			sum3TR += tr[j]
		}

		// Calculate UO
		var avg1, avg2, avg3 float64
		if sum1TR != 0 {
			avg1 = sum1BP / sum1TR
		}
		if sum2TR != 0 {
			avg2 = sum2BP / sum2TR
		}
		if sum3TR != 0 {
			avg3 = sum3BP / sum3TR
		}

		result[i] = 100 * ((4*avg1 + 2*avg2 + avg3) / 7)
	}

	// Fill initial values with NaN
	for i := 0; i < period3-1; i++ {
		result[i] = math.NaN()
	}

	return result
}

// smoothArray helper function for TSI calculation
func (ti *TechnicalIndicators) smoothArray(data []float64, period int) []float64 {
	length := len(data)
	result := make([]float64, length)

	if length == 0 || period <= 0 {
		return result
	}

	multiplier := 2.0 / (float64(period) + 1.0)
	result[0] = data[0]

	for i := 1; i < length; i++ {
		result[i] = result[i-1] + multiplier*(data[i]-result[i-1])
	}

	return result
}

// calculateHeikinAshi calculates Heikin Ashi candlesticks
func (ti *TechnicalIndicators) calculateHeikinAshi(open, high, low, close *starlark.List) ([]float64, []float64, []float64, []float64) {
	length := close.Len()
	if length == 0 {
		return nil, nil, nil, nil
	}

	haOpen := make([]float64, length)
	haHigh := make([]float64, length)
	haLow := make([]float64, length)
	haClose := make([]float64, length)

	// First candle
	o, _ := starlark.AsFloat(open.Index(0))
	h, _ := starlark.AsFloat(high.Index(0))
	l, _ := starlark.AsFloat(low.Index(0))
	c, _ := starlark.AsFloat(close.Index(0))

	haOpen[0] = (o + c) / 2
	haClose[0] = (o + h + l + c) / 4
	haHigh[0] = math.Max(h, math.Max(haOpen[0], haClose[0]))
	haLow[0] = math.Min(l, math.Min(haOpen[0], haClose[0]))

	for i := 1; i < length; i++ {
		o, _ := starlark.AsFloat(open.Index(i))
		h, _ := starlark.AsFloat(high.Index(i))
		l, _ := starlark.AsFloat(low.Index(i))
		c, _ := starlark.AsFloat(close.Index(i))

		haOpen[i] = (haOpen[i-1] + haClose[i-1]) / 2
		haClose[i] = (o + h + l + c) / 4
		haHigh[i] = math.Max(h, math.Max(haOpen[i], haClose[i]))
		haLow[i] = math.Min(l, math.Min(haOpen[i], haClose[i]))
	}

	return haOpen, haHigh, haLow, haClose
}

// calculateVortex calculates Vortex Indicator
func (ti *TechnicalIndicators) calculateVortex(high, low, close *starlark.List, period int) ([]float64, []float64) {
	length := close.Len()
	if length < period+1 {
		return nil, nil
	}

	viPlus := make([]float64, length)
	viMinus := make([]float64, length)

	for i := period; i < length; i++ {
		var vmPlus, vmMinus, tr float64

		for j := i - period + 1; j <= i; j++ {
			h, _ := starlark.AsFloat(high.Index(j))
			l, _ := starlark.AsFloat(low.Index(j))
			prevC, _ := starlark.AsFloat(close.Index(j - 1))

			// Vortex Movement
			if j > 0 {
				prevLAtJ, _ := starlark.AsFloat(low.Index(j - 1))
				prevHAtJ, _ := starlark.AsFloat(high.Index(j - 1))
				vmPlus += math.Abs(h - prevLAtJ)
				vmMinus += math.Abs(l - prevHAtJ)
			}

			// True Range
			tr1 := h - l
			tr2 := math.Abs(h - prevC)
			tr3 := math.Abs(l - prevC)
			tr += math.Max(tr1, math.Max(tr2, tr3))
		}

		if tr != 0 {
			viPlus[i] = vmPlus / tr
			viMinus[i] = vmMinus / tr
		}
	}

	// Fill initial values with NaN
	for i := 0; i < period; i++ {
		viPlus[i] = math.NaN()
		viMinus[i] = math.NaN()
	}

	return viPlus, viMinus
}

// calculateWilliamsAlligator calculates Williams Alligator
func (ti *TechnicalIndicators) calculateWilliamsAlligator(prices *starlark.List) ([]float64, []float64, []float64) {
	// Jaw (blue line) - 13-period smoothed moving average, shifted into the future by 8 bars
	jaw := ti.calculateSMMA(prices, 13)

	// Teeth (red line) - 8-period smoothed moving average, shifted into the future by 5 bars
	teeth := ti.calculateSMMA(prices, 8)

	// Lips (green line) - 5-period smoothed moving average, shifted into the future by 3 bars
	lips := ti.calculateSMMA(prices, 5)

	// Apply forward shifts
	length := len(jaw)
	shiftedJaw := make([]float64, length)
	shiftedTeeth := make([]float64, length)
	shiftedLips := make([]float64, length)

	// Shift jaw by 8 bars
	for i := 0; i < length; i++ {
		if i >= 8 {
			shiftedJaw[i] = jaw[i-8]
		} else {
			shiftedJaw[i] = math.NaN()
		}
	}

	// Shift teeth by 5 bars
	for i := 0; i < length; i++ {
		if i >= 5 {
			shiftedTeeth[i] = teeth[i-5]
		} else {
			shiftedTeeth[i] = math.NaN()
		}
	}

	// Shift lips by 3 bars
	for i := 0; i < length; i++ {
		if i >= 3 {
			shiftedLips[i] = lips[i-3]
		} else {
			shiftedLips[i] = math.NaN()
		}
	}

	return shiftedJaw, shiftedTeeth, shiftedLips
}

// calculateSMMA calculates Smoothed Moving Average (SMMA)
func (ti *TechnicalIndicators) calculateSMMA(prices *starlark.List, period int) []float64 {
	length := prices.Len()
	if length < period {
		return nil
	}

	result := make([]float64, length)

	// First SMMA value is SMA
	var sum float64
	for i := 0; i < period; i++ {
		price, _ := starlark.AsFloat(prices.Index(i))
		sum += price
		if i < period-1 {
			result[i] = math.NaN()
		}
	}
	result[period-1] = sum / float64(period)

	// Subsequent values use SMMA formula
	for i := period; i < length; i++ {
		price, _ := starlark.AsFloat(prices.Index(i))
		result[i] = (result[i-1]*float64(period-1) + price) / float64(period)
	}

	return result
}

// calculateSupertrend calculates Supertrend indicator
func (ti *TechnicalIndicators) calculateSupertrend(high, low, close *starlark.List, period int, multiplier float64) ([]float64, []bool) {
	length := close.Len()
	if length < period {
		return nil, nil
	}

	// Calculate ATR
	atr := ti.calculateATR(high, low, close, period)

	supertrend := make([]float64, length)
	trend := make([]bool, length) // true for uptrend, false for downtrend

	for i := 0; i < length; i++ {
		if i < period-1 {
			supertrend[i] = math.NaN()
			trend[i] = true
			continue
		}

		h, _ := starlark.AsFloat(high.Index(i))
		l, _ := starlark.AsFloat(low.Index(i))
		c, _ := starlark.AsFloat(close.Index(i))

		hl2 := (h + l) / 2
		upperBand := hl2 + (multiplier * atr[i])
		lowerBand := hl2 - (multiplier * atr[i])

		if i == period-1 {
			// Initialize
			if c <= hl2 {
				supertrend[i] = upperBand
				trend[i] = false
			} else {
				supertrend[i] = lowerBand
				trend[i] = true
			}
		} else {
			prevClose, _ := starlark.AsFloat(close.Index(i - 1))

			// Calculate basic upper and lower bands
			if upperBand < supertrend[i-1] || prevClose > supertrend[i-1] {
				upperBand = math.Min(upperBand, supertrend[i-1])
			}

			if lowerBand > supertrend[i-1] || prevClose < supertrend[i-1] {
				lowerBand = math.Max(lowerBand, supertrend[i-1])
			}

			// Determine trend and supertrend value
			if trend[i-1] && c <= lowerBand {
				supertrend[i] = upperBand
				trend[i] = false
			} else if !trend[i-1] && c >= upperBand {
				supertrend[i] = lowerBand
				trend[i] = true
			} else {
				supertrend[i] = supertrend[i-1]
				trend[i] = trend[i-1]
			}
		}
	}

	return supertrend, trend
}

// calculateStochasticRSI calculates Stochastic RSI
func (ti *TechnicalIndicators) calculateStochasticRSI(prices *starlark.List, rsiPeriod, stochPeriod, kPeriod, dPeriod int) ([]float64, []float64) {
	// First calculate RSI
	rsi := ti.calculateRSI(prices, rsiPeriod)

	// Calculate Stochastic of RSI
	length := len(rsi)
	stochRSI := make([]float64, length)

	for i := stochPeriod - 1; i < length; i++ {
		if math.IsNaN(rsi[i]) {
			stochRSI[i] = math.NaN()
			continue
		}

		var minRSI, maxRSI float64 = math.MaxFloat64, -math.MaxFloat64

		for j := i - stochPeriod + 1; j <= i; j++ {
			if !math.IsNaN(rsi[j]) {
				if rsi[j] < minRSI {
					minRSI = rsi[j]
				}
				if rsi[j] > maxRSI {
					maxRSI = rsi[j]
				}
			}
		}

		if maxRSI != minRSI {
			stochRSI[i] = 100 * (rsi[i] - minRSI) / (maxRSI - minRSI)
		} else {
			stochRSI[i] = 50
		}
	}

	// Fill early values with NaN
	for i := 0; i < stochPeriod-1; i++ {
		stochRSI[i] = math.NaN()
	}

	// Convert to starlark list for SMA calculation
	stochRSIList := starlark.NewList(nil)
	for _, val := range stochRSI {
		if !math.IsNaN(val) {
			stochRSIList.Append(starlark.Float(val))
		}
	}

	// Calculate %K and %D
	k := ti.calculateSMA(stochRSIList, kPeriod)

	kList := starlark.NewList(nil)
	for _, val := range k {
		if !math.IsNaN(val) {
			kList.Append(starlark.Float(val))
		}
	}

	d := ti.calculateSMA(kList, dPeriod)

	return k, d
}

// calculateAwesome calculates Awesome Oscillator
func (ti *TechnicalIndicators) calculateAwesome(high, low *starlark.List) []float64 {
	length := high.Len()
	if length < 34 {
		return nil
	}

	// Calculate median prices
	medianPrices := starlark.NewList(nil)
	for i := 0; i < length; i++ {
		h, _ := starlark.AsFloat(high.Index(i))
		l, _ := starlark.AsFloat(low.Index(i))
		medianPrices.Append(starlark.Float((h + l) / 2))
	}

	// Calculate 5-period and 34-period SMAs
	sma5 := ti.calculateSMA(medianPrices, 5)
	sma34 := ti.calculateSMA(medianPrices, 34)

	// Calculate Awesome Oscillator
	result := make([]float64, length)
	for i := 0; i < length; i++ {
		if math.IsNaN(sma5[i]) || math.IsNaN(sma34[i]) {
			result[i] = math.NaN()
		} else {
			result[i] = sma5[i] - sma34[i]
		}
	}

	return result
}

// calculateAcceleratorOscillator calculates Accelerator Oscillator
func (ti *TechnicalIndicators) calculateAcceleratorOscillator(high, low, close *starlark.List) []float64 {
	// First calculate Awesome Oscillator
	ao := ti.calculateAwesome(high, low)
	if len(ao) < 5 {
		return make([]float64, len(ao)) // Return NaN values for insufficient data
	}

	// Convert to starlark list
	aoList := starlark.NewList(nil)
	for _, val := range ao {
		if !math.IsNaN(val) {
			aoList.Append(starlark.Float(val))
		}
	}

	// Calculate 5-period SMA of AO
	aoSMA := ti.calculateSMA(aoList, 5)

	// Calculate AC (AO - SMA of AO)
	length := len(ao)
	result := make([]float64, length)

	// Fill with NaN values
	for i := 0; i < length; i++ {
		result[i] = math.NaN()
	}

	// Calculate AC values where we have sufficient data
	smaLength := len(aoSMA)
	startIdx := length - smaLength

	for i := 0; i < smaLength; i++ {
		aoIdx := startIdx + i
		if aoIdx >= 0 && aoIdx < length {
			if !math.IsNaN(ao[aoIdx]) && !math.IsNaN(aoSMA[i]) {
				result[aoIdx] = ao[aoIdx] - aoSMA[i]
			}
		}
	}

	return result
}

// calculateHullMA calculates Hull Moving Average
func (ti *TechnicalIndicators) calculateHullMA(prices *starlark.List, period int) []float64 {
	length := prices.Len()
	if length < period {
		return nil
	}

	halfPeriod := period / 2
	sqrtPeriod := int(math.Sqrt(float64(period)))

	// Calculate WMA for half period
	wmaHalf := ti.calculateWMA(prices, halfPeriod)
	
	// Calculate WMA for full period
	wmaFull := ti.calculateWMA(prices, period)

	// Calculate 2*WMA(n/2) - WMA(n)
	diffValues := make([]float64, length)
	for i := 0; i < length; i++ {
		if math.IsNaN(wmaHalf[i]) || math.IsNaN(wmaFull[i]) {
			diffValues[i] = math.NaN()
		} else {
			diffValues[i] = 2*wmaHalf[i] - wmaFull[i]
		}
	}

	// Convert to starlark list for final WMA calculation
	diffList := starlark.NewList(nil)
	for _, val := range diffValues {
		if !math.IsNaN(val) {
			diffList.Append(starlark.Float(val))
		} else {
			diffList.Append(starlark.None)
		}
	}

	// Calculate WMA of sqrt(period) on the difference
	return ti.calculateWMA(diffList, sqrtPeriod)
}

// calculateWMA calculates Weighted Moving Average
func (ti *TechnicalIndicators) calculateWMA(prices *starlark.List, period int) []float64 {
	length := prices.Len()
	if length < period {
		return nil
	}

	result := make([]float64, length)
	weightSum := float64(period * (period + 1) / 2)

	for i := 0; i < length; i++ {
		if i < period-1 {
			result[i] = math.NaN()
			continue
		}

		weightedSum := 0.0
		for j := 0; j < period; j++ {
			price, _ := starlark.AsFloat(prices.Index(i - period + 1 + j))
			weight := float64(j + 1)
			weightedSum += price * weight
		}
		result[i] = weightedSum / weightSum
	}

	return result
}

// calculateChandelierExit calculates Chandelier Exit indicator
func (ti *TechnicalIndicators) calculateChandelierExit(high, low, close *starlark.List, period int, multiplier float64) ([]float64, []float64) {
	length := close.Len()
	if length < period {
		return nil, nil
	}

	atr := ti.calculateATR(high, low, close, period)
	longExit := make([]float64, length)
	shortExit := make([]float64, length)

	for i := 0; i < length; i++ {
		if i < period-1 {
			longExit[i] = math.NaN()
			shortExit[i] = math.NaN()
			continue
		}

		// Find highest high and lowest low over period
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

		if !math.IsNaN(atr[i]) {
			longExit[i] = highestHigh - (multiplier * atr[i])
			shortExit[i] = lowestLow + (multiplier * atr[i])
		} else {
			longExit[i] = math.NaN()
			shortExit[i] = math.NaN()
		}
	}

	return longExit, shortExit
}

// calculateALMA calculates Arnaud Legoux Moving Average
func (ti *TechnicalIndicators) calculateALMA(prices *starlark.List, period int, offset float64, sigma float64) []float64 {
	length := prices.Len()
	if length < period {
		return nil
	}

	result := make([]float64, length)
	m := offset * float64(period-1)
	s := float64(period) / sigma

	// Pre-calculate weights
	weights := make([]float64, period)
	weightSum := 0.0
	for i := 0; i < period; i++ {
		weight := math.Exp(-math.Pow(float64(i)-m, 2)/(2*s*s))
		weights[i] = weight
		weightSum += weight
	}

	for i := 0; i < length; i++ {
		if i < period-1 {
			result[i] = math.NaN()
			continue
		}

		weightedSum := 0.0
		for j := 0; j < period; j++ {
			price, _ := starlark.AsFloat(prices.Index(i - period + 1 + j))
			weightedSum += price * weights[j]
		}
		result[i] = weightedSum / weightSum
	}

	return result
}

// calculateCMO calculates Chande Momentum Oscillator
func (ti *TechnicalIndicators) calculateCMO(prices *starlark.List, period int) []float64 {
	length := prices.Len()
	if length < period+1 {
		return nil
	}

	result := make([]float64, length)
	gains := make([]float64, length)
	losses := make([]float64, length)

	// Calculate price changes
	for i := 1; i < length; i++ {
		current, _ := starlark.AsFloat(prices.Index(i))
		previous, _ := starlark.AsFloat(prices.Index(i-1))
		change := current - previous

		if change > 0 {
			gains[i] = change
			losses[i] = 0
		} else {
			gains[i] = 0
			losses[i] = -change
		}
	}

	// Calculate CMO
	for i := period; i < length; i++ {
		sumGains := 0.0
		sumLosses := 0.0

		for j := i - period + 1; j <= i; j++ {
			sumGains += gains[j]
			sumLosses += losses[j]
		}

		if sumGains+sumLosses == 0 {
			result[i] = 0
		} else {
			result[i] = 100 * (sumGains - sumLosses) / (sumGains + sumLosses)
		}
	}

	// Fill initial values with NaN
	for i := 0; i < period; i++ {
		result[i] = math.NaN()
	}

	return result
}

// calculateTEMA calculates Triple Exponential Moving Average
func (ti *TechnicalIndicators) calculateTEMA(prices *starlark.List, period int) []float64 {
	// First EMA
	ema1 := ti.calculateEMA(prices, period)
	
	// Convert to starlark list for second EMA
	ema1List := starlark.NewList(nil)
	for _, val := range ema1 {
		if !math.IsNaN(val) {
			ema1List.Append(starlark.Float(val))
		} else {
			ema1List.Append(starlark.None)
		}
	}

	// Second EMA
	ema2 := ti.calculateEMA(ema1List, period)

	// Convert to starlark list for third EMA
	ema2List := starlark.NewList(nil)
	for _, val := range ema2 {
		if !math.IsNaN(val) {
			ema2List.Append(starlark.Float(val))
		} else {
			ema2List.Append(starlark.None)
		}
	}

	// Third EMA
	ema3 := ti.calculateEMA(ema2List, period)

	// Calculate TEMA: 3*EMA1 - 3*EMA2 + EMA3
	length := len(ema1)
	result := make([]float64, length)

	for i := 0; i < length; i++ {
		if math.IsNaN(ema1[i]) || math.IsNaN(ema2[i]) || math.IsNaN(ema3[i]) {
			result[i] = math.NaN()
		} else {
			result[i] = 3*ema1[i] - 3*ema2[i] + ema3[i]
		}
	}

	return result
}

// calculateEMV calculates Ease of Movement
func (ti *TechnicalIndicators) calculateEMV(high, low, close, volume *starlark.List, period int) []float64 {
	length := close.Len()
	if length < 2 {
		return nil
	}

	emvRaw := make([]float64, length)
	emvRaw[0] = 0

	for i := 1; i < length; i++ {
		h, _ := starlark.AsFloat(high.Index(i))
		l, _ := starlark.AsFloat(low.Index(i))
		v, _ := starlark.AsFloat(volume.Index(i))
		
		prevH, _ := starlark.AsFloat(high.Index(i-1))
		prevL, _ := starlark.AsFloat(low.Index(i-1))

		// Distance moved
		dm := ((h + l) / 2) - ((prevH + prevL) / 2)
		
		// Box height (high - low)
		boxHeight := h - l
		
		// Scale factor
		if boxHeight != 0 && v != 0 {
			scaleFactor := v / (100000000 * boxHeight) // Scale down volume
			emvRaw[i] = dm / scaleFactor
		} else {
			emvRaw[i] = 0
		}
	}

	// Apply moving average smoothing
	emvList := starlark.NewList(nil)
	for _, val := range emvRaw {
		emvList.Append(starlark.Float(val))
	}

	return ti.calculateSMA(emvList, period)
}

// calculateForceIndex calculates Force Index
func (ti *TechnicalIndicators) calculateForceIndex(close, volume *starlark.List, period int) []float64 {
	length := close.Len()
	if length < 2 {
		return nil
	}

	fi := make([]float64, length)
	fi[0] = 0

	for i := 1; i < length; i++ {
		c, _ := starlark.AsFloat(close.Index(i))
		v, _ := starlark.AsFloat(volume.Index(i))
		prevC, _ := starlark.AsFloat(close.Index(i-1))

		fi[i] = (c - prevC) * v
	}

	// Apply EMA smoothing
	fiList := starlark.NewList(nil)
	for _, val := range fi {
		fiList.Append(starlark.Float(val))
	}

	return ti.calculateEMA(fiList, period)
}

// calculateBOP calculates Balance of Power
func (ti *TechnicalIndicators) calculateBOP(open, high, low, close *starlark.List) []float64 {
	length := close.Len()
	result := make([]float64, length)

	for i := 0; i < length; i++ {
		o, _ := starlark.AsFloat(open.Index(i))
		h, _ := starlark.AsFloat(high.Index(i))
		l, _ := starlark.AsFloat(low.Index(i))
		c, _ := starlark.AsFloat(close.Index(i))

		if h != l {
			result[i] = (c - o) / (h - l)
		} else {
			result[i] = 0
		}
	}

	return result
}

// calculatePriceChannel calculates Price Channel
func (ti *TechnicalIndicators) calculatePriceChannel(high, low *starlark.List, period int) ([]float64, []float64, []float64) {
	length := high.Len()
	if length < period {
		return nil, nil, nil
	}

	upper := make([]float64, length)
	lower := make([]float64, length)
	middle := make([]float64, length)

	for i := 0; i < length; i++ {
		if i < period-1 {
			upper[i] = math.NaN()
			lower[i] = math.NaN()
			middle[i] = math.NaN()
			continue
		}

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

		upper[i] = highestHigh
		lower[i] = lowestLow
		middle[i] = (highestHigh + lowestLow) / 2
	}

	return upper, middle, lower
}

// calculateMassIndex calculates Mass Index
func (ti *TechnicalIndicators) calculateMassIndex(high, low *starlark.List, period int, sumPeriod int) []float64 {
	length := high.Len()
	if length < period {
		return nil
	}

	// Calculate high-low range
	ranges := make([]float64, length)
	for i := 0; i < length; i++ {
		h, _ := starlark.AsFloat(high.Index(i))
		l, _ := starlark.AsFloat(low.Index(i))
		ranges[i] = h - l
	}

	// Convert to starlark list for EMA calculation
	rangeList := starlark.NewList(nil)
	for _, val := range ranges {
		rangeList.Append(starlark.Float(val))
	}

	// Calculate 9-period EMA of range
	ema1 := ti.calculateEMA(rangeList, period)

	// Convert first EMA to starlark list
	ema1List := starlark.NewList(nil)
	for _, val := range ema1 {
		if !math.IsNaN(val) {
			ema1List.Append(starlark.Float(val))
		} else {
			ema1List.Append(starlark.None)
		}
	}

	// Calculate 9-period EMA of the first EMA
	ema2 := ti.calculateEMA(ema1List, period)

	// Calculate ratio and sum over sumPeriod
	result := make([]float64, length)

	for i := 0; i < length; i++ {
		if i < sumPeriod-1 {
			result[i] = math.NaN()
			continue
		}

		sum := 0.0
		validCount := 0

		for j := i - sumPeriod + 1; j <= i; j++ {
			if !math.IsNaN(ema1[j]) && !math.IsNaN(ema2[j]) && ema2[j] != 0 {
				sum += ema1[j] / ema2[j]
				validCount++
			}
		}

		if validCount > 0 {
			result[i] = sum
		} else {
			result[i] = math.NaN()
		}
	}

	return result
}

// calculateVolumeOscillator calculates Volume Oscillator
func (ti *TechnicalIndicators) calculateVolumeOscillator(volume *starlark.List, fastPeriod, slowPeriod int) []float64 {
	length := volume.Len()
	if length < slowPeriod {
		return nil
	}

	fastMA := ti.calculateSMA(volume, fastPeriod)
	slowMA := ti.calculateSMA(volume, slowPeriod)

	result := make([]float64, length)
	for i := 0; i < length; i++ {
		if math.IsNaN(fastMA[i]) || math.IsNaN(slowMA[i]) || slowMA[i] == 0 {
			result[i] = math.NaN()
		} else {
			result[i] = ((fastMA[i] - slowMA[i]) / slowMA[i]) * 100
		}
	}

	return result
}

// calculateKST calculates Know Sure Thing oscillator
func (ti *TechnicalIndicators) calculateKST(prices *starlark.List, rocPeriod1, rocPeriod2, rocPeriod3, rocPeriod4, smaPeriod1, smaPeriod2, smaPeriod3, smaPeriod4 int) ([]float64, []float64) {
	length := prices.Len()
	
	// Calculate Rate of Change for different periods
	roc1 := ti.calculateROC(prices, rocPeriod1)
	roc2 := ti.calculateROC(prices, rocPeriod2)
	roc3 := ti.calculateROC(prices, rocPeriod3)
	roc4 := ti.calculateROC(prices, rocPeriod4)

	// Convert ROCs to starlark lists for SMA calculation
	roc1List := starlark.NewList(nil)
	for _, val := range roc1 {
		if !math.IsNaN(val) {
			roc1List.Append(starlark.Float(val))
		} else {
			roc1List.Append(starlark.None)
		}
	}

	roc2List := starlark.NewList(nil)
	for _, val := range roc2 {
		if !math.IsNaN(val) {
			roc2List.Append(starlark.Float(val))
		} else {
			roc2List.Append(starlark.None)
		}
	}

	roc3List := starlark.NewList(nil)
	for _, val := range roc3 {
		if !math.IsNaN(val) {
			roc3List.Append(starlark.Float(val))
		} else {
			roc3List.Append(starlark.None)
		}
	}

	roc4List := starlark.NewList(nil)
	for _, val := range roc4 {
		if !math.IsNaN(val) {
			roc4List.Append(starlark.Float(val))
		} else {
			roc4List.Append(starlark.None)
		}
	}

	// Calculate SMAs of ROCs
	smaRoc1 := ti.calculateSMA(roc1List, smaPeriod1)
	smaRoc2 := ti.calculateSMA(roc2List, smaPeriod2)
	smaRoc3 := ti.calculateSMA(roc3List, smaPeriod3)
	smaRoc4 := ti.calculateSMA(roc4List, smaPeriod4)

	// Calculate KST line
	kst := make([]float64, length)
	for i := 0; i < length; i++ {
		if math.IsNaN(smaRoc1[i]) || math.IsNaN(smaRoc2[i]) || math.IsNaN(smaRoc3[i]) || math.IsNaN(smaRoc4[i]) {
			kst[i] = math.NaN()
		} else {
			kst[i] = 1*smaRoc1[i] + 2*smaRoc2[i] + 3*smaRoc3[i] + 4*smaRoc4[i]
		}
	}

	// Calculate signal line (SMA of KST)
	kstList := starlark.NewList(nil)
	for _, val := range kst {
		if !math.IsNaN(val) {
			kstList.Append(starlark.Float(val))
		} else {
			kstList.Append(starlark.None)
		}
	}

	signal := ti.calculateSMA(kstList, 9) // Standard 9-period signal line

	return kst, signal
}

// calculateSTC calculates Schaff Trend Cycle
func (ti *TechnicalIndicators) calculateSTC(prices *starlark.List, fastPeriod, slowPeriod, cyclePeriod int, factor float64) []float64 {
	length := prices.Len()
	if length < slowPeriod {
		// Return array of NaN values instead of nil
		result := make([]float64, length)
		for i := range result {
			result[i] = math.NaN()
		}
		return result
	}

	// Calculate MACD
	macdLine, _, _ := ti.calculateMACD(prices, fastPeriod, slowPeriod, 9)

	// Calculate Stochastic of MACD
	stoch1 := make([]float64, length)
	for i := cyclePeriod - 1; i < length; i++ {
		if math.IsNaN(macdLine[i]) {
			stoch1[i] = math.NaN()
			continue
		}

		var minMACD, maxMACD float64 = math.MaxFloat64, -math.MaxFloat64
		for j := i - cyclePeriod + 1; j <= i; j++ {
			if !math.IsNaN(macdLine[j]) {
				if macdLine[j] < minMACD {
					minMACD = macdLine[j]
				}
				if macdLine[j] > maxMACD {
					maxMACD = macdLine[j]
				}
			}
		}

		if maxMACD != minMACD {
			stoch1[i] = 100 * (macdLine[i] - minMACD) / (maxMACD - minMACD)
		} else {
			stoch1[i] = 50
		}
	}

	// Smooth first stochastic
	smoothed1 := make([]float64, length)
	for i := 0; i < length; i++ {
		if i == 0 || math.IsNaN(stoch1[i]) {
			smoothed1[i] = stoch1[i]
		} else {
			if math.IsNaN(smoothed1[i-1]) {
				smoothed1[i] = stoch1[i]
			} else {
				smoothed1[i] = smoothed1[i-1] + factor*(stoch1[i]-smoothed1[i-1])
			}
		}
	}

	// Calculate second stochastic
	stoch2 := make([]float64, length)
	for i := cyclePeriod - 1; i < length; i++ {
		if math.IsNaN(smoothed1[i]) {
			stoch2[i] = math.NaN()
			continue
		}

		var minSmooth, maxSmooth float64 = math.MaxFloat64, -math.MaxFloat64
		for j := i - cyclePeriod + 1; j <= i; j++ {
			if !math.IsNaN(smoothed1[j]) {
				if smoothed1[j] < minSmooth {
					minSmooth = smoothed1[j]
				}
				if smoothed1[j] > maxSmooth {
					maxSmooth = smoothed1[j]
				}
			}
		}

		if maxSmooth != minSmooth {
			stoch2[i] = 100 * (smoothed1[i] - minSmooth) / (maxSmooth - minSmooth)
		} else {
			stoch2[i] = 50
		}
	}

	// Final smoothing
	result := make([]float64, length)
	for i := 0; i < length; i++ {
		if i == 0 || math.IsNaN(stoch2[i]) {
			result[i] = stoch2[i]
		} else {
			if math.IsNaN(result[i-1]) {
				result[i] = stoch2[i]
			} else {
				result[i] = result[i-1] + factor*(stoch2[i]-result[i-1])
			}
		}
	}

	// Fill early values with NaN
	for i := 0; i < cyclePeriod-1; i++ {
		result[i] = math.NaN()
	}

	return result
}

// calculateCoppockCurve calculates Coppock Curve
func (ti *TechnicalIndicators) calculateCoppockCurve(prices *starlark.List, roc1Period, roc2Period, wmaPeriod int) []float64 {
	length := prices.Len()
	
	// Calculate Rate of Change for both periods
	roc1 := ti.calculateROC(prices, roc1Period)
	roc2 := ti.calculateROC(prices, roc2Period)

	// Sum the ROCs
	rocSum := make([]float64, length)
	for i := 0; i < length; i++ {
		if math.IsNaN(roc1[i]) || math.IsNaN(roc2[i]) {
			rocSum[i] = math.NaN()
		} else {
			rocSum[i] = roc1[i] + roc2[i]
		}
	}

	// Convert to starlark list for WMA calculation
	rocSumList := starlark.NewList(nil)
	for _, val := range rocSum {
		if !math.IsNaN(val) {
			rocSumList.Append(starlark.Float(val))
		} else {
			rocSumList.Append(starlark.None)
		}
	}

	// Apply Weighted Moving Average
	return ti.calculateWMA(rocSumList, wmaPeriod)
}

// calculateChandeKrollStop calculates Chande Kroll Stop
func (ti *TechnicalIndicators) calculateChandeKrollStop(high, low, close *starlark.List, period int, multiplier float64) ([]float64, []float64) {
	length := close.Len()
	if length < period {
		return nil, nil
	}

	atr := ti.calculateATR(high, low, close, period)
	longStop := make([]float64, length)
	shortStop := make([]float64, length)

	for i := 0; i < length; i++ {
		if i < period-1 {
			longStop[i] = math.NaN()
			shortStop[i] = math.NaN()
			continue
		}

		// Find highest high and lowest low over period
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

		if !math.IsNaN(atr[i]) {
			longStop[i] = highestHigh - (multiplier * atr[i])
			shortStop[i] = lowestLow + (multiplier * atr[i])
		} else {
			longStop[i] = math.NaN()
			shortStop[i] = math.NaN()
		}
	}

	return longStop, shortStop
}

// calculateElderForceIndex calculates Elder's Force Index (enhanced version)
func (ti *TechnicalIndicators) calculateElderForceIndex(close, volume *starlark.List, shortPeriod, longPeriod int) ([]float64, []float64) {
	// Calculate raw Force Index
	length := close.Len()
	if length < 2 {
		return nil, nil
	}

	rawFI := make([]float64, length)
	rawFI[0] = 0

	for i := 1; i < length; i++ {
		c, _ := starlark.AsFloat(close.Index(i))
		v, _ := starlark.AsFloat(volume.Index(i))
		prevC, _ := starlark.AsFloat(close.Index(i-1))

		rawFI[i] = (c - prevC) * v
	}

	// Convert to starlark list
	rawFIList := starlark.NewList(nil)
	for _, val := range rawFI {
		rawFIList.Append(starlark.Float(val))
	}

	// Calculate short and long period EMAs
	shortFI := ti.calculateEMA(rawFIList, shortPeriod)
	longFI := ti.calculateEMA(rawFIList, longPeriod)

	return shortFI, longFI
}

// calculateKlingerOscillator calculates Klinger Volume Oscillator
func (ti *TechnicalIndicators) calculateKlingerOscillator(high, low, close, volume *starlark.List, fastPeriod, slowPeriod, signalPeriod int) ([]float64, []float64) {
	length := close.Len()
	if length < 2 {
		return nil, nil
	}

	// Calculate trend and volume force
	vf := make([]float64, length)
	vf[0] = 0

	for i := 1; i < length; i++ {
		h, _ := starlark.AsFloat(high.Index(i))
		l, _ := starlark.AsFloat(low.Index(i))
		c, _ := starlark.AsFloat(close.Index(i))
		v, _ := starlark.AsFloat(volume.Index(i))
		
		prevH, _ := starlark.AsFloat(high.Index(i-1))
		prevL, _ := starlark.AsFloat(low.Index(i-1))
		prevC, _ := starlark.AsFloat(close.Index(i-1))

		// Typical price and previous typical price
		tp := (h + l + c) / 3
		prevTP := (prevH + prevL + prevC) / 3

		// Determine trend
		var trend int
		if tp > prevTP {
			trend = 1
		} else {
			trend = -1
		}

		// Calculate volume force
		if i == 1 {
			vf[i] = float64(trend) * v
		} else {
			// Check if trend changed
			prevTrend := 1
			if i >= 2 {
				prevPrevH, _ := starlark.AsFloat(high.Index(i-2))
				prevPrevL, _ := starlark.AsFloat(low.Index(i-2))
				prevPrevC, _ := starlark.AsFloat(close.Index(i-2))
				prevPrevTP := (prevPrevH + prevPrevL + prevPrevC) / 3
				
				if prevTP <= prevPrevTP {
					prevTrend = -1
				}
			}

			if trend == prevTrend {
				vf[i] = vf[i-1] + float64(trend)*v
			} else {
				vf[i] = float64(trend) * v
			}
		}
	}

	// Convert to starlark list for EMA calculation
	vfList := starlark.NewList(nil)
	for _, val := range vf {
		vfList.Append(starlark.Float(val))
	}

	// Calculate fast and slow EMAs
	fastEMA := ti.calculateEMA(vfList, fastPeriod)
	slowEMA := ti.calculateEMA(vfList, slowPeriod)

	// Calculate Klinger Oscillator
	ko := make([]float64, length)
	for i := 0; i < length; i++ {
		if math.IsNaN(fastEMA[i]) || math.IsNaN(slowEMA[i]) {
			ko[i] = math.NaN()
		} else {
			ko[i] = fastEMA[i] - slowEMA[i]
		}
	}

	// Calculate signal line
	koList := starlark.NewList(nil)
	for _, val := range ko {
		if !math.IsNaN(val) {
			koList.Append(starlark.Float(val))
		} else {
			koList.Append(starlark.None)
		}
	}

	signal := ti.calculateEMA(koList, signalPeriod)

	return ko, signal
}

// calculateVolumeProfile calculates Volume Profile (simplified version returning volume at price levels)
func (ti *TechnicalIndicators) calculateVolumeProfile(high, low, close, volume *starlark.List, period int, levels int) map[float64]float64 {
	length := close.Len()
	if length < period {
		return nil
	}

	profile := make(map[float64]float64)
	
	// Take the last 'period' bars for analysis
	startIdx := length - period
	if startIdx < 0 {
		startIdx = 0
	}

	// Find price range
	var minPrice, maxPrice float64 = math.MaxFloat64, -math.MaxFloat64
	for i := startIdx; i < length; i++ {
		h, _ := starlark.AsFloat(high.Index(i))
		l, _ := starlark.AsFloat(low.Index(i))
		
		if h > maxPrice {
			maxPrice = h
		}
		if l < minPrice {
			minPrice = l
		}
	}

	// Create price levels
	priceStep := (maxPrice - minPrice) / float64(levels)
	if priceStep == 0 {
		return profile
	}

	// Initialize volume at each level
	for i := 0; i < levels; i++ {
		priceLevel := minPrice + float64(i)*priceStep
		profile[priceLevel] = 0
	}

	// Distribute volume across price levels
	for i := startIdx; i < length; i++ {
		h, _ := starlark.AsFloat(high.Index(i))
		l, _ := starlark.AsFloat(low.Index(i))
		v, _ := starlark.AsFloat(volume.Index(i))

		// Distribute volume proportionally across the high-low range
		barRange := h - l
		if barRange == 0 {
			// If no range, assign all volume to the close price level
			c, _ := starlark.AsFloat(close.Index(i))
			levelIdx := int((c - minPrice) / priceStep)
			if levelIdx >= 0 && levelIdx < levels {
				priceLevel := minPrice + float64(levelIdx)*priceStep
				profile[priceLevel] += v
			}
		} else {
			// Distribute volume across all price levels within the bar's range
			volumePerLevel := v / (barRange / priceStep)
			
			startLevel := int((l - minPrice) / priceStep)
			endLevel := int((h - minPrice) / priceStep)
			
			if startLevel < 0 {
				startLevel = 0
			}
			if endLevel >= levels {
				endLevel = levels - 1
			}
			
			for levelIdx := startLevel; levelIdx <= endLevel; levelIdx++ {
				priceLevel := minPrice + float64(levelIdx)*priceStep
				profile[priceLevel] += volumePerLevel
			}
		}
	}

	return profile
}

// calculateRVI calculates Relative Vigor Index
func (ti *TechnicalIndicators) calculateRVI(open, high, low, close *starlark.List, period int) ([]float64, []float64) {
	length := close.Len()
	if length < period {
		return nil, nil
	}

	rvi := make([]float64, length)
	signal := make([]float64, length)

	// Calculate numerator and denominator
	numerator := make([]float64, length)
	denominator := make([]float64, length)

	for i := 0; i < length; i++ {
		if i < period-1 {
			rvi[i] = math.NaN()
			signal[i] = math.NaN()
			continue
		}

		o, _ := starlark.AsFloat(open.Index(i))
		h, _ := starlark.AsFloat(high.Index(i))
		l, _ := starlark.AsFloat(low.Index(i))
		c, _ := starlark.AsFloat(close.Index(i))

		numerator[i] = c - o
		denominator[i] = h - l

		// Calculate RVI as SMA ratio
		numSum := 0.0
		denSum := 0.0
		for j := i - period + 1; j <= i; j++ {
			o2, _ := starlark.AsFloat(open.Index(j))
			h2, _ := starlark.AsFloat(high.Index(j))
			l2, _ := starlark.AsFloat(low.Index(j))
			c2, _ := starlark.AsFloat(close.Index(j))

			numSum += c2 - o2
			denSum += h2 - l2
		}

		if denSum != 0 {
			rvi[i] = numSum / denSum
		} else {
			rvi[i] = 0
		}
	}

	// Calculate signal line as SMA of RVI
	for i := 3; i < length; i++ {
		if !math.IsNaN(rvi[i]) && !math.IsNaN(rvi[i-1]) && !math.IsNaN(rvi[i-2]) && !math.IsNaN(rvi[i-3]) {
			signal[i] = (rvi[i] + 2*rvi[i-1] + 2*rvi[i-2] + rvi[i-3]) / 6
		} else {
			signal[i] = math.NaN()
		}
	}

	return rvi, signal
}

// calculatePPO calculates Percentage Price Oscillator
func (ti *TechnicalIndicators) calculatePPO(prices *starlark.List, fastPeriod, slowPeriod, signalPeriod int) ([]float64, []float64, []float64) {
	length := prices.Len()
	if length < slowPeriod {
		return nil, nil, nil
	}

	// Calculate EMAs
	fastEMA := ti.calculateEMA(prices, fastPeriod)
	slowEMA := ti.calculateEMA(prices, slowPeriod)

	// Calculate PPO line (percentage difference)
	ppo := make([]float64, length)
	for i := 0; i < length; i++ {
		if math.IsNaN(fastEMA[i]) || math.IsNaN(slowEMA[i]) || slowEMA[i] == 0 {
			ppo[i] = math.NaN()
		} else {
			ppo[i] = ((fastEMA[i] - slowEMA[i]) / slowEMA[i]) * 100
		}
	}

	// Convert PPO to starlark list for signal calculation
	ppoList := starlark.NewList(nil)
	for _, val := range ppo {
		if !math.IsNaN(val) {
			ppoList.Append(starlark.Float(val))
		} else {
			ppoList.Append(starlark.None)
		}
	}

	// Calculate signal line
	signal := ti.calculateEMA(ppoList, signalPeriod)

	// Calculate histogram
	histogram := make([]float64, length)
	for i := 0; i < length; i++ {
		if math.IsNaN(ppo[i]) || math.IsNaN(signal[i]) {
			histogram[i] = math.NaN()
		} else {
			histogram[i] = ppo[i] - signal[i]
		}
	}

	return ppo, signal, histogram
}

// calculateAccumulationDistribution calculates Accumulation/Distribution Line
func (ti *TechnicalIndicators) calculateAccumulationDistribution(high, low, close, volume *starlark.List) []float64 {
	length := close.Len()
	if length == 0 {
		return nil
	}

	result := make([]float64, length)
	result[0] = 0

	for i := 1; i < length; i++ {
		h, _ := starlark.AsFloat(high.Index(i))
		l, _ := starlark.AsFloat(low.Index(i))
		c, _ := starlark.AsFloat(close.Index(i))
		v, _ := starlark.AsFloat(volume.Index(i))

		// Money Flow Multiplier
		var mfm float64
		if h != l {
			mfm = ((c - l) - (h - c)) / (h - l)
		} else {
			mfm = 0
		}

		// Money Flow Volume
		mfv := mfm * v

		// A/D Line
		result[i] = result[i-1] + mfv
	}

	return result
}

// calculateChaikinMoneyFlow calculates Chaikin Money Flow
func (ti *TechnicalIndicators) calculateChaikinMoneyFlow(high, low, close, volume *starlark.List, period int) []float64 {
	length := close.Len()
	if length < period {
		return nil
	}

	result := make([]float64, length)

	for i := 0; i < length; i++ {
		if i < period-1 {
			result[i] = math.NaN()
			continue
		}

		sumMFV := 0.0 // Sum of Money Flow Volume
		sumVolume := 0.0

		for j := i - period + 1; j <= i; j++ {
			h, _ := starlark.AsFloat(high.Index(j))
			l, _ := starlark.AsFloat(low.Index(j))
			c, _ := starlark.AsFloat(close.Index(j))
			v, _ := starlark.AsFloat(volume.Index(j))

			// Money Flow Multiplier
			var mfm float64
			if h != l {
				mfm = ((c - l) - (h - c)) / (h - l)
			} else {
				mfm = 0
			}

			mfv := mfm * v
			sumMFV += mfv
			sumVolume += v
		}

		if sumVolume != 0 {
			result[i] = sumMFV / sumVolume
		} else {
			result[i] = 0
		}
	}

	return result
}

// calculateLinearRegression calculates Linear Regression
func (ti *TechnicalIndicators) calculateLinearRegression(prices *starlark.List, period int) []float64 {
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

		// Calculate linear regression for the period
		sumX := 0.0
		sumY := 0.0
		sumXY := 0.0
		sumX2 := 0.0

		for j := 0; j < period; j++ {
			x := float64(j)
			y, _ := starlark.AsFloat(prices.Index(i - period + 1 + j))

			sumX += x
			sumY += y
			sumXY += x * y
			sumX2 += x * x
		}

		n := float64(period)
		slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
		intercept := (sumY - slope*sumX) / n

		// Linear regression value at the end of the period
		result[i] = intercept + slope*float64(period-1)
	}

	return result
}

// calculateLinearRegressionSlope calculates Linear Regression Slope
func (ti *TechnicalIndicators) calculateLinearRegressionSlope(prices *starlark.List, period int) []float64 {
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

		// Calculate linear regression slope for the period
		sumX := 0.0
		sumY := 0.0
		sumXY := 0.0
		sumX2 := 0.0

		for j := 0; j < period; j++ {
			x := float64(j)
			y, _ := starlark.AsFloat(prices.Index(i - period + 1 + j))

			sumX += x
			sumY += y
			sumXY += x * y
			sumX2 += x * x
		}

		n := float64(period)
		slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
		result[i] = slope
	}

	return result
}

// calculateCorrelationCoefficient calculates Correlation Coefficient between price and time
func (ti *TechnicalIndicators) calculateCorrelationCoefficient(prices *starlark.List, period int) []float64 {
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

		// Calculate correlation between price and time
		sumX := 0.0
		sumY := 0.0
		sumXY := 0.0
		sumX2 := 0.0
		sumY2 := 0.0

		for j := 0; j < period; j++ {
			x := float64(j)
			y, _ := starlark.AsFloat(prices.Index(i - period + 1 + j))

			sumX += x
			sumY += y
			sumXY += x * y
			sumX2 += x * x
			sumY2 += y * y
		}

		n := float64(period)
		numerator := n*sumXY - sumX*sumY
		denominator := math.Sqrt((n*sumX2 - sumX*sumX) * (n*sumY2 - sumY*sumY))

		if denominator != 0 {
			result[i] = numerator / denominator
		} else {
			result[i] = 0
		}
	}

	return result
}

// calculateBollingerPercentB calculates Bollinger %B
func (ti *TechnicalIndicators) calculateBollingerPercentB(prices *starlark.List, period int, multiplier float64) []float64 {
	length := prices.Len()
	if length < period {
		return nil
	}

	// Calculate Bollinger Bands
	upper, _, lower := ti.calculateBollinger(prices, period, multiplier)

	result := make([]float64, length)
	for i := 0; i < length; i++ {
		if math.IsNaN(upper[i]) || math.IsNaN(lower[i]) {
			result[i] = math.NaN()
			continue
		}

		price, _ := starlark.AsFloat(prices.Index(i))
		bandWidth := upper[i] - lower[i]

		if bandWidth != 0 {
			result[i] = (price - lower[i]) / bandWidth
		} else {
			result[i] = 0.5 // Middle of bands when width is zero
		}
	}

	return result
}

// calculateBollingerBandWidth calculates Bollinger Band Width
func (ti *TechnicalIndicators) calculateBollingerBandWidth(prices *starlark.List, period int, multiplier float64) []float64 {
	length := prices.Len()
	if length < period {
		return nil
	}

	// Calculate Bollinger Bands
	upper, middle, lower := ti.calculateBollinger(prices, period, multiplier)

	result := make([]float64, length)
	for i := 0; i < length; i++ {
		if math.IsNaN(upper[i]) || math.IsNaN(middle[i]) || math.IsNaN(lower[i]) {
			result[i] = math.NaN()
			continue
		}

		bandWidth := upper[i] - lower[i]
		if middle[i] != 0 {
			result[i] = bandWidth / middle[i]
		} else {
			result[i] = 0
		}
	}

	return result
}

// calculateStandardError calculates Standard Error of Linear Regression
func (ti *TechnicalIndicators) calculateStandardError(prices *starlark.List, period int) []float64 {
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

		// Calculate linear regression
		sumX := 0.0
		sumY := 0.0
		sumXY := 0.0
		sumX2 := 0.0

		for j := 0; j < period; j++ {
			x := float64(j)
			y, _ := starlark.AsFloat(prices.Index(i - period + 1 + j))

			sumX += x
			sumY += y
			sumXY += x * y
			sumX2 += x * x
		}

		n := float64(period)
		slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
		intercept := (sumY - slope*sumX) / n

		// Calculate sum of squared residuals
		sumSquaredResiduals := 0.0
		for j := 0; j < period; j++ {
			x := float64(j)
			y, _ := starlark.AsFloat(prices.Index(i - period + 1 + j))
			predicted := intercept + slope*x
			residual := y - predicted
			sumSquaredResiduals += residual * residual
		}

		// Standard error calculation
		if n > 2 {
			result[i] = math.Sqrt(sumSquaredResiduals / (n - 2))
		} else {
			result[i] = 0
		}
	}

	return result
}

// calculateWilliamsAD calculates Williams Accumulation/Distribution
func (ti *TechnicalIndicators) calculateWilliamsAD(high, low, close *starlark.List) []float64 {
	length := close.Len()
	if length < 2 {
		return nil
	}

	result := make([]float64, length)
	result[0] = 0

	for i := 1; i < length; i++ {
		h, _ := starlark.AsFloat(high.Index(i))
		l, _ := starlark.AsFloat(low.Index(i))
		c, _ := starlark.AsFloat(close.Index(i))
		prevC, _ := starlark.AsFloat(close.Index(i - 1))

		// True Range High and Low
		trueHigh := math.Max(h, prevC)
		trueLow := math.Min(l, prevC)

		// Williams A/D calculation
		if c > prevC {
			result[i] = result[i-1] + (c - trueLow)
		} else if c < prevC {
			result[i] = result[i-1] + (c - trueHigh)
		} else {
			result[i] = result[i-1]
		}
	}

	return result
}

// calculateMoneyFlowVolume calculates Money Flow Volume
func (ti *TechnicalIndicators) calculateMoneyFlowVolume(high, low, close, volume *starlark.List) []float64 {
	length := close.Len()
	if length == 0 {
		return nil
	}

	result := make([]float64, length)

	for i := 0; i < length; i++ {
		h, _ := starlark.AsFloat(high.Index(i))
		l, _ := starlark.AsFloat(low.Index(i))
		c, _ := starlark.AsFloat(close.Index(i))
		v, _ := starlark.AsFloat(volume.Index(i))

		// Money Flow Multiplier
		var mfm float64
		if h != l {
			mfm = ((c - l) - (h - c)) / (h - l)
		} else {
			mfm = 0
		}

		// Money Flow Volume
		result[i] = mfm * v
	}

	return result
}

// calculatePriceROC calculates Price Rate of Change
func (ti *TechnicalIndicators) calculatePriceROC(prices *starlark.List, period int) []float64 {
	length := prices.Len()
	if length < period+1 {
		return nil
	}

	result := make([]float64, length)

	for i := 0; i < length; i++ {
		if i < period {
			result[i] = math.NaN()
			continue
		}

		currentPrice, _ := starlark.AsFloat(prices.Index(i))
		pastPrice, _ := starlark.AsFloat(prices.Index(i - period))

		if pastPrice != 0 {
			result[i] = ((currentPrice - pastPrice) / pastPrice) * 100
		} else {
			result[i] = 0
		}
	}

	return result
}

// calculateVolatilityIndex calculates a custom volatility index
func (ti *TechnicalIndicators) calculateVolatilityIndex(high, low, close *starlark.List, period int) []float64 {
	length := close.Len()
	if length < period {
		return nil
	}

	result := make([]float64, length)

	for i := 0; i < length; i++ {
		if i < period-1 {
			result[i] = math.NaN()
			continue
		}

		// Calculate volatility as standard deviation of returns
		returns := make([]float64, period-1)
		for j := 1; j < period; j++ {
			idx := i - period + 1 + j
			current, _ := starlark.AsFloat(close.Index(idx))
			previous, _ := starlark.AsFloat(close.Index(idx - 1))
			
			if previous != 0 {
				returns[j-1] = math.Log(current / previous)
			} else {
				returns[j-1] = 0
			}
		}

		// Calculate mean return
		mean := 0.0
		for _, ret := range returns {
			mean += ret
		}
		mean /= float64(len(returns))

		// Calculate standard deviation
		variance := 0.0
		for _, ret := range returns {
			variance += math.Pow(ret-mean, 2)
		}
		variance /= float64(len(returns))

		// Annualized volatility (assuming daily data)
		result[i] = math.Sqrt(variance) * math.Sqrt(252) * 100
	}

	return result
}
