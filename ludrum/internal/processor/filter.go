package processor

import "math"

func isNearATM(strike float64, atm float64) bool{
	return math.Abs(strike-atm) <= 200
}