package signalengine

import "ludrum/internal/models"

// CE should show momentum
func isLTPIncreasing(a models.StrikeAnalytics) bool {
	if len(a.LTPDeltas) < 2 {
		return false
	}

	n := len(a.LTPDeltas)

	return a.LTPDeltas[n-1] > 0 &&
		a.LTPDeltas[n-2] > 0
}

// PE should be weak or flat
func isLTPDecreasingOrFlat(a models.StrikeAnalytics) bool {
	if len(a.LTPDeltas) < 1 {
		return false
	}

	n := len(a.LTPDeltas)

	return a.LTPDeltas[n-1] <= 0
}