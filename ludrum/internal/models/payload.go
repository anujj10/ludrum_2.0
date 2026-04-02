package models

import (
	"ludrum/internal/simulator"
)

type StreamPayload struct {
	Spot float64 `json:"spot"`

	Pairs []PairSignal `json:"pairs"`
	Type string `json:"type"`

	OpenPositions   []simulator.Position  `json:"open_positions"`
	ClosedPositions []*simulator.Position `json:"closed_positions"`

	Portfolio simulator.Portfolio `json:"portfolio"`
}