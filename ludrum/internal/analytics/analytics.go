package analytics

import (
	"ludrum/internal/models"
)

type PositionType string

const (
	LongBuildUp   PositionType = "LONG_BUILDUP"
	ShortBuildUp  PositionType = "SHORT_BUILDUP"
	ShortCovering PositionType = "SHORT_COVERING"
	LongUnwinding PositionType = "LONG_UNWINDING"
	Neutral       PositionType = "NEUTRAL"
)

type Signal struct {
	PCR        float64
	Support    float64
	Resistance float64
}

type SmartMoneySignal struct {
	StrongLongs  []float64
	StrongShorts []float64
}

type StrikePosition struct {
	Strike float64

	CE PositionType
	PE PositionType
}

type MarketSignal struct {
	Bias        string
	Support     float64
	Resistance  float64
	KeyZoneLow  float64
	KeyZoneHigh float64
	Confidence  float64
	Strength    string
}

type TradeAction struct {
	Action     string  // BUY / SELL / WAIT / AVOID
	Entry      float64
	StopLoss   float64
	Reason     string
}

type ShortTermTrend string

const (
	TrendBullish ShortTermTrend = "BULLISH"
	TrendBearish ShortTermTrend = "BEARISH"
	TrendNeutral ShortTermTrend = "NEUTRAL"
)

// DetectPosition determines position type using OI + price change
func DetectSignal(opt *models.OptionChain) PositionType {

	if opt == nil {
		return Neutral
	}

	if opt.OICh > 0 && opt.LTPCh > 0 {
		return LongBuildUp
	}

	if opt.OICh > 0 && opt.LTPCh < 0 {
		return ShortBuildUp
	}

	if opt.OICh < 0 && opt.LTPCh > 0 {
		return ShortCovering
	}

	if opt.OICh < 0 && opt.LTPCh < 0 {
		return LongUnwinding
	}

	return Neutral
}

func DetectPosition(strikes map[float64]*models.StrikeData) []StrikePosition {

	var result []StrikePosition

	for strike, data := range strikes {

		pos := StrikePosition{
			Strike: strike,
			CE:     DetectSignal(data.CE),
			PE:     DetectSignal(data.PE),
		}

		result = append(result, pos)
	}

	return result
}

// GenerateSignals calculates PCR, support, resistance
func GenerateSignal(snapshot *models.MarketSnapshot) *Signal {

	var maxPE_OI int64
	var maxCE_OI int64

	var support float64
	var resistance float64

	for strike, data := range snapshot.Strikes {

		if data.PE != nil && data.PE.OI > maxPE_OI {
			maxPE_OI = data.PE.OI
			support = strike
		}

		if data.CE != nil && data.CE.OI > maxCE_OI {
			maxCE_OI = data.CE.OI
			resistance = strike
		}
	}

	pcr := 0.0
	if snapshot.TotalCallOI > 0 {
		pcr = float64(snapshot.TotalPutOI) / float64(snapshot.TotalCallOI)
	}

	return &Signal{
		PCR:        pcr,
		Support:    support,
		Resistance: resistance,
	}
}
/*
func GenerateFinalSignal(snap *models.MarketSnapshot) MarketSignal {

	atm := snap.SpotPrice

	var maxSupport float64
	var maxResistance float64

	var maxSupportScore float64
	var maxResistanceScore float64

	totalScore := 0.0
	count := 0.0

	for strike, data := range snap.Strikes {

		// 🎯 Focus near ATM (VERY IMPORTANT)
		if math.Abs(strike-atm) > 200 {
			continue
		}

		if data.CE == nil || data.PE == nil {
			continue
		}

		// 🔥 Score calculation
		score := float64(data.PE.OI-data.CE.OI) +
			float64(data.PE.OICh-data.CE.OICh)

		totalOI := float64(data.CE.OI + data.PE.OI)

		if totalOI < 1000000 { // filter noise
			continue
		}

		// 📈 Support detection
		if score > maxSupportScore {
			maxSupportScore = score
			maxSupport = strike
		}

		// 📉 Resistance detection
		if score < maxResistanceScore {
			maxResistanceScore = score
			maxResistance = strike
		}

		totalScore += score
		count++
	}

	// 🔥 PCR-based bias
	pcr := float64(snap.TotalPutOI) / float64(snap.TotalCallOI)

	bias := "NEUTRAL"
	if pcr > 1.1 {
		bias = "BULLISH"
	} else if pcr < 0.9 {
		bias = "BEARISH"
	}

	// 🔥 Confidence (normalized)
	confidence := math.Min(math.Abs(totalScore)/(count*1000000), 1)

	// 🔥 Strength
	strength := "WEAK"
	if confidence > 0.7 {
		strength = "STRONG"
	} else if confidence > 0.4 {
		strength = "MODERATE"
	}

	// 🎯 Key zone (between support & resistance)
	keyLow := math.Min(maxSupport, maxResistance)
	keyHigh := math.Max(maxSupport, maxResistance)

	return MarketSignal{
		Bias:        bias,
		Support:     maxSupport,
		Resistance:  maxResistance,
		KeyZoneLow:  keyLow,
		KeyZoneHigh: keyHigh,
		Confidence:  confidence,
		Strength:    strength,
	}
}

func ExecuteTrade(snap *models.MarketSnapshot, signal MarketSignal) TradeAction {

	spot := snap.SpotPrice

	// 🎯 Zones
	support := signal.Support
	resistance := signal.Resistance

	// distance thresholds
	nearSupport := math.Abs(spot-support) < 50
	nearResistance := math.Abs(spot-resistance) < 50

	// 🔥 1. TRAP DETECTION (MOST IMPORTANT)
	if nearResistance && signal.Bias == "BULLISH" {
		return TradeAction{
			Action: "AVOID",
			Reason: "Near resistance, possible rejection / trap",
		}
	}

	if nearSupport && signal.Bias == "BEARISH" {
		return TradeAction{
			Action: "AVOID",
			Reason: "Near support, possible bounce trap",
		}
	}

	// 🔥 2. BUY LOGIC (only near support)
	if signal.Bias == "BULLISH" && nearSupport {

		return TradeAction{
			Action:   "BUY",
			Entry:    spot,
			StopLoss: support - 30,
			Reason:   "Bounce from support with bullish bias",
		}
	}

	// 🔥 3. SELL LOGIC (only near resistance)
	if signal.Bias == "BEARISH" && nearResistance {

		return TradeAction{
			Action:   "SELL",
			Entry:    spot,
			StopLoss: resistance + 30,
			Reason:   "Rejection from resistance with bearish bias",
		}
	}

	// 🔥 4. BREAKOUT LOGIC (SAFE ENTRY)

	if signal.Bias == "BULLISH" && spot > resistance {

		return TradeAction{
			Action:   "BUY",
			Entry:    spot,
			StopLoss: resistance - 30,
			Reason:   "Confirmed breakout above resistance",
		}
	}

	if signal.Bias == "BEARISH" && spot < support {

		return TradeAction{
			Action:   "SELL",
			Entry:    spot,
			StopLoss: support + 30,
			Reason:   "Breakdown below support",
		}
	}

	// 🔥 DEFAULT → WAIT
	return TradeAction{
		Action: "WAIT",
		Reason: "No clean setup",
	}
}

func GetShortTermTrend(current, prev float64) ShortTermTrend {

	if current > prev {
		return TrendBullish
	}

	if current < prev {
		return TrendBearish
	}

	return TrendNeutral
}

func IsPullback(bias string, trend ShortTermTrend) bool {

	if bias == "BULLISH" && trend == TrendBearish {
		return true
	}

	if bias == "BEARISH" && trend == TrendBullish {
		return true
	}

	return false
}

func IsExtendedMove(spot, localResistance float64) bool {

	// 🔥 if price too close to resistance → avoid
	if math.Abs(spot-localResistance) < 30 {
		return true
	}

	return false
}

func ExecuteTradeV2(
	snap *models.MarketSnapshot,
	signal MarketSignal,
	prevSpot float64,
	localSupport float64,
	localResistance float64,
) TradeAction {

	spot := snap.SpotPrice

	// 🔥 1. Short-term trend
	trend := GetShortTermTrend(spot, prevSpot)

	// 🔥 2. Pullback check
	pullback := IsPullback(signal.Bias, trend)

	// 🔥 3. Extension check
	extended := IsExtendedMove(spot, localResistance)

	// 🔥 4. Distance checks
	nearSupport := math.Abs(spot-localSupport) < 50
	nearResistance := math.Abs(spot-localResistance) < 50

	// 💀 HARD BLOCKS

	if signal.Confidence < 0.4 {
		return TradeAction{
			Action: "WAIT",
			Reason: "Low confidence",
		}
	}

	if extended {
		return TradeAction{
			Action: "AVOID",
			Reason: "Price too extended near resistance",
		}
	}

	// 🔥 PULLBACK LOGIC (BEST ENTRY)

	if signal.Bias == "BULLISH" && pullback && nearSupport {
		return TradeAction{
			Action:   "BUY",
			Entry:    spot,
			StopLoss: localSupport - 30,
			Reason:   "Pullback in bullish trend near support",
		}
	}

	if signal.Bias == "BEARISH" && pullback && nearResistance {
		return TradeAction{
			Action:   "SELL",
			Entry:    spot,
			StopLoss: localResistance + 30,
			Reason:   "Pullback in bearish trend near resistance",
		}
	}

	// 🔥 BREAKOUT (ONLY IF CONFIRMED)

	if signal.Bias == "BULLISH" && spot > localResistance && trend == TrendBullish {
		return TradeAction{
			Action:   "BUY",
			Entry:    spot,
			StopLoss: localResistance - 30,
			Reason:   "Confirmed breakout with momentum",
		}
	}

	if signal.Bias == "BEARISH" && spot < localSupport && trend == TrendBearish {
		return TradeAction{
			Action:   "SELL",
			Entry:    spot,
			StopLoss: localSupport + 30,
			Reason:   "Breakdown with momentum",
		}
	}

	// 🔥 DEFAULT

	return TradeAction{
		Action: "WAIT",
		Reason: "No clean setup (waiting for pullback or breakout)",
	}
}

func GetLocalStrikes(strikes map[float64]*models.StrikeData, spot float64) map[float64]*models.StrikeData {

	local := make(map[float64]*models.StrikeData)

	for strike, data := range strikes {

		if math.Abs(strike-spot) <= 100 { // 🔥 range
			local[strike] = data
		}
	}

	return local
}

func GetLocalLevels(strikes map[float64]*models.StrikeData, spot float64) (float64, float64) {

	var bestSupport float64
	var bestResistance float64

	bestSupportScore := -math.MaxFloat64
	bestResistanceScore := math.MaxFloat64

	for strike, data := range strikes {

		if data.CE == nil || data.PE == nil {
			continue
		}

		distance := math.Abs(strike - spot)

		// 🔥 Ignore too far strikes (>80 pts)
		if distance > 80 {
			continue
		}

		// 🔥 Base score
		score := float64(data.PE.OI - data.CE.OI)

		// 🔥 Distance penalty (KEY FIX)
		adjustedScore := score - (distance * 5000)

		// Support
		if adjustedScore > bestSupportScore {
			bestSupportScore = adjustedScore
			bestSupport = strike
		}

		// Resistance
		if adjustedScore < bestResistanceScore {
			bestResistanceScore = adjustedScore
			bestResistance = strike
		}
	}

	return bestSupport, bestResistance
}

func GetDynamicBias(snap *models.MarketSnapshot) (string, float64) {

	var bullScore float64
	var bearScore float64

	for strike, data := range snap.Strikes {

		if data.CE == nil || data.PE == nil {
			continue
		}

		distance := math.Abs(strike - snap.SpotPrice)

		// ⚡ focus only near strikes (IMPORTANT)
		if distance > 100 {
			continue
		}

		// 🔥 OI change matters more than total OI
		bullScore += float64(data.PE.OICh)
		bearScore += float64(data.CE.OICh)
	}

	// normalize
	if bullScore > bearScore {
		return "BULLISH", bullScore / (bearScore + 1)
	}

	return "BEARISH", bearScore / (bullScore + 1)
}



Detec/tSmartMoney identifies strong positioning across strikes
func DetectSmartMoney(snapshot *models.MarketSnapshot) *SmartMoneySignal {

	longs := make([]float64, 0)
	shorts := make([]float64, 0)

	for strike, data := range snapshot.Strikes {

		// CE side → short build-up = resistance
if data.CE != nil {
	pos := DetectSignal(data.CE)
	if pos == ShortBuildUp {
		shorts = append(shorts, strike)
	}
}

// PE side → long build-up = support
if data.PE != nil {
	pos := DetectSignal(data.PE)
	if pos == LongBuildUp {
		longs = append(longs, strike)
	}
}
}

	return &SmartMoneySignal{
		StrongLongs:  longs,
		StrongShorts: shorts,
	}
}

*/