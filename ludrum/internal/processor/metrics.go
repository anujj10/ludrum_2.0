package processor

import "ludrum/internal/models"

// ==========================================
// ⚡ LTP VELOCITY (last move speed)
// ==========================================
func calculateLTPVelocity(data []models.TickData) float64 {
	n := len(data)
	if n < 2 {
		return 0
	}

	last := data[n-1].LTP
	prev := data[n-2].LTP

	return last - prev
}

// ==========================================
// 🚀 LTP ACCELERATION (change in velocity)
// ==========================================
func calculateLTPAcceleration(data []models.TickData) float64 {
	n := len(data)
	if n < 3 {
		return 0
	}

	v1 := data[n-1].LTP - data[n-2].LTP
	v2 := data[n-2].LTP - data[n-3].LTP

	return v1 - v2
}

// ==========================================
// 💰 OI MOMENTUM
// ==========================================
func calculateOIMomentum(data []models.TickData) float64 {
	n := len(data)
	if n < 2 {
		return 0
	}

	return float64(data[n-1].OI - data[n-2].OI)
}

// ==========================================
// 🔥 VOLUME SPIKE
// ==========================================
func calculateVolumeSpike(data []models.TickData) float64 {
	n := len(data)
	if n < 2 {
		return 0
	}

	last := float64(data[n-1].Volume)
	prev := float64(data[n-2].Volume)

	if prev == 0 {
		return 0
	}

	return (last - prev) / prev
}