package scoringengine

type ScoreResult struct {
	Score      float64  `json:"score"`
	Confidence float64  `json:"confidence"`
	Reasons    []string `json:"reasons"`
}