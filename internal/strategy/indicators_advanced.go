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
