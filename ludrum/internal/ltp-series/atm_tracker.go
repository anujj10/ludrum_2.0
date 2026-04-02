package ltpSeries

import "math"

type ATMTracker struct {
	lastATM float64
	step    float64
}

func NewATMTracker(step float64) *ATMTracker {
	return &ATMTracker{
		step: step,
	}
}

func (a *ATMTracker) Update(ltp float64) (float64, bool) {
newATM := math.Round(ltp/a.step) * a.step

// 🔥 HYSTERESIS BUFFER (25 pts)
if a.lastATM != 0 {
	if math.Abs(newATM-a.lastATM) < (a.step / 2) {
		return a.lastATM, false
	}
}

if newATM != a.lastATM {
	a.lastATM = newATM
	return newATM, true
}

return newATM, false

}