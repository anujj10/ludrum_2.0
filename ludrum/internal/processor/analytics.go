package processor

import (
	"ludrum/internal/models"
)

func calculateChange(data []models.TickData) (int64, int64, float64)  {
	if len(data) < 2 {
		return 0, 0, 0
	}

	first:= data[0]
	last:= data[len(data) - 1]

	volChange:= last.Volume - first.Volume
	oiChange:= last.OI - first.OI
	ltpChange:= last.LTP - first.LTP

// 	log.Printf("First: V:%d OI:%d LTP:%.2f | Last: V:%d OI:%d LTP:%.2f",
// 	first.Volume, first.OI, first.LTP,
// 	last.Volume, last.OI, last.LTP,
// )

	return volChange, oiChange, ltpChange
}

func extractLTPSeries(data []models.TickData) []float64 {
	ltps := make([]float64, len(data))

	for i, d := range data {
		ltps[i] = d.LTP
	}

	return ltps
}

func calculateLTPDeltas(data []models.TickData) []float64 {

	if len(data) < 2 {
		return nil
	}

	var deltas []float64

	for i := 1; i < len(data); i++ {
		diff := data[i].LTP - data[i-1].LTP
		deltas = append(deltas, diff)
	}

	return deltas
}

func calculateLTPWindowChange(data []models.TickData) float64 {

	if len(data) < 2 {
		return 0
	}

	return data[len(data)-1].LTP - data[0].LTP
}


func ltpDirectionPattern(data []models.TickData) []string {

	var pattern []string

	for i := 1; i < len(data); i++ {
		if data[i].LTP > data[i-1].LTP {
			pattern = append(pattern, "UP")
		} else if data[i].LTP < data[i-1].LTP {
			pattern = append(pattern, "DOWN")
		} else {
			pattern = append(pattern, "FLAT")
		}
	}

	return pattern
}