package processor

import (
	"ludrum/internal/models"
	"math"
)

type Key struct {
    Strike float64
    Type   uint8 // 0 = CE, 1 = PE
}

type LastState struct {
    LTP    float64
    OI     int64
    Volume int64
}

type ChangeDetector struct {
    state map[Key]LastState
}

func NewChangeDetector() *ChangeDetector {
    return &ChangeDetector{
        state: make(map[Key]LastState),
    }
}

const (
    ChangeNone = 0
    ChangeLTP  = 1 << 0
    ChangeOI   = 1 << 1
    ChangeVol  = 1 << 2
)

func (cd *ChangeDetector) Detect(
    key Key,
    ltp float64,
    oi int64,
    vol int64,
) uint8 {

    prev, exists := cd.state[key]

    var change uint8 = ChangeNone

    if !exists {
        change = ChangeLTP | ChangeOI | ChangeVol
    } else {
        if prev.LTP != ltp {
            change |= ChangeLTP
        }
        if prev.OI != oi {
            change |= ChangeOI
        }
        if prev.Volume != vol {
            change |= ChangeVol
        }
    }

    if change != ChangeNone {
        cd.state[key] = LastState{
            LTP: ltp,
            OI: oi,
            Volume: vol,
        }
    }

    return change
}

type ChangedStrike struct {
    Strike float64
    Type   uint8

    LTP    float64
    OI     int64
    Volume int64

    ChangeMask uint8
}

func (cd *ChangeDetector) Filter(
    snapshot *models.MarketSnapshot,
    strikeRange float64,
) []ChangedStrike {

    spot := snapshot.SpotPrice
    result := make([]ChangedStrike, 0, 64)

    for strike, sd := range snapshot.Strikes {

        if math.Abs(strike-spot) > strikeRange {
            continue
        }

        if sd.CE != nil {
            key := Key{Strike: strike, Type: 0}

            mask := cd.Detect(key, sd.CE.LTP, sd.CE.OI, sd.CE.Volume)

            if mask != ChangeNone {
                result = append(result, ChangedStrike{
                    Strike: strike,
                    Type: 0,
                    LTP: sd.CE.LTP,
                    OI: sd.CE.OI,
                    Volume: sd.CE.Volume,
                    ChangeMask: mask,
                })
            }
        }

        if sd.PE != nil {
            key := Key{Strike: strike, Type: 1}

            mask := cd.Detect(key, sd.PE.LTP, sd.PE.OI, sd.PE.Volume)

            if mask != ChangeNone {
                result = append(result, ChangedStrike{
                    Strike: strike,
                    Type: 1,
                    LTP: sd.PE.LTP,
                    OI: sd.PE.OI,
                    Volume: sd.PE.Volume,
                    ChangeMask: mask,
                })
            }
        }
    }

    return result
}