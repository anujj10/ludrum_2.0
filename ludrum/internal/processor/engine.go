package processor

import (
	"ludrum/internal/models"
)

type Engine struct {
	Data      map[float64]*models.StrikeBuffer
	tickCount int
}

func NewEngine() *Engine {
	return &Engine{
		Data: make(map[float64]*models.StrikeBuffer),
	}
}

// ==========================================
// ✅ Add ONLY changed ticks (NO append, NO trim)
// ==========================================
func (e *Engine) AddChangedTicks(changed []ChangedStrike) {

	for _, cs := range changed {

		if _, ok := e.Data[cs.Strike]; !ok {
			e.Data[cs.Strike] = &models.StrikeBuffer{}
		}

		buffer := e.Data[cs.Strike]

		tick := models.TickData{
			Volume: cs.Volume,
			OI:     cs.OI,
			LTP:    cs.LTP,
		}

if cs.Type == 0 {
    buffer.CE.Add(tick)

    // 🔥 OI EVENT TRACKING
    if len(buffer.CEOIEvents) == 0 ||
        buffer.CEOIEvents[len(buffer.CEOIEvents)-1].OI != cs.OI {

        buffer.CEOIEvents = append(buffer.CEOIEvents, models.TickData{
            Timestamp: tick.Timestamp,
            OI:        cs.OI,
        })

		const maxOIEvents = 10

		if len(buffer.CEOIEvents) > maxOIEvents {
			buffer.CEOIEvents = buffer.CEOIEvents[1:]
		}
    }

} else {
    buffer.PE.Add(tick)

    // 🔥 OI EVENT TRACKING
    if len(buffer.PEOIEvents) == 0 ||
        buffer.PEOIEvents[len(buffer.PEOIEvents)-1].OI != cs.OI {

        buffer.PEOIEvents = append(buffer.PEOIEvents, models.TickData{
            Timestamp: tick.Timestamp,
            OI:        cs.OI,
        })
    }
}
	}

	e.tickCount++
}

// ==========================================
// ✅ Analyze ONLY changed strikes
// ==========================================
func (e *Engine) AnalyzeChanged(
	changed []ChangedStrike,
) []models.StrikeAnalytics {

	result := make([]models.StrikeAnalytics, 0, len(changed))

	for _, cs := range changed {

		buffer := e.Data[cs.Strike]
		if buffer == nil {
			continue
		}

		// =====================
		// CE
		// =====================
		if cs.Type == 0 && buffer.CE.Size > 1 {

			data := buffer.CE.GetAll()

			vol, oi, _ := calculateChange(data)
			ltpSeries := extractLTPSeries(data)
			ltpDeltas := calculateLTPDeltas(data)
			ltpPattern := ltpDirectionPattern(data)
			ltpWindow := calculateLTPWindowChange(data)

			velocity := calculateLTPVelocity(data)
			acceleration := calculateLTPAcceleration(data)
			oiMomentum := calculateOIMomentum(data)
			volSpike := calculateVolumeSpike(data)

			latest, _ := buffer.CE.Last()

			result = append(result, models.StrikeAnalytics{
				Strike: cs.Strike,
				Type:   "CE",

				VolumeChange: vol,
				OIChange:     oi,
				LTPChange:    ltpWindow,
				CurrentOI:    latest.OI,

				LTPSeries:  ltpSeries,
				LTPDeltas:  ltpDeltas,
				LTPPattern: ltpPattern,

				Velocity:     velocity,
				Acceleration: acceleration,
				OIMomentum:   oiMomentum,
				VolumeSpike:  volSpike,

			Signal: detectAdvancedSignal(
					ltpWindow,
					oi,
					velocity,
					acceleration,
					oiMomentum,
					volSpike,
				),
			})
		}

		// =====================
		// PE
		// =====================
		if cs.Type == 1 && buffer.PE.Size > 1 {

			data := buffer.PE.GetAll()

			vol, oi, _ := calculateChange(data)
			ltpSeries := extractLTPSeries(data)
			ltpDeltas := calculateLTPDeltas(data)
			ltpPattern := ltpDirectionPattern(data)
			ltpWindow := calculateLTPWindowChange(data)

			velocity := calculateLTPVelocity(data)
			acceleration := calculateLTPAcceleration(data)
			oiMomentum := calculateOIMomentum(data)
			volSpike := calculateVolumeSpike(data)

			latest, _ := buffer.PE.Last()

			result = append(result, models.StrikeAnalytics{
				Strike: cs.Strike,
				Type:   "PE",

				VolumeChange: vol,
				OIChange:     oi,
				LTPChange:    ltpWindow,
				CurrentOI:    latest.OI,

				LTPSeries:  ltpSeries,
				LTPDeltas:  ltpDeltas,
				LTPPattern: ltpPattern,

				Velocity:     velocity,
				Acceleration: acceleration,
				OIMomentum:   oiMomentum,
				VolumeSpike:  volSpike,

			Signal: detectAdvancedSignal(
					ltpWindow,
					oi,
					velocity,
					acceleration,
					oiMomentum,
					volSpike,
				),
			})
		}
	}

	return result
}

// ==========================================
// 🔧 Cleanup
// ==========================================
func (e *Engine) Cleanup(spot float64, cleanupRange float64) {

	for strike := range e.Data {
		if abs(strike-spot) > cleanupRange {
			delete(e.Data, strike)
		}
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}


func extractOIChangeSeries(events []models.TickData) []int64 {

    if len(events) < 2 {
        return nil
    }

    var result []int64

    for i := 1; i < len(events); i++ {
        change := events[i].OI - events[i-1].OI
        result = append(result, change)
    }
    return result
}