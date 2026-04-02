package processor

func detectAdvancedSignal(
	ltpChange float64,
	oiChange int64,
	velocity float64,
	acceleration float64,
	oiMomentum float64,
	volSpike float64,
) string {

	// =========================
	// 🚀 LONG BUILDUP
	// =========================
	if ltpChange > 0 &&
		oiChange > 0 &&
		velocity > 0 &&
		oiMomentum > 0 {

		return "LONG_BUILDUP"
	}

	// =========================
	// 💣 SHORT BUILDUP
	// =========================
	if ltpChange < 0 &&
		oiChange > 0 &&
		velocity < 0 &&
		oiMomentum > 0 {

		return "SHORT_BUILDUP"
	}

	// =========================
	// 🔥 SHORT COVERING
	// =========================
	if ltpChange > 0 &&
		oiChange < 0 &&
		velocity > 0 {

		return "SHORT_COVERING"
	}

	// =========================
	// 🧊 LONG UNWINDING
	// =========================
	if ltpChange < 0 &&
		oiChange < 0 &&
		velocity < 0 {

		return "LONG_UNWINDING"
	}

	// =========================
	// ⚡ MOMENTUM BURST
	// =========================
	if acceleration > 0 &&
		volSpike > 0.2 {

		return "MOMENTUM_BURST"
	}

	return "NEUTRAL"
}