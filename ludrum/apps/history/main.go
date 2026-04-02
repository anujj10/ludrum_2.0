package main

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"ludrum/internal/storage/postgres"
)

type historyRow struct {
	Time       time.Time
	Strike     float64
	OptionType string
	LTP        float64
	OI         int64
}

type pageData struct {
	Symbol     string
	Strike     string
	OptionType string
	FromDate   string
	ToDate     string
	Limit      int
	Rows       []historyRow
	Error      string
}

var pageTemplate = template.Must(template.New("history").Parse(`
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Ludrum History</title>
  <style>
    :root {
      color-scheme: dark;
      font-family: "Segoe UI", sans-serif;
      background:
        radial-gradient(circle at top, rgba(42, 91, 255, 0.16), transparent 28%),
        linear-gradient(180deg, #08101c 0%, #050912 100%);
      color: #d8e2f0;
    }
    * { box-sizing: border-box; }
    body { margin: 0; min-height: 100vh; }
    .shell { width: min(1200px, calc(100vw - 32px)); margin: 0 auto; padding: 24px 0 40px; }
    .panel {
      border: 1px solid rgba(123, 160, 255, 0.18);
      background: linear-gradient(180deg, rgba(12, 21, 37, 0.95), rgba(7, 13, 24, 0.92));
      box-shadow: 0 20px 60px rgba(0, 0, 0, 0.32), inset 0 1px 0 rgba(203, 219, 255, 0.06);
      border-radius: 22px;
      padding: 22px;
      margin-bottom: 18px;
    }
    h1, h2, p { margin: 0; }
    h1 { font-size: clamp(2rem, 4vw, 3rem); margin-bottom: 8px; }
    .sub { color: #8aa0bf; margin-bottom: 18px; }
    form {
      display: grid;
      grid-template-columns: repeat(6, minmax(0, 1fr)) auto;
      gap: 12px;
      align-items: end;
    }
    label {
      display: grid;
      gap: 8px;
      color: #8aa0bf;
      font-size: 0.82rem;
      text-transform: uppercase;
      letter-spacing: 0.1em;
    }
    input, select, button {
      border-radius: 14px;
      border: 1px solid rgba(128, 160, 214, 0.16);
      background: rgba(12, 19, 32, 0.72);
      color: #eff6ff;
      min-height: 46px;
      padding: 0 14px;
      font: inherit;
    }
    button {
      cursor: pointer;
      background: linear-gradient(180deg, rgba(41, 89, 182, 0.95), rgba(29, 64, 130, 0.95));
      font-weight: 600;
    }
    .stats {
      display: grid;
      grid-template-columns: repeat(3, minmax(0, 1fr));
      gap: 12px;
      margin-bottom: 18px;
    }
    .stat {
      border: 1px solid rgba(128, 160, 214, 0.16);
      background: rgba(12, 19, 32, 0.72);
      border-radius: 16px;
      padding: 16px;
    }
    .stat span { color: #8aa0bf; display: block; margin-bottom: 8px; font-size: 0.85rem; }
    .stat strong { font-size: 1.25rem; }
    table { width: 100%; border-collapse: collapse; }
    th, td {
      padding: 14px 12px;
      border-bottom: 1px solid rgba(110, 135, 172, 0.12);
      text-align: left;
    }
    thead th {
      background: rgba(10, 17, 29, 0.96);
      color: #7f95b5;
      text-transform: uppercase;
      letter-spacing: 0.14em;
      font-size: 0.76rem;
      position: sticky;
      top: 0;
    }
    .table-wrap {
      overflow: auto;
      border: 1px solid rgba(120, 145, 184, 0.14);
      border-radius: 18px;
    }
    .empty, .error { color: #8ea1bd; padding: 24px 8px; text-align: center; }
    .error { color: #ff9b9b; }
    @media (max-width: 900px) {
      form, .stats { grid-template-columns: 1fr; }
    }
  </style>
</head>
<body>
  <div class="shell">
    <section class="panel">
      <h1>Option Chain History</h1>
      <p class="sub">Browse option_chain rows from PostgreSQL with only time, LTP, and OI.</p>
      <form method="get" action="/">
        <label>
          Symbol
          <input type="text" name="symbol" value="{{.Symbol}}" placeholder="NIFTY">
        </label>
        <label>
          Strike
          <input type="text" name="strike" value="{{.Strike}}" placeholder="22900">
        </label>
        <label>
          Option Type
          <select name="option_type">
            <option value="" {{if eq .OptionType ""}}selected{{end}}>All</option>
            <option value="CE" {{if eq .OptionType "CE"}}selected{{end}}>CE</option>
            <option value="PE" {{if eq .OptionType "PE"}}selected{{end}}>PE</option>
          </select>
        </label>
        <label>
          From Date
          <input type="date" name="from_date" value="{{.FromDate}}">
        </label>
        <label>
          To Date
          <input type="date" name="to_date" value="{{.ToDate}}">
        </label>
        <label>
          Limit
          <input type="number" min="10" max="1000" name="limit" value="{{.Limit}}">
        </label>
        <button type="submit">Load</button>
      </form>
    </section>

    <section class="panel">
      <div class="stats">
        <div class="stat"><span>Rows</span><strong>{{len .Rows}}</strong></div>
        <div class="stat"><span>Symbol</span><strong>{{if .Symbol}}{{.Symbol}}{{else}}All{{end}}</strong></div>
        <div class="stat"><span>Date Range</span><strong>{{if .FromDate}}{{.FromDate}}{{else}}Start{{end}} to {{if .ToDate}}{{.ToDate}}{{else}}Now{{end}}</strong></div>
      </div>
      {{if .Error}}
        <div class="error">{{.Error}}</div>
      {{else if .Rows}}
        <div class="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Time</th>
                <th>Strike</th>
                <th>Type</th>
                <th>LTP</th>
                <th>OI</th>
              </tr>
            </thead>
            <tbody>
              {{range .Rows}}
              <tr>
                <td>{{.Time.Format "2006-01-02 15:04:05"}}</td>
                <td>{{printf "%.0f" .Strike}}</td>
                <td>{{.OptionType}}</td>
                <td>{{printf "%.2f" .LTP}}</td>
                <td>{{.OI}}</td>
              </tr>
              {{end}}
            </tbody>
          </table>
        </div>
      {{else}}
        <div class="empty">No rows found for this filter.</div>
      {{end}}
    </section>
  </div>
</body>
</html>
`))

func main() {
	postgres.InitDB()
	defer postgres.DB.Close()

	http.HandleFunc("/", historyPage)

	port := os.Getenv("HISTORY_PORT")
	if port == "" {
		port = "8095"
	}

	log.Printf("History UI running on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func historyPage(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	data := pageData{
		Symbol:     defaultString(query.Get("symbol"), "NIFTY"),
		Strike:     query.Get("strike"),
		OptionType: query.Get("option_type"),
		FromDate:   query.Get("from_date"),
		ToDate:     query.Get("to_date"),
		Limit:      defaultLimit(query.Get("limit")),
	}

	rows, err := fetchHistory(
		r.Context(),
		data.Symbol,
		data.Strike,
		data.OptionType,
		data.FromDate,
		data.ToDate,
		data.Limit,
	)
	if err != nil {
		data.Error = err.Error()
	} else {
		data.Rows = rows
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pageTemplate.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func fetchHistory(
	ctx context.Context,
	symbol, strikeText, optionType, fromDateText, toDateText string,
	limit int,
) ([]historyRow, error) {
	var strike *float64
	if strikeText != "" {
		value, err := strconv.ParseFloat(strikeText, 64)
		if err != nil {
			return nil, err
		}
		strike = &value
	}

	var fromDate *time.Time
	if fromDateText != "" {
		value, err := time.Parse("2006-01-02", fromDateText)
		if err != nil {
			return nil, err
		}
		fromDate = &value
	}

	var toDate *time.Time
	if toDateText != "" {
		value, err := time.Parse("2006-01-02", toDateText)
		if err != nil {
			return nil, err
		}
		value = value.Add(24*time.Hour - time.Nanosecond)
		toDate = &value
	}

	rows, err := postgres.DB.Query(ctx, `
		SELECT time, strike, option_type, ltp, oi
		FROM option_chain
		WHERE ($1 = '' OR symbol = $1)
		  AND ($2::double precision IS NULL OR strike = $2)
		  AND ($3 = '' OR option_type = $3)
		  AND ($4::timestamptz IS NULL OR time >= $4)
		  AND ($5::timestamptz IS NULL OR time <= $5)
		ORDER BY time DESC
		LIMIT $6
	`, symbol, strike, optionType, fromDate, toDate, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]historyRow, 0, limit)
	for rows.Next() {
		var row historyRow
		if err := rows.Scan(&row.Time, &row.Strike, &row.OptionType, &row.LTP, &row.OI); err != nil {
			return nil, err
		}
		result = append(result, row)
	}

	return result, rows.Err()
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func defaultLimit(raw string) int {
	if raw == "" {
		return 200
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 200
	}

	if value > 1000 {
		return 1000
	}

	return value
}
