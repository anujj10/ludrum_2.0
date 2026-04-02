package entryengine

import "ludrum/internal/models"

// Last N movement pattern
func getLastMoves(a models.StrikeAnalytics, n int) []string {
	if len(a.LTPPattern) < n {
		return nil
	}
	return a.LTPPattern[len(a.LTPPattern)-n:]
}

// Detect pullback: UP → DOWN → UP
func isBullishPullback(a models.StrikeAnalytics) bool {
	moves := getLastMoves(a, 3)
	if moves == nil {
		return false
	}

	return moves[0] == "UP" &&
		moves[1] == "DOWN" &&
		moves[2] == "UP"
}

// Detect overextension
func isOverExtended(a models.StrikeAnalytics) bool {
	moves := getLastMoves(a, 4)
	if moves == nil {
		return false
	}

	return moves[0] == "UP" &&
		moves[1] == "UP" &&
		moves[2] == "UP" &&
		moves[3] == "UP"
}