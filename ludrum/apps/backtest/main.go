package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"ludrum/internal/backtest"
	"ludrum/internal/storage/postgres"
)

func main() {
	postgres.InitDB()
	engine := backtest.NewEngine(postgres.DB)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(backtestPage))
	})

	mux.HandleFunc("/backtest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}

		var req backtest.Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		result, err := engine.Run(ctx, req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, result)
	})

	mux.HandleFunc("/backtest/dates", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}

		rows, err := postgres.DB.Query(
			r.Context(),
			`SELECT DISTINCT DATE(time) AS trading_day FROM option_chain ORDER BY trading_day DESC`,
		)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		defer rows.Close()

		dates := make([]string, 0)
		for rows.Next() {
			var day time.Time
			if err := rows.Scan(&day); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			dates = append(dates, day.Format("2006-01-02"))
		}

		writeJSON(w, http.StatusOK, map[string]any{"dates": dates})
	})

	log.Println("Backtest API running on http://localhost:8092")
	log.Fatal(http.ListenAndServe(":8092", mux))
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

var backtestPage = strings.TrimSpace(`
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Ludrum Replay Lab</title>
  <style>
    :root {
      --bg: #090909;
      --panel: #121212;
      --panel-2: #171717;
      --border: rgba(255, 255, 255, 0.08);
      --text: #f3f3f3;
      --muted: #a1a1aa;
      --up: #3dd68c;
      --down: #ff6b6b;
      --accent: #f59e0b;
      --shadow: 0 24px 64px rgba(0, 0, 0, 0.42);
    }

    * { box-sizing: border-box; }

    body {
      margin: 0;
      font-family: "Segoe UI", system-ui, sans-serif;
      background:
        radial-gradient(circle at top, rgba(245, 158, 11, 0.08), transparent 24%),
        linear-gradient(180deg, #090909, #050505 72%);
      color: var(--text);
      min-height: 100vh;
    }

    .shell {
      width: min(1500px, calc(100vw - 32px));
      margin: 0 auto;
      padding: 24px 0 40px;
    }

    .hero,
    .panel {
      border: 1px solid var(--border);
      background: linear-gradient(180deg, rgba(20, 20, 20, 0.98), rgba(10, 10, 10, 0.96));
      box-shadow: var(--shadow);
      border-radius: 24px;
    }

    .hero {
      display: flex;
      justify-content: space-between;
      gap: 20px;
      padding: 24px;
      margin-bottom: 18px;
      flex-wrap: wrap;
    }

    .eyebrow {
      margin: 0 0 8px;
      color: var(--muted);
      text-transform: uppercase;
      letter-spacing: 0.18em;
      font-size: 12px;
    }

    h1, h2, h3, p { margin: 0; }

    h1 {
      font-size: clamp(32px, 5vw, 54px);
      letter-spacing: -0.05em;
    }

    .hero p {
      max-width: 720px;
      margin-top: 10px;
      color: var(--muted);
      line-height: 1.6;
    }

    .hero-stats {
      display: grid;
      grid-template-columns: repeat(2, minmax(160px, 1fr));
      gap: 12px;
      min-width: min(380px, 100%);
    }

    .stat {
      padding: 16px;
      border-radius: 18px;
      border: 1px solid var(--border);
      background: rgba(255, 255, 255, 0.025);
    }

    .stat span {
      display: block;
      font-size: 12px;
      letter-spacing: 0.12em;
      text-transform: uppercase;
      color: var(--muted);
      margin-bottom: 8px;
    }

    .stat strong {
      font-size: 20px;
    }

    .grid {
      display: grid;
      grid-template-columns: 440px minmax(0, 1fr);
      gap: 18px;
      align-items: start;
    }

    .panel {
      padding: 20px;
    }

    .panel-head {
      display: flex;
      justify-content: space-between;
      align-items: end;
      gap: 12px;
      margin-bottom: 18px;
    }

    .panel-head span {
      color: var(--muted);
      font-size: 13px;
    }

    .form-grid {
      display: grid;
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: 12px;
    }

    .field,
    .rule-card {
      display: grid;
      gap: 8px;
    }

    .highlight-field {
      padding: 14px;
      border-radius: 18px;
      border: 1px solid rgba(245, 158, 11, 0.22);
      background: rgba(245, 158, 11, 0.06);
    }

    .field span,
    .section-label,
    .rule-head span {
      font-size: 12px;
      text-transform: uppercase;
      letter-spacing: 0.12em;
      color: var(--muted);
    }

    input,
    select,
    button {
      font: inherit;
    }

    input,
    select {
      width: 100%;
      padding: 12px 14px;
      border-radius: 14px;
      border: 1px solid rgba(255, 255, 255, 0.08);
      background: rgba(255, 255, 255, 0.03);
      color: var(--text);
      outline: none;
    }

    input:focus,
    select:focus {
      border-color: rgba(245, 158, 11, 0.5);
      box-shadow: 0 0 0 3px rgba(245, 158, 11, 0.12);
    }

    .full {
      grid-column: 1 / -1;
    }

    .stack {
      display: grid;
      gap: 18px;
      margin-top: 20px;
    }

    .rule-block {
      padding: 16px;
      border-radius: 18px;
      border: 1px solid rgba(255, 255, 255, 0.06);
      background: rgba(255, 255, 255, 0.02);
    }

    .rule-head {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 14px;
      gap: 10px;
      flex-wrap: wrap;
    }

    .rule-list {
      display: grid;
      gap: 10px;
    }

    .rule-row {
      display: grid;
      grid-template-columns: 1.15fr 0.8fr 1fr auto;
      gap: 8px;
      align-items: center;
    }

    .btn-row {
      display: flex;
      gap: 10px;
      flex-wrap: wrap;
      margin-top: 18px;
    }

    button {
      border: 0;
      border-radius: 14px;
      padding: 12px 16px;
      cursor: pointer;
      color: white;
      background: linear-gradient(180deg, #2b2b2b, #1b1b1b);
      box-shadow: 0 12px 24px rgba(0, 0, 0, 0.28);
    }

    button.secondary {
      background: rgba(255, 255, 255, 0.035);
      border: 1px solid rgba(255, 255, 255, 0.08);
      box-shadow: none;
      color: var(--text);
    }

    button.ghost {
      background: transparent;
      border: 1px solid rgba(255, 255, 255, 0.08);
      box-shadow: none;
      color: var(--muted);
      padding-inline: 12px;
    }

    button.run {
      min-width: 180px;
      font-weight: 600;
      background: linear-gradient(180deg, #f59e0b, #d97706);
      color: #111111;
    }

    .status {
      margin-top: 14px;
      min-height: 22px;
      color: var(--muted);
    }

    .status.error {
      color: var(--down);
    }

    .summary-grid {
      display: grid;
      grid-template-columns: repeat(4, minmax(0, 1fr));
      gap: 12px;
      margin-bottom: 18px;
    }

    .summary-card {
      padding: 16px;
      border-radius: 18px;
      border: 1px solid rgba(255, 255, 255, 0.06);
      background: rgba(255, 255, 255, 0.025);
    }

    .summary-card span {
      display: block;
      color: var(--muted);
      font-size: 12px;
      text-transform: uppercase;
      letter-spacing: 0.12em;
      margin-bottom: 8px;
    }

    .summary-card strong {
      font-size: 22px;
    }

    .table-wrap {
      overflow: auto;
      border-radius: 18px;
      border: 1px solid rgba(255, 255, 255, 0.06);
    }

    table {
      width: 100%;
      border-collapse: collapse;
      min-width: 980px;
    }

    th,
    td {
      padding: 13px 12px;
      border-bottom: 1px solid rgba(128, 160, 214, 0.1);
      text-align: left;
    }

    th {
      background: rgba(12, 12, 12, 0.96);
      font-size: 12px;
      text-transform: uppercase;
      letter-spacing: 0.12em;
      color: var(--muted);
      position: sticky;
      top: 0;
    }

    .up { color: var(--up); }
    .down { color: var(--down); }

    .empty {
      padding: 28px 18px;
      text-align: center;
      color: var(--muted);
      border: 1px dashed rgba(255, 255, 255, 0.08);
      border-radius: 18px;
    }

    @media (max-width: 1120px) {
      .grid {
        grid-template-columns: 1fr;
      }
    }

    @media (max-width: 760px) {
      .form-grid,
      .summary-grid,
      .hero-stats,
      .rule-row {
        grid-template-columns: 1fr;
      }

      .shell {
        width: min(100vw - 20px, 1500px);
      }
    }
  </style>
</head>
<body>
  <div class="shell">
    <section class="hero">
      <div>
        <p class="eyebrow">Ludrum Replay Lab</p>
        <h1>Replay trades with custom entry and exit rules</h1>
        <p>Pick a day, define your strike and side, set SL or target in points, then add rule rows for entry and exit. Price and timing come from <code>option_chain</code>, while OI change is taken from <code>option_oi_change_events</code> so it matches your stored event logic.</p>
      </div>
      <div class="hero-stats">
        <div class="stat"><span>Dataset</span><strong>option_chain</strong></div>
        <div class="stat"><span>Lot Size</span><strong>65 qty</strong></div>
        <div class="stat"><span>Mode</span><strong>Replay Backtest</strong></div>
        <div class="stat"><span>Endpoint</span><strong>/backtest</strong></div>
      </div>
    </section>

    <section class="grid">
      <article class="panel">
        <div class="panel-head">
          <div>
            <h2>Strategy Builder</h2>
            <span>Entry and exit like a replay ticket</span>
          </div>
        </div>

        <div class="form-grid">
          <label class="field">
            <span>Date</span>
            <select id="date"></select>
          </label>
          <label class="field">
            <span>Symbol</span>
            <input id="symbol" value="NIFTY" />
          </label>
          <label class="field">
            <span>Option Type</span>
            <select id="optionType">
              <option value="CE">CE</option>
              <option value="PE">PE</option>
            </select>
          </label>
          <label class="field">
            <span>Strike</span>
            <input id="strike" placeholder="Leave blank to scan all strikes. Fill it only to lock one strike." />
          </label>
          <label class="field">
            <span>Side</span>
            <select id="side">
              <option value="BUY">BUY</option>
              <option value="SELL">SELL</option>
            </select>
          </label>
          <label class="field">
            <span>Lots</span>
            <input id="lots" type="number" min="1" value="1" />
          </label>
          <label class="field">
            <span>Start Time</span>
            <input id="startTime" type="time" value="09:15" />
          </label>
          <label class="field">
            <span>End Time</span>
            <input id="endTime" type="time" value="15:30" />
          </label>
          <label class="field">
            <span>SL (points)</span>
            <input id="sl" type="number" step="0.05" value="15" />
          </label>
          <label class="field">
            <span>Target (points)</span>
            <input id="target" type="number" step="0.05" value="20" />
          </label>
          <label class="field full">
            <span>Max Hold (minutes)</span>
            <input id="maxHold" type="number" min="0" value="20" />
          </label>
          <label class="field full highlight-field">
            <span>Consecutive down LTP exit</span>
            <select id="redExitCount">
              <option value="0">Disabled</option>
              <option value="3">Exit after 3 down ticks</option>
              <option value="4">Exit after 4 down ticks</option>
            </select>
          </label>
        </div>

        <div class="stack">
          <section class="rule-block">
            <div class="rule-head">
              <div>
                <h3>Entry Rules</h3>
                <span>All or Any conditions</span>
              </div>
              <select id="entryMatch">
                <option value="all">ALL</option>
                <option value="any">ANY</option>
              </select>
            </div>
            <div id="entryRules" class="rule-list"></div>
            <div class="btn-row">
              <button type="button" class="secondary" data-add-rule="entry">Add entry rule</button>
            </div>
          </section>

          <section class="rule-block">
            <div class="rule-head">
              <div>
                <h3>Exit Rules</h3>
                <span>Optional rule-based exit on top of SL, target, and time exit</span>
              </div>
              <select id="exitMatch">
                <option value="any">ANY</option>
                <option value="all">ALL</option>
              </select>
            </div>
            <div id="exitRules" class="rule-list"></div>
            <div class="btn-row">
              <button type="button" class="secondary" data-add-rule="exit">Add exit rule</button>
            </div>
          </section>
        </div>

        <div class="btn-row">
          <button type="button" class="run" id="runBacktest">Run replay</button>
          <button type="button" class="secondary" id="loadExample">Load example</button>
          <button type="button" class="secondary" id="loadCEOpposite">Load CE up / PE down</button>
          <button type="button" class="secondary" id="loadPEOpposite">Load PE up / CE down</button>
          <button type="button" class="secondary" id="loadPESetup">Load PE event setup</button>
        </div>
        <div id="status" class="status"></div>
      </article>

      <article class="panel">
        <div class="panel-head">
          <div>
            <h2>Replay Results</h2>
            <span>Trade log, win rate, and probability</span>
          </div>
        </div>

        <div id="summary" class="summary-grid"></div>
        <div id="resultEmpty" class="empty">Run a replay to see the trade log here.</div>
        <div id="results" style="display:none;">
          <div class="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Entry</th>
                  <th>Exit</th>
                  <th>Strike</th>
                  <th>Type</th>
                  <th>Side</th>
                  <th>Qty</th>
                  <th>Entry Price</th>
                  <th>Exit Price</th>
                  <th>Points</th>
                  <th>PnL</th>
                  <th>Reason</th>
                </tr>
              </thead>
              <tbody id="tradesBody"></tbody>
            </table>
          </div>
        </div>
      </article>
    </section>
  </div>

  <template id="ruleTemplate">
    <div class="rule-row">
      <select data-role="field">
        <option value="ltp">ltp</option>
        <option value="oi">oi</option>
        <option value="oi_change">oi_change</option>
        <option value="strike">strike</option>
        <option value="ce_oi_change">ce_oi_change</option>
        <option value="pe_oi_change">pe_oi_change</option>
        <option value="ce_oi">ce_oi</option>
        <option value="pe_oi">pe_oi</option>
        <option value="ce_ltp">ce_ltp</option>
        <option value="pe_ltp">pe_ltp</option>
      </select>
      <select data-role="op">
        <option value=">">></option>
        <option value=">=">>=</option>
        <option value="<"><</option>
        <option value="<="><=</option>
        <option value="==">==</option>
        <option value="!=">!=</option>
      </select>
      <input data-role="value" type="number" step="0.01" placeholder="Value" />
      <button type="button" class="ghost" data-role="remove">Remove</button>
    </div>
  </template>

  <script>
    const fields = {
      date: document.getElementById("date"),
      symbol: document.getElementById("symbol"),
      optionType: document.getElementById("optionType"),
      strike: document.getElementById("strike"),
      side: document.getElementById("side"),
      lots: document.getElementById("lots"),
      startTime: document.getElementById("startTime"),
      endTime: document.getElementById("endTime"),
      sl: document.getElementById("sl"),
      target: document.getElementById("target"),
      maxHold: document.getElementById("maxHold"),
      redExitCount: document.getElementById("redExitCount"),
      entryMatch: document.getElementById("entryMatch"),
      exitMatch: document.getElementById("exitMatch"),
      status: document.getElementById("status"),
      summary: document.getElementById("summary"),
      results: document.getElementById("results"),
      resultEmpty: document.getElementById("resultEmpty"),
      tradesBody: document.getElementById("tradesBody"),
      entryRules: document.getElementById("entryRules"),
      exitRules: document.getElementById("exitRules")
    }

    const ruleTemplate = document.getElementById("ruleTemplate")

    function addRule(target, rule = {}) {
      const fragment = ruleTemplate.content.cloneNode(true)
      const row = fragment.querySelector(".rule-row")
      row.querySelector('[data-role="field"]').value = rule.field || "oi_change"
      row.querySelector('[data-role="op"]').value = rule.op || ">"
      row.querySelector('[data-role="value"]').value = rule.value ?? ""
      row.querySelector('[data-role="remove"]').addEventListener("click", () => {
        row.remove()
      })
      target.appendChild(fragment)
    }

    function collectRules(target) {
      return Array.from(target.querySelectorAll(".rule-row"))
        .map((row) => ({
          field: row.querySelector('[data-role="field"]').value,
          op: row.querySelector('[data-role="op"]').value,
          value: Number(row.querySelector('[data-role="value"]').value)
        }))
        .filter((rule) => Number.isFinite(rule.value))
    }

    function formatNumber(value, digits = 2) {
      if (value === null || value === undefined || Number.isNaN(value)) return "-"
      return new Intl.NumberFormat("en-IN", {
        minimumFractionDigits: digits,
        maximumFractionDigits: digits
      }).format(value)
    }

    function tone(value) {
      if (value > 0) return "up"
      if (value < 0) return "down"
      return ""
    }

    function renderSummary(summary) {
      const cards = [
        ["Probability", formatNumber(summary.probability) + "%", tone(summary.probability >= 50 ? 1 : -1)],
        ["Total PnL", formatNumber(summary.total_pnl), tone(summary.total_pnl)],
        ["Win Rate", formatNumber(summary.win_rate) + "%", tone(summary.win_rate >= 50 ? 1 : -1)],
        ["Trades", String(summary.total_trades || 0), ""],
        ["Wins", String(summary.wins || 0), "up"],
        ["Losses", String(summary.losses || 0), summary.losses ? "down" : ""],
        ["Avg Win", formatNumber(summary.avg_win), "up"],
        ["Avg Loss", formatNumber(summary.avg_loss), "down"]
      ]

      fields.summary.innerHTML = cards.map(([label, value, cls]) =>
        '<div class="summary-card"><span>' + label + '</span><strong class="' + cls + '">' + value + '</strong></div>'
      ).join("")
    }

    function renderTrades(trades) {
      fields.tradesBody.innerHTML = trades.map((trade) => {
        const pnlClass = tone(trade.pnl)
        const pointsClass = tone(trade.points)
        return '<tr>' +
          '<td>' + new Date(trade.entry_time).toLocaleString("en-IN") + '</td>' +
          '<td>' + new Date(trade.exit_time).toLocaleString("en-IN") + '</td>' +
          '<td>' + formatNumber(trade.strike, 0) + '</td>' +
          '<td>' + trade.option_type + '</td>' +
          '<td>' + trade.side + '</td>' +
          '<td>' + trade.qty + '</td>' +
          '<td>' + formatNumber(trade.entry_price) + '</td>' +
          '<td>' + formatNumber(trade.exit_price) + '</td>' +
          '<td class="' + pointsClass + '">' + formatNumber(trade.points) + '</td>' +
          '<td class="' + pnlClass + '">' + formatNumber(trade.pnl) + '</td>' +
          '<td>' + trade.exit_reason + '</td>' +
        '</tr>'
      }).join("")
    }

    async function loadDates() {
      try {
        const response = await fetch("/backtest/dates")
        const payload = await response.json()
        const dates = payload.dates || []
        fields.date.innerHTML = dates.map((value) => '<option value="' + value + '">' + value + '</option>').join("")
      } catch (error) {
        fields.status.textContent = "Unable to load trading dates."
        fields.status.className = "status error"
      }
    }

    function loadExample() {
      fields.symbol.value = "NIFTY"
      fields.optionType.value = "CE"
      fields.side.value = "BUY"
      fields.lots.value = "1"
      fields.startTime.value = "09:15"
      fields.endTime.value = "15:30"
      fields.sl.value = "15"
      fields.target.value = "20"
      fields.maxHold.value = "20"
      fields.redExitCount.value = "0"
      fields.strike.value = ""
      fields.entryMatch.value = "all"
      fields.exitMatch.value = "any"
      fields.entryRules.innerHTML = ""
      fields.exitRules.innerHTML = ""
      addRule(fields.entryRules, { field: "oi_change", op: ">", value: 10000 })
      addRule(fields.entryRules, { field: "ltp", op: ">", value: 100 })
      addRule(fields.exitRules, { field: "oi_change", op: "<", value: -5000 })
      addRule(fields.exitRules, { field: "ltp", op: "<", value: 95 })
    }

    function loadOppositeFlowSetup(tradeType) {
      fields.symbol.value = "NIFTY"
      fields.optionType.value = tradeType
      fields.side.value = "BUY"
      fields.lots.value = "1"
      fields.startTime.value = "09:15"
      fields.endTime.value = "15:30"
      fields.sl.value = "15"
      fields.target.value = "20"
      fields.maxHold.value = "20"
      fields.redExitCount.value = "3"
      fields.strike.value = ""
      fields.entryMatch.value = "all"
      fields.exitMatch.value = "any"
      fields.entryRules.innerHTML = ""
      fields.exitRules.innerHTML = ""

      if (tradeType === "CE") {
        addRule(fields.entryRules, { field: "ce_oi_change", op: ">", value: 0 })
        addRule(fields.entryRules, { field: "pe_oi_change", op: "<", value: 0 })
      } else {
        addRule(fields.entryRules, { field: "pe_oi_change", op: ">", value: 0 })
        addRule(fields.entryRules, { field: "ce_oi_change", op: "<", value: 0 })
      }
    }

    function loadPEEventSetup() {
      fields.symbol.value = "NIFTY"
      fields.optionType.value = "PE"
      fields.side.value = "BUY"
      fields.lots.value = "1"
      fields.startTime.value = "09:15"
      fields.endTime.value = "15:30"
      fields.sl.value = "15"
      fields.target.value = "20"
      fields.maxHold.value = "20"
      fields.redExitCount.value = "3"
      fields.strike.value = ""
      fields.entryMatch.value = "all"
      fields.exitMatch.value = "any"
      fields.entryRules.innerHTML = ""
      fields.exitRules.innerHTML = ""
      addRule(fields.entryRules, { field: "pe_oi_change", op: ">", value: 0 })
      addRule(fields.entryRules, { field: "ce_oi_change", op: "<", value: 0 })
    }

    async function runBacktest() {
      fields.status.textContent = "Running replay..."
      fields.status.className = "status"

      const body = {
        symbol: fields.symbol.value.trim() || "NIFTY",
        date: fields.date.value,
        option_type: fields.optionType.value,
        start_time: fields.startTime.value,
        end_time: fields.endTime.value,
        side: fields.side.value,
        lots: Number(fields.lots.value) || 1,
        stop_loss_points: Number(fields.sl.value) || 0,
        target_points: Number(fields.target.value) || 0,
        max_hold_minutes: Number(fields.maxHold.value) || 0,
        exit_on_red_count: Number(fields.redExitCount.value) || 0,
        entry: {
          match: fields.entryMatch.value,
          conditions: collectRules(fields.entryRules)
        },
        exit: {
          match: fields.exitMatch.value,
          conditions: collectRules(fields.exitRules)
        }
      }

      const strike = fields.strike.value.trim()
      if (strike !== "") {
        body.strike = Number(strike)
      }

      try {
        const response = await fetch("/backtest", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body)
        })

        const payload = await response.json()
        if (!response.ok) {
          throw new Error(payload.error || "Backtest failed")
        }

        renderSummary(payload.summary || {})
        renderTrades(payload.trades || [])
        fields.resultEmpty.style.display = "none"
        fields.results.style.display = "block"
        fields.status.textContent = "Replay complete."
        fields.status.className = "status"
      } catch (error) {
        fields.status.textContent = error.message || "Backtest failed"
        fields.status.className = "status error"
      }
    }

    document.querySelectorAll("[data-add-rule]").forEach((button) => {
      button.addEventListener("click", () => {
        const target = button.getAttribute("data-add-rule") === "entry" ? fields.entryRules : fields.exitRules
        addRule(target)
      })
    })

    document.getElementById("runBacktest").addEventListener("click", runBacktest)
    document.getElementById("loadExample").addEventListener("click", loadExample)
    document.getElementById("loadCEOpposite").addEventListener("click", () => loadOppositeFlowSetup("CE"))
    document.getElementById("loadPEOpposite").addEventListener("click", () => loadOppositeFlowSetup("PE"))
    document.getElementById("loadPESetup").addEventListener("click", loadPEEventSetup)

    addRule(fields.entryRules, { field: "oi_change", op: ">", value: 10000 })
    addRule(fields.exitRules, { field: "oi_change", op: "<", value: -5000 })
    loadDates()
  </script>
</body>
</html>
`)
