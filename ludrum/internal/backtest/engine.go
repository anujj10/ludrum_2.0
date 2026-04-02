package backtest

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const lotSize = 65

type Row struct {
	Time        time.Time `json:"time"`
	Symbol      string    `json:"symbol"`
	Strike      float64   `json:"strike"`
	OptionType  string    `json:"option_type"`
	LTP         float64   `json:"ltp"`
	Bid         float64   `json:"bid"`
	Ask         float64   `json:"ask"`
	OI          int64     `json:"oi"`
	OIChange    int64     `json:"oi_change"`
	OIChangePct float64   `json:"oi_change_pct"`
	Volume      int64     `json:"volume"`
}

type Tick struct {
	Time   time.Time
	Symbol string
	Strike float64
	CE     *Row
	PE     *Row
}

type Condition struct {
	Field string  `json:"field"`
	Op    string  `json:"op"`
	Value float64 `json:"value"`
}

type RuleSet struct {
	Match      string      `json:"match"`
	Conditions []Condition `json:"conditions"`
}

type Request struct {
	Symbol         string   `json:"symbol"`
	Date           string   `json:"date"`
	OptionType     string   `json:"option_type"`
	Strike         *float64 `json:"strike,omitempty"`
	StartTime      string   `json:"start_time"`
	EndTime        string   `json:"end_time"`
	Side           string   `json:"side"`
	Lots           int      `json:"lots"`
	Entry          RuleSet  `json:"entry"`
	Exit           RuleSet  `json:"exit"`
	StopLossPoints float64  `json:"stop_loss_points"`
	TargetPoints   float64  `json:"target_points"`
	MaxHoldMinutes int      `json:"max_hold_minutes"`
	ExitOnRedCount int      `json:"exit_on_red_count"`
}

type Trade struct {
	EntryTime  time.Time `json:"entry_time"`
	ExitTime   time.Time `json:"exit_time"`
	Strike     float64   `json:"strike"`
	OptionType string    `json:"option_type"`
	Side       string    `json:"side"`
	Qty        int       `json:"qty"`
	EntryPrice float64   `json:"entry_price"`
	ExitPrice  float64   `json:"exit_price"`
	PnL        float64   `json:"pnl"`
	Points     float64   `json:"points"`
	ExitReason string    `json:"exit_reason"`
}

type Summary struct {
	TotalTrades int     `json:"total_trades"`
	Wins        int     `json:"wins"`
	Losses      int     `json:"losses"`
	WinRate     float64 `json:"win_rate"`
	TotalPnL    float64 `json:"total_pnl"`
	AvgPnL      float64 `json:"avg_pnl"`
	AvgWin      float64 `json:"avg_win"`
	AvgLoss     float64 `json:"avg_loss"`
	Probability float64 `json:"probability"`
}

type Result struct {
	Request Request `json:"request"`
	Summary Summary `json:"summary"`
	Trades  []Trade `json:"trades"`
}

type Engine struct {
	db *pgxpool.Pool
}

func NewEngine(db *pgxpool.Pool) *Engine {
	return &Engine{db: db}
}

func (e *Engine) Run(ctx context.Context, req Request) (Result, error) {
	if req.Symbol == "" {
		req.Symbol = "NIFTY"
	}
	if req.OptionType == "" {
		req.OptionType = "CE"
	}
	if req.Lots <= 0 {
		req.Lots = 1
	}
	if req.StartTime == "" {
		req.StartTime = "09:15"
	}
	if req.EndTime == "" {
		req.EndTime = "15:30"
	}
	if req.Side == "" {
		req.Side = "BUY"
	}
	rows, err := e.loadRows(ctx, req)
	if err != nil {
		return Result{}, err
	}

	trades := simulate(rows, req, buildTicks(rows))
	summary := buildSummary(trades)

	return Result{
		Request: req,
		Summary: summary,
		Trades:  trades,
	}, nil
}

func (e *Engine) loadRows(ctx context.Context, req Request) ([]Row, error) {
	start, end, err := requestWindow(req)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT time, symbol, strike, option_type, ltp, bid, ask, oi, volume
		FROM option_chain
		WHERE symbol = $1
		  AND time BETWEEN $2 AND $3
		  AND ($4::double precision IS NULL OR strike = $4)
		  AND ($5::text = '' OR option_type = $5)
		ORDER BY time ASC
	`

	optionTypeFilter := strings.ToUpper(req.OptionType)
	if usesCrossLegFields(req) {
		optionTypeFilter = ""
	}

	rows, err := e.db.Query(ctx, query, req.Symbol, start, end, req.Strike, optionTypeFilter)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]Row, 0, 4096)
	for rows.Next() {
		var row Row
		if err := rows.Scan(
			&row.Time,
			&row.Symbol,
			&row.Strike,
			&row.OptionType,
			&row.LTP,
			&row.Bid,
			&row.Ask,
			&row.OI,
			&row.Volume,
		); err != nil {
			return nil, err
		}
		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := e.applyOIEvents(ctx, result, req, start, end); err != nil {
		return nil, err
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Time.Equal(result[j].Time) {
			if result[i].Strike == result[j].Strike {
				return result[i].OptionType < result[j].OptionType
			}
			return result[i].Strike < result[j].Strike
		}
		return result[i].Time.Before(result[j].Time)
	})

	return result, nil
}

func (e *Engine) applyOIEvents(ctx context.Context, rows []Row, req Request, start time.Time, end time.Time) error {
	if len(rows) == 0 {
		return nil
	}

	query := `
		SELECT time, strike, option_type, oi_change
		FROM option_oi_change_events
		WHERE symbol = $1
		  AND time BETWEEN $2 AND $3
		  AND ($4::double precision IS NULL OR strike = $4)
		  AND ($5::text = '' OR option_type = $5)
	`

	optionTypeFilter := strings.ToUpper(req.OptionType)
	if usesCrossLegFields(req) {
		optionTypeFilter = ""
	}

	eventRows, err := e.db.Query(ctx, query, req.Symbol, start, end, req.Strike, optionTypeFilter)
	if err != nil {
		return err
	}
	defer eventRows.Close()

	eventMap := make(map[string]int64, len(rows))
	for eventRows.Next() {
		var ts time.Time
		var strike float64
		var optionType string
		var oiChange int64
		if err := eventRows.Scan(&ts, &strike, &optionType, &oiChange); err != nil {
			return err
		}
		eventMap[eventKey(ts, strike, optionType)] = oiChange
	}

	if err := eventRows.Err(); err != nil {
		return err
	}

	for index := range rows {
		rows[index].OIChange = eventMap[eventKey(rows[index].Time, rows[index].Strike, rows[index].OptionType)]
		rows[index].OIChangePct = 0
		rows[index].Bid = 0
		rows[index].Ask = 0
		rows[index].Volume = 0
	}

	return nil
}

func requestWindow(req Request) (time.Time, time.Time, error) {
	day, err := time.ParseInLocation("2006-01-02", req.Date, time.Local)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid date: %w", err)
	}

	startClock, err := time.ParseInLocation("15:04", req.StartTime, time.Local)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid start_time: %w", err)
	}

	endClock, err := time.ParseInLocation("15:04", req.EndTime, time.Local)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid end_time: %w", err)
	}

	start := time.Date(day.Year(), day.Month(), day.Day(), startClock.Hour(), startClock.Minute(), 0, 0, time.Local)
	end := time.Date(day.Year(), day.Month(), day.Day(), endClock.Hour(), endClock.Minute(), 59, 0, time.Local)
	return start, end, nil
}

func simulate(rows []Row, req Request, ticks []Tick) []Trade {
	if usesCrossLegFields(req) {
		return simulateTicks(ticks, req)
	}

	trades := make([]Trade, 0)
	var open *Trade
	var negativeLTPStreak int
	var previousLTP *float64

	for _, row := range rows {
		if open == nil {
			if matchesRule(req.Entry, fieldResolverFromRow(row)) {
				open = &Trade{
					EntryTime:  row.Time,
					Strike:     row.Strike,
					OptionType: row.OptionType,
					Side:       strings.ToUpper(req.Side),
					Qty:        req.Lots * lotSize,
					EntryPrice: row.LTP,
				}
				negativeLTPStreak = 0
				currentLTP := row.LTP
				previousLTP = &currentLTP
			}
			continue
		}

		if row.Strike != open.Strike || row.OptionType != open.OptionType {
			continue
		}

		exitReason := ""
		points := pointsMoved(open.Side, open.EntryPrice, row.LTP)
		if req.ExitOnRedCount > 0 && previousLTP != nil {
			if row.LTP-*previousLTP < 0 {
				negativeLTPStreak++
			} else {
				negativeLTPStreak = 0
			}
		}
		currentLTP := row.LTP
		previousLTP = &currentLTP

		if req.TargetPoints > 0 && points >= req.TargetPoints {
			exitReason = "TARGET"
		}
		if exitReason == "" && req.StopLossPoints > 0 && points <= -req.StopLossPoints {
			exitReason = "STOP_LOSS"
		}
		if exitReason == "" && req.MaxHoldMinutes > 0 && row.Time.Sub(open.EntryTime) >= time.Duration(req.MaxHoldMinutes)*time.Minute {
			exitReason = "TIME_EXIT"
		}
		if exitReason == "" && matchesRule(req.Exit, fieldResolverFromRow(row)) {
			exitReason = "RULE_EXIT"
		}
		if exitReason == "" && req.ExitOnRedCount > 0 && negativeLTPStreak >= req.ExitOnRedCount {
			exitReason = fmt.Sprintf("%d_RED_LTP", req.ExitOnRedCount)
		}

		if exitReason == "" {
			continue
		}

		open.ExitTime = row.Time
		open.ExitPrice = row.LTP
		open.Points = points
		open.PnL = points * float64(open.Qty)
		open.ExitReason = exitReason
		trades = append(trades, *open)
		open = nil
		negativeLTPStreak = 0
		previousLTP = nil
	}

	if open != nil && len(rows) > 0 {
		last := rows[len(rows)-1]
		if last.Strike == open.Strike && last.OptionType == open.OptionType {
			open.ExitTime = last.Time
			open.ExitPrice = last.LTP
			open.Points = pointsMoved(open.Side, open.EntryPrice, last.LTP)
			open.PnL = open.Points * float64(open.Qty)
			open.ExitReason = "SESSION_CLOSE"
			trades = append(trades, *open)
		}
	}

	return trades
}

func simulateTicks(ticks []Tick, req Request) []Trade {
	trades := make([]Trade, 0)
	var open *Trade
	tradedType := strings.ToUpper(req.OptionType)
	var negativeLTPStreak int
	var previousLTP *float64

	for _, tick := range ticks {
		tradeRow := tick.rowFor(tradedType)
		if tradeRow == nil {
			continue
		}

		if open == nil {
			if matchesRule(req.Entry, fieldResolverFromTick(tick, tradedType)) {
				open = &Trade{
					EntryTime:  tick.Time,
					Strike:     tick.Strike,
					OptionType: tradedType,
					Side:       strings.ToUpper(req.Side),
					Qty:        req.Lots * lotSize,
					EntryPrice: tradeRow.LTP,
				}
				negativeLTPStreak = 0
				currentLTP := tradeRow.LTP
				previousLTP = &currentLTP
			}
			continue
		}

		exitReason := ""
		points := pointsMoved(open.Side, open.EntryPrice, tradeRow.LTP)
		if req.ExitOnRedCount > 0 && previousLTP != nil {
			if tradeRow.LTP-*previousLTP < 0 {
				negativeLTPStreak++
			} else {
				negativeLTPStreak = 0
			}
		}
		currentLTP := tradeRow.LTP
		previousLTP = &currentLTP

		if req.TargetPoints > 0 && points >= req.TargetPoints {
			exitReason = "TARGET"
		}
		if exitReason == "" && req.StopLossPoints > 0 && points <= -req.StopLossPoints {
			exitReason = "STOP_LOSS"
		}
		if exitReason == "" && req.MaxHoldMinutes > 0 && tick.Time.Sub(open.EntryTime) >= time.Duration(req.MaxHoldMinutes)*time.Minute {
			exitReason = "TIME_EXIT"
		}
		if exitReason == "" && matchesRule(req.Exit, fieldResolverFromTick(tick, tradedType)) {
			exitReason = "RULE_EXIT"
		}
		if exitReason == "" && req.ExitOnRedCount > 0 && negativeLTPStreak >= req.ExitOnRedCount {
			exitReason = fmt.Sprintf("%d_RED_LTP", req.ExitOnRedCount)
		}

		if exitReason == "" {
			continue
		}

		open.ExitTime = tick.Time
		open.ExitPrice = tradeRow.LTP
		open.Points = points
		open.PnL = points * float64(open.Qty)
		open.ExitReason = exitReason
		trades = append(trades, *open)
		open = nil
		negativeLTPStreak = 0
		previousLTP = nil
	}

	if open != nil && len(ticks) > 0 {
		last := ticks[len(ticks)-1].rowFor(tradedType)
		if last != nil {
			open.ExitTime = ticks[len(ticks)-1].Time
			open.ExitPrice = last.LTP
			open.Points = pointsMoved(open.Side, open.EntryPrice, last.LTP)
			open.PnL = open.Points * float64(open.Qty)
			open.ExitReason = "SESSION_CLOSE"
			trades = append(trades, *open)
		}
	}

	return trades
}

func pointsMoved(side string, entry, current float64) float64 {
	if strings.EqualFold(side, "SELL") {
		return entry - current
	}
	return current - entry
}

func matchesRule(rule RuleSet, resolve func(string) float64) bool {
	if len(rule.Conditions) == 0 {
		return false
	}

	matchAll := !strings.EqualFold(rule.Match, "any")
	if matchAll {
		for _, condition := range rule.Conditions {
			if !matchesCondition(condition, resolve) {
				return false
			}
		}
		return true
	}

	for _, condition := range rule.Conditions {
		if matchesCondition(condition, resolve) {
			return true
		}
	}
	return false
}

func matchesCondition(condition Condition, resolve func(string) float64) bool {
	value := resolve(condition.Field)
	switch condition.Op {
	case ">":
		return value > condition.Value
	case ">=":
		return value >= condition.Value
	case "<":
		return value < condition.Value
	case "<=":
		return value <= condition.Value
	case "==":
		return value == condition.Value
	case "!=":
		return value != condition.Value
	default:
		return false
	}
}

func fieldValue(field string, row Row) float64 {
	switch strings.ToLower(strings.TrimSpace(field)) {
	case "ltp":
		return row.LTP
	case "oi":
		return float64(row.OI)
	case "oi_change":
		return float64(row.OIChange)
	case "strike":
		return row.Strike
	default:
		return 0
	}
}

func eventKey(ts time.Time, strike float64, optionType string) string {
	return fmt.Sprintf("%d|%.0f|%s", ts.UnixNano(), strike, strings.ToUpper(strings.TrimSpace(optionType)))
}

func fieldResolverFromRow(row Row) func(string) float64 {
	return func(field string) float64 {
		return fieldValue(field, row)
	}
}

func fieldResolverFromTick(tick Tick, tradedOptionType string) func(string) float64 {
	return func(field string) float64 {
		switch strings.ToLower(strings.TrimSpace(field)) {
		case "ce_ltp":
			if tick.CE != nil {
				return tick.CE.LTP
			}
		case "pe_ltp":
			if tick.PE != nil {
				return tick.PE.LTP
			}
		case "ce_oi":
			if tick.CE != nil {
				return float64(tick.CE.OI)
			}
		case "pe_oi":
			if tick.PE != nil {
				return float64(tick.PE.OI)
			}
		case "ce_oi_change":
			if tick.CE != nil {
				return float64(tick.CE.OIChange)
			}
		case "pe_oi_change":
			if tick.PE != nil {
				return float64(tick.PE.OIChange)
			}
		case "ce_volume":
			if tick.CE != nil {
				return float64(tick.CE.Volume)
			}
		case "pe_volume":
			if tick.PE != nil {
				return float64(tick.PE.Volume)
			}
		}

		row := tick.rowFor(tradedOptionType)
		if row == nil {
			return 0
		}
		return fieldValue(field, *row)
	}
}

func buildTicks(rows []Row) []Tick {
	grouped := make(map[string]*Tick, len(rows))
	keys := make([]string, 0, len(rows))

	for _, row := range rows {
		key := fmt.Sprintf("%d|%.0f", row.Time.UnixNano(), row.Strike)
		tick, exists := grouped[key]
		if !exists {
			tick = &Tick{
				Time:   row.Time,
				Symbol: row.Symbol,
				Strike: row.Strike,
			}
			grouped[key] = tick
			keys = append(keys, key)
		}

		copyRow := row
		if strings.EqualFold(row.OptionType, "CE") {
			tick.CE = &copyRow
		} else if strings.EqualFold(row.OptionType, "PE") {
			tick.PE = &copyRow
		}
	}

	sort.Slice(keys, func(i, j int) bool {
		left := grouped[keys[i]]
		right := grouped[keys[j]]
		if left.Time.Equal(right.Time) {
			return left.Strike < right.Strike
		}
		return left.Time.Before(right.Time)
	})

	ticks := make([]Tick, 0, len(keys))
	for _, key := range keys {
		ticks = append(ticks, *grouped[key])
	}
	return ticks
}

func (t Tick) rowFor(optionType string) *Row {
	if strings.EqualFold(optionType, "PE") {
		return t.PE
	}
	return t.CE
}

func usesCrossLegFields(req Request) bool {
	return rulesUseCrossLegFields(req.Entry) || rulesUseCrossLegFields(req.Exit)
}

func rulesUseCrossLegFields(rule RuleSet) bool {
	for _, condition := range rule.Conditions {
		switch strings.ToLower(strings.TrimSpace(condition.Field)) {
		case "ce_ltp", "pe_ltp", "ce_oi", "pe_oi", "ce_oi_change", "pe_oi_change":
			return true
		}
	}
	return false
}

func buildSummary(trades []Trade) Summary {
	if len(trades) == 0 {
		return Summary{}
	}

	var wins int
	var losses int
	var total float64
	var totalWin float64
	var totalLoss float64

	for _, trade := range trades {
		total += trade.PnL
		if trade.PnL > 0 {
			wins++
			totalWin += trade.PnL
		} else if trade.PnL < 0 {
			losses++
			totalLoss += trade.PnL
		}
	}

	summary := Summary{
		TotalTrades: len(trades),
		Wins:        wins,
		Losses:      losses,
		WinRate:     float64(wins) * 100 / float64(len(trades)),
		Probability: float64(wins) * 100 / float64(len(trades)),
		TotalPnL:    total,
		AvgPnL:      total / float64(len(trades)),
	}

	if wins > 0 {
		summary.AvgWin = totalWin / float64(wins)
	}
	if losses > 0 {
		summary.AvgLoss = totalLoss / float64(losses)
	}

	return summary
}
