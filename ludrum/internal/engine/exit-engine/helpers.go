package exitengine

import "ludrum/internal/models"

// last N moves
func getLastMoves(a models.StrikeAnalytics, n int) []string {
	if len(a.LTPPattern) < n {
		return nil
	}
	return a.LTPPattern[len(a.LTPPattern)-n:]
}

// detect momentum fading
func isMomentumFading(a models.StrikeAnalytics) bool {
	moves := getLastMoves(a, 3)
	if moves == nil {
		return false
	}

	// UP → FLAT → DOWN OR UP → DOWN → DOWN
	return (moves[0] == "UP" && moves[1] == "FLAT" && moves[2] == "DOWN") ||
		(moves[0] == "UP" && moves[1] == "DOWN" && moves[2] == "DOWN")
}

// trailing weakness
func isWeakContinuation(a models.StrikeAnalytics) bool {
	moves := getLastMoves(a, 2)
	if moves == nil {
		return false
	}

	return moves[0] == "UP" && moves[1] == "FLAT"
}