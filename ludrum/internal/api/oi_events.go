package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"ludrum/internal/runtime"
	"ludrum/internal/storage/postgres"
)

type oiEventRow struct {
	Time       time.Time `json:"time"`
	Symbol     string    `json:"symbol"`
	Strike     float64   `json:"strike"`
	OptionType string    `json:"option_type"`
	OIChange   int64     `json:"oi_change"`
	LTPChange  float64   `json:"ltp_change"`
}

var oiEventRuntimeManager *runtime.Manager

func RegisterOIEventRoutes(mux *http.ServeMux, runtimeManager *runtime.Manager) {
	oiEventRuntimeManager = runtimeManager
	mux.HandleFunc("/oi-events", handleOIEvents)
}

func handleOIEvents(w http.ResponseWriter, r *http.Request) {
	allowCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	user, err := authorizeRequest(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	symbol := r.URL.Query().Get("symbol")
	if symbol == "" {
		symbol = "NIFTY"
	}

	rawStrikes := strings.TrimSpace(r.URL.Query().Get("strikes"))
	if rawStrikes == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "strikes query is required"})
		return
	}

	parts := strings.Split(rawStrikes, ",")
	strikes := make([]float64, 0, len(parts))
	for _, part := range parts {
		value, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid strike in query"})
			return
		}
		strikes = append(strikes, value)
	}

	limit := 12
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		if parsed, err := strconv.Atoi(rawLimit); err == nil && parsed > 0 && parsed <= 50 {
			limit = parsed
		}
	}

	if oiEventRuntimeManager != nil && user != nil {
		if userRuntime, runtimeErr := ensureFyersRuntimeForUser(r.Context(), oiEventRuntimeManager, user.ID); runtimeErr == nil {
			if events := userRuntime.GetOIEvents(symbol, strikes, limit); len(events) > 0 {
				writeJSON(w, http.StatusOK, events)
				return
			}
		}
	}

	if user != nil {
		if account, accountErr := postgres.GetFyersAccountByUserID(r.Context(), user.ID); accountErr == nil {
			if events, loadErr := postgres.LoadUserRuntimeOIEvents(r.Context(), user.ID, account.ID, symbol, strikes, limit); loadErr == nil && len(events) > 0 {
				result := make([]oiEventRow, 0, len(events))
				for _, event := range events {
					result = append(result, oiEventRow{
						Time:       event.Time,
						Symbol:     event.Symbol,
						Strike:     event.Strike,
						OptionType: event.OptionType,
						OIChange:   event.OIChange,
						LTPChange:  event.LTPChange,
					})
				}
				writeJSON(w, http.StatusOK, result)
				return
			}
		}
	}

	rows, err := postgres.DB.Query(
		r.Context(),
		`
		WITH ranked AS (
			SELECT
				time,
				symbol,
				strike,
				option_type,
				oi_change,
				ltp_change,
				ROW_NUMBER() OVER (
					PARTITION BY strike, option_type
					ORDER BY time DESC
				) AS rn
			FROM option_oi_change_events
			WHERE symbol = $1
			  AND strike = ANY($2)
		)
		SELECT time, symbol, strike, option_type, oi_change, ltp_change
		FROM ranked
		WHERE rn <= $3
		ORDER BY strike ASC, option_type ASC, time DESC
		`,
		symbol,
		strikes,
		limit,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	result := make([]oiEventRow, 0)
	for rows.Next() {
		var row oiEventRow
		if err := rows.Scan(&row.Time, &row.Symbol, &row.Strike, &row.OptionType, &row.OIChange, &row.LTPChange); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, result)
}
