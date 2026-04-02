package simulator

import (
	"errors"
	"time"
)

const LotSize = 65
const MarginFactor = 1.5

type TradeRequest struct {
	Symbol     string
	Strike     float64
	OptionType string
	Side       string // BUY / SELL
	Lots       int

	SL     *float64
	Target *float64

	EnableSL     bool
	EnableTarget bool
}

func (s *Simulator) Execute(req TradeRequest, price float64) error {
	qty := req.Lots * LotSize
	var sl *float64
	var target *float64

	if req.EnableSL && req.SL != nil {
		value := price
		if req.Side == "SELL" {
			value += *req.SL
		} else {
			value -= *req.SL
		}
		sl = &value
	}

	if req.EnableTarget && req.Target != nil {
		value := price
		if req.Side == "SELL" {
			value -= *req.Target
		} else {
			value += *req.Target
		}
		target = &value
	}

	key := positionKey(req.Strike, req.OptionType)
	pos, exists := s.positions[key]

	// ===== SHORT (SELL) =====
	if req.Side == "SELL" {
		margin := price * float64(qty) * MarginFactor

		if !s.portfolio.CanTakeMargin(margin) {
			return errors.New("insufficient margin")
		}

		s.portfolio.BlockMargin(margin)

		if !exists {
			s.positions[key] = &Position{
				Symbol:     req.Symbol,
				Strike:     req.Strike,
				OptionType: req.OptionType,
				Qty:        qty,
				AvgPrice:   price,
				Side:       SHORT,
				SL:         sl,
				Target:     target,
				EntryTime:  time.Now().Unix(),
				LastUpdate: time.Now().Unix(),
			}
		} else {
			s.mergePosition(pos, qty, price, SHORT, sl, target)
		}

		return nil
	}

	// ===== BUY =====
	cost := price * float64(qty)

	if s.portfolio.AvailableCapital < cost {
		return errors.New("insufficient capital")
	}

	s.portfolio.AvailableCapital -= cost

	if !exists {
		s.positions[key] = &Position{
			Symbol:     req.Symbol,
			Strike:     req.Strike,
			OptionType: req.OptionType,
			Qty:        qty,
			AvgPrice:   price,
			Side:       LONG,
			SL:         sl,
			Target:     target,
			EntryTime:  time.Now().Unix(),
			LastUpdate: time.Now().Unix(),
		}
	} else {
		s.mergePosition(pos, qty, price, LONG, sl, target)
	}

	return nil
}

func (s *Simulator) mergePosition(
	pos *Position,
	qty int,
	price float64,
	side PositionSide,
	sl *float64,
	target *float64,
) {

	// SAME SIDE → average
	if pos.Side == side {
		totalQty := pos.Qty + qty
		pos.AvgPrice = ((float64(pos.Qty)*pos.AvgPrice + float64(qty)*price) / float64(totalQty))
		pos.Qty = totalQty
		pos.LastUpdate = time.Now().Unix()
		if sl != nil {
			pos.SL = sl
		}
		if target != nil {
			pos.Target = target
		}
		return
	}

	// OPPOSITE SIDE → reduce or flip
	if qty < pos.Qty {
		pos.Qty -= qty
		return
	}

	if qty == pos.Qty {
		delete(s.positions, positionKey(pos.Strike, pos.OptionType))
		return
	}

	// flip
	newQty := qty - pos.Qty
	pos.Qty = newQty
	pos.AvgPrice = price
	pos.Side = side
	pos.SL = sl
	pos.Target = target
	pos.LastUpdate = time.Now().Unix()
}
