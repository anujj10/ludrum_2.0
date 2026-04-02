package models

type TickData struct {
	Timestamp int64
	Volume    int64
	OI        int64
	LTP       float64
}

type StrikeBuffer struct {
	CE RingBuffer
	PE RingBuffer

	CEOIEvents []TickData
    PEOIEvents []TickData
}


type StrikeAnalytics struct {
	Strike float64
	Type   string

	VolumeChange int64
	OIChange     int64
	LTPChange    float64
	CurrentOI	int64

	LTPSeries   []float64
	LTPDeltas   []float64
	LTPPattern  []string

	Velocity     float64
	Acceleration float64
	OIMomentum   float64
	VolumeSpike  float64

	Signal    string
	Highlight bool
}

type PairSignal struct {
	Strike float64

	CE 		StrikeAnalytics
	PE 		StrikeAnalytics

	Bias     string
	Score    float64
	Strength string
}

// ==========================================
// 🔁 RING BUFFER (FIXED SIZE = 5)
// ==========================================
type RingBuffer struct {
	Data  [5]TickData
	Index int
	Size  int
}

func (rb *RingBuffer) Add(t TickData) {
	rb.Data[rb.Index] = t
	rb.Index = (rb.Index + 1) % len(rb.Data)

	if rb.Size < len(rb.Data) {
		rb.Size++
	}
}

// Get last N elements in correct order
func (rb *RingBuffer) GetAll() []TickData {
	result := make([]TickData, rb.Size)

	start := (rb.Index - rb.Size + len(rb.Data)) % len(rb.Data)

	for i := 0; i < rb.Size; i++ {
		idx := (start + i) % len(rb.Data)
		result[i] = rb.Data[idx]
	}

	return result
}

// Get last element
func (rb *RingBuffer) Last() (TickData, bool) {
	if rb.Size == 0 {
		return TickData{}, false
	}

	idx := (rb.Index - 1 + len(rb.Data)) % len(rb.Data)
	return rb.Data[idx], true
}

// Get previous element
func (rb *RingBuffer) Prev() (TickData, bool) {
	if rb.Size < 2 {
		return TickData{}, false
	}

	idx := (rb.Index - 2 + len(rb.Data)) % len(rb.Data)
	return rb.Data[idx], true
}

