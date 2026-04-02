import { useEffect, useState } from "react"
import { useOptionStore } from "../store/useOptionStore"
import { API_BASE_URL } from "../config"
import type { OIChangeEvent, PairSignal, Portfolio, Position, StrikeAnalytics } from "../types/option"

const STRIKE_STEP = 50

type OIChangeEntry = {
  value: number
  time: string
}

type OIHistoryMap = Record<number, { CE: OIChangeEntry[]; PE: OIChangeEntry[] }>

function formatNumber(value: number | undefined, digits = 2) {
  if (value === undefined || value === null || Number.isNaN(value)) return "-"
  return new Intl.NumberFormat("en-IN", {
    minimumFractionDigits: digits,
    maximumFractionDigits: digits,
  }).format(value)
}

function formatCompact(value: number | undefined) {
  if (value === undefined || value === null || Number.isNaN(value)) return "-"
  return new Intl.NumberFormat("en-IN", {
    notation: "compact",
    maximumFractionDigits: 2,
  }).format(value)
}

function formatTapeValue(value: number | undefined) {
  if (value === undefined || value === null || Number.isNaN(value)) return "-"

  const absolute = Math.abs(value)
  const sign = value < 0 ? "-" : ""

  if (absolute >= 100000) {
    return `${sign}${(absolute / 100000).toFixed(1)}L`
  }

  if (absolute >= 1000) {
    return `${sign}${(absolute / 1000).toFixed(1)}K`
  }

  return `${value}`
}

function formatTime(unix: number | undefined) {
  if (!unix) return "-"
  return new Date(unix * 1000).toLocaleTimeString("en-IN", {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  })
}

function toneClass(value: number | undefined) {
  if (!value) return "flat"
  return value > 0 ? "up" : "down"
}

function ltpSeriesTone(series: number[] | undefined) {
  if (!series || series.length < 2) return "flat"

  let deltaSum = 0
  for (let index = 1; index < series.length; index += 1) {
    deltaSum += series[index] - series[index - 1]
  }

  if (deltaSum > 1) return "up"
  if (deltaSum < 1) return "down"
  return "flat"
}

function sparkline(series: number[] | undefined) {
  if (!series?.length) return "[]"
  return `[${series.slice(-6).map((value) => formatNumber(value, 1)).join(" ")}]`
}

function OISeriesValues({ series, align }: { series: OIChangeEntry[] | undefined; align: "left" | "right" }) {
  if (!series?.length) {
    return <span className="oi-series-empty">No OI change yet</span>
  }

  const visible = series.slice(-12)

  return (
    <div className={`oi-values ${align}`}>
      {visible.map((value, index) => (
        <span
          key={`${index}-${value.time}-${value.value}`}
          className={`oi-value ${value.value > 0 ? "up" : "down"}`}
          title={`${value.value} at ${value.time}`}
        >
          <span>{formatTapeValue(value.value)}</span>
          <span className="oi-time">{value.time}</span>
        </span>
      ))}
    </div>
  )
}

function formatEventTime(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return "--:--"
  return date.toLocaleTimeString("en-IN", {
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  })
}

function buildOIHistoryMap(events: OIChangeEvent[]) {
  const grouped: OIHistoryMap = {}

  for (const event of events) {
    const strike = Number(event.strike)
    if (!grouped[strike]) {
      grouped[strike] = { CE: [], PE: [] }
    }

    grouped[strike][event.option_type].push({
      value: event.oi_change,
      time: formatEventTime(event.time),
    })
  }

  return grouped
}

function createEmptyLeg(strike: number, type: "CE" | "PE"): StrikeAnalytics {
  return {
    Strike: strike,
    Type: type,
    VolumeChange: 0,
    OIChange: 0,
    LTPChange: 0,
    CurrentOI: 0,
    LTPSeries: [],
    LTPDeltas: [],
    LTPPattern: [],
    Velocity: 0,
    Acceleration: 0,
    OIMomentum: 0,
    VolumeSpike: 0,
    Signal: "NEUTRAL",
    Highlight: false,
  }
}

function TradeTicket({
  strike,
  optionType,
  price,
  onFeedback,
}: {
  strike: number
  optionType: "CE" | "PE"
  price: number | undefined
  onFeedback: (message: string) => void
}) {
  const [lots, setLots] = useState("1")
  const [sl, setSl] = useState("")
  const [target, setTarget] = useState("")
  const [submitting, setSubmitting] = useState(false)

  async function submitTrade(side: "BUY" | "SELL") {
    const parsedLots = Number(lots)
    if (!Number.isFinite(parsedLots) || parsedLots <= 0) {
      onFeedback("Lots must be greater than zero")
      return
    }

    setSubmitting(true)
    try {
      const response = await fetch(`${API_BASE_URL}/trade`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          symbol: "NIFTY",
          strike,
          optionType,
          side,
          lots: parsedLots,
          SL: sl ? Number(sl) : null,
          Target: target ? Number(target) : null,
          EnableSL: Boolean(sl),
          EnableTarget: Boolean(target),
        }),
      })

      const payload = await response.json().catch(() => null)
      if (!response.ok) {
        onFeedback(payload?.error || `${side} failed`)
        return
      }

      onFeedback(`${side} ${optionType} ${formatNumber(strike, 0)} placed at ${formatNumber(price)}`)
    } catch {
      onFeedback("Trade request failed")
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="trade-ticket">
      <div className="ticket-grid">
        <label className="ticket-field">
          <span>Lots</span>
          <input value={lots} onChange={(event) => setLots(event.target.value)} inputMode="numeric" />
        </label>
        <label className="ticket-field">
          <span>SL (pts)</span>
          <input value={sl} onChange={(event) => setSl(event.target.value)} inputMode="decimal" />
        </label>
        <label className="ticket-field">
          <span>Target (pts)</span>
          <input value={target} onChange={(event) => setTarget(event.target.value)} inputMode="decimal" />
        </label>
      </div>
      <div className="ticket-actions">
        <button type="button" className="trade-btn buy" disabled={submitting} onClick={() => submitTrade("BUY")}>
          Buy
        </button>
        <button type="button" className="trade-btn sell" disabled={submitting} onClick={() => submitTrade("SELL")}>
          Sell
        </button>
      </div>
    </div>
  )
}

function OptionLegCell({
  leg,
  history,
  align,
  strike,
  optionType,
  onFeedback,
}: {
  leg: StrikeAnalytics
  history: OIChangeEntry[]
  align: "left" | "right"
  strike: number
  optionType: "CE" | "PE"
  onFeedback: (message: string) => void
}) {
  const latestLtp = leg.LTPSeries?.[leg.LTPSeries.length - 1]

  return (
    <div className={`leg-panel ${align}`}>
      <div className="leg-top">
        <div className="metric-block">
          <span className="metric-label">LTP</span>
          <span className={`metric price ${toneClass(leg.LTPChange)}`}>{formatNumber(latestLtp)}</span>
        </div>
        <div className="metric-block">
          <span className="metric-label">LTP change</span>
          <span className={`metric ${toneClass(leg.LTPChange)}`}>{formatNumber(leg.LTPChange)}</span>
        </div>
        <div className="metric-block">
          <span className="metric-label">Total OI</span>
          <span className="metric oi-total">{formatCompact(leg.CurrentOI)}</span>
        </div>
      </div>

      <div className="series-section">
        <span className="series-label">LTP Series</span>
        <div className="series-box">
          <span className={ltpSeriesTone(leg.LTPSeries)}>{sparkline(leg.LTPSeries)}</span>
        </div>
      </div>

      <div className="series-section">
        <div className="series-head">
          <span className="series-label">OI change</span>
          <span className={toneClass(leg.OIChange)}>Now {formatCompact(leg.OIChange)}</span>
        </div>
        <div className="series-box">
          <OISeriesValues series={history} align={align} />
        </div>
      </div>

      <TradeTicket strike={strike} optionType={optionType} price={latestLtp} onFeedback={onFeedback} />
    </div>
  )
}

function PositionRow({
  position,
  onExit,
}: {
  position: Position
  onExit: (position: Position) => Promise<void>
}) {
  return (
    <tr>
      <td>{position.Symbol || "NIFTY"}</td>
      <td>{position.OptionType}</td>
      <td>{formatNumber(position.Strike, 0)}</td>
      <td>{position.Side}</td>
      <td>{position.Qty}</td>
      <td>{formatNumber(position.AvgPrice)}</td>
      <td className={toneClass(position.UnrealizedPnL)}>{formatNumber(position.UnrealizedPnL)}</td>
      <td>{formatTime(position.LastUpdate || position.EntryTime)}</td>
      <td>
        <button type="button" className="exit-btn" onClick={() => void onExit(position)}>
          Exit
        </button>
      </td>
    </tr>
  )
}

function ChainRow({
  pair,
  strike,
  spot,
  history,
  onFeedback,
}: {
  pair?: PairSignal
  strike: number
  spot: number
  history?: { CE: OIChangeEntry[]; PE: OIChangeEntry[] }
  onFeedback: (message: string) => void
}) {
  const resolvedHistory = history ?? { CE: [], PE: [] }
  const resolvedPair =
    pair ??
    ({
      Strike: strike,
      CE: createEmptyLeg(strike, "CE"),
      PE: createEmptyLeg(strike, "PE"),
      Bias: "NEUTRAL",
      Score: 0,
      Strength: "WAIT",
    } satisfies PairSignal)

  const distance = strike - spot
  const distanceClass = Math.abs(distance) <= 50 ? "atm" : distance > 0 ? "up" : "down"

  return (
    <tr>
      <td className="chain-leg">
        <OptionLegCell
          leg={resolvedPair.CE}
          history={resolvedHistory.CE}
          align="left"
          strike={strike}
          optionType="CE"
          onFeedback={onFeedback}
        />
      </td>
      <td className="chain-strike">
        <div className={`strike-chip ${distanceClass}`}>
          <strong>{formatNumber(strike, 0)}</strong>
          <span>{distance >= 0 ? `+${formatNumber(distance)}` : formatNumber(distance)}</span>
        </div>
      </td>
      <td className="chain-leg">
        <OptionLegCell
          leg={resolvedPair.PE}
          history={resolvedHistory.PE}
          align="right"
          strike={strike}
          optionType="PE"
          onFeedback={onFeedback}
        />
      </td>
    </tr>
  )
}

function SpotRow({ spot }: { spot: number }) {
  return (
    <tr className="spot-row">
      <td className="chain-leg">
        <div className="spot-side-label">Below spot focus</div>
      </td>
      <td className="chain-strike">
        <div className="strike-chip spot">
          <strong>{formatNumber(spot)}</strong>
          <span>SPOT</span>
        </div>
      </td>
      <td className="chain-leg">
        <div className="spot-side-label right">Above spot focus</div>
      </td>
    </tr>
  )
}

export default function OptionTable() {
  const [tradeMessage, setTradeMessage] = useState("")
  const [oiHistoryMap, setOIHistoryMap] = useState<OIHistoryMap>({})
  const hydrate = useOptionStore((state) => state.hydrate)
  const strikeMap = useOptionStore((state) => state.strikeMap)
  const spot = useOptionStore((state) => state.spot)
  const portfolio = useOptionStore((state) => state.portfolio)
  const openPositions = useOptionStore((state) => state.openPositions)
  const closedPositions = useOptionStore((state) => state.closedPositions)
  const lastType = useOptionStore((state) => state.lastType)

  const rows = Object.values(strikeMap).sort((a, b) => a.Strike - b.Strike)
  const lowerReference =
    spot > 0 ? Math.floor((spot - 0.0001) / STRIKE_STEP) * STRIKE_STEP : rows[0]?.Strike
  const upperReference =
    spot > 0 ? Math.ceil((spot + 0.0001) / STRIKE_STEP) * STRIKE_STEP : rows[rows.length - 1]?.Strike
  const displayStrikes = [
    lowerReference !== undefined ? lowerReference - STRIKE_STEP : undefined,
    lowerReference,
    upperReference,
    upperReference !== undefined ? upperReference + STRIKE_STEP : undefined,
  ].filter(
    (strike, index, array): strike is number =>
      typeof strike === "number" && Number.isFinite(strike) && array.indexOf(strike) === index,
  )
  const atm = rows.reduce<PairSignal | null>((closest, pair) => {
    if (!closest) return pair
    return Math.abs(pair.Strike - spot) < Math.abs(closest.Strike - spot) ? pair : closest
  }, null)

  useEffect(() => {
    let active = true

    async function hydrateFromHttp() {
      try {
        const [pairsResponse, positionsResponse] = await Promise.all([
          fetch(`${API_BASE_URL}/pairs`),
          fetch(`${API_BASE_URL}/positions`),
        ])

        const nextState: {
          pairs?: PairSignal[]
          open_positions?: Position[]
          portfolio?: Portfolio
        } = {}

        if (pairsResponse.ok) {
          nextState.pairs = (await pairsResponse.json()) as PairSignal[]
        }

        if (positionsResponse.ok) {
          const positionsPayload = (await positionsResponse.json()) as {
            positions?: Position[]
            portfolio?: Portfolio
          }
          nextState.open_positions = positionsPayload.positions
          nextState.portfolio = positionsPayload.portfolio
        }

        if (!active) return

        if (nextState.pairs?.length || nextState.open_positions || nextState.portfolio) {
          hydrate(nextState, "snapshot")
        }
      } catch {
        // Keep the websocket as the primary feed and silently skip HTTP fallback failures.
      }
    }

    void hydrateFromHttp()
    const interval = window.setInterval(() => {
      void hydrateFromHttp()
    }, 5000)

    return () => {
      active = false
      window.clearInterval(interval)
    }
  }, [hydrate])

  useEffect(() => {
    if (!displayStrikes.length) {
      setOIHistoryMap({})
      return
    }

    let active = true

    async function loadOIEvents() {
      try {
        const params = new URLSearchParams({
          symbol: "NIFTY",
          strikes: displayStrikes.join(","),
          limit: "12",
        })
        const response = await fetch(`${API_BASE_URL}/oi-events?${params.toString()}`)
        if (!response.ok) return

        const payload = (await response.json()) as OIChangeEvent[]
        if (!active) return

        setOIHistoryMap(buildOIHistoryMap(payload))
      } catch {
        if (active) {
          setOIHistoryMap({})
        }
      }
    }

    void loadOIEvents()
    const interval = window.setInterval(() => {
      void loadOIEvents()
    }, 3000)

    return () => {
      active = false
      window.clearInterval(interval)
    }
  }, [displayStrikes.join(",")])

  async function handleExit(position: Position) {
    try {
      const response = await fetch(`${API_BASE_URL}/exit`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          strike: position.Strike,
          optionType: position.OptionType,
        }),
      })

      const payload = await response.json().catch(() => null)
      if (!response.ok) {
        setTradeMessage(payload?.error || "Exit failed")
        return
      }

      setTradeMessage(`Exited ${position.OptionType} ${formatNumber(position.Strike, 0)}`)
    } catch {
      setTradeMessage("Exit request failed")
    }
  }

  return (
    <div className="terminal-shell">
      <header className="topbar">
        <div>
          <p className="eyebrow">Ludrum Terminal</p>
          <h1>Options Flow Deck</h1>
        </div>
        <div className="status-grid">
          <div className="status-card">
            <span>Feed</span>
            <strong>{lastType ? `LIVE ${lastType.toUpperCase()}` : "WAITING"}</strong>
          </div>
          <div className="status-card">
            <span>Spot</span>
            <strong>{formatNumber(spot)}</strong>
          </div>
          <div className="status-card">
            <span>ATM</span>
            <strong>{atm ? formatNumber(atm.Strike, 0) : "-"}</strong>
          </div>
          <div className="status-card">
            <span>Open Trades</span>
            <strong>{openPositions.length}</strong>
          </div>
        </div>
      </header>

      {tradeMessage ? (
        <section className="panel trade-feedback-panel">
          <div className="trade-feedback">
            <span>Paper trade status</span>
            <strong>{tradeMessage}</strong>
          </div>
        </section>
      ) : null}

      <section className="dashboard-grid">
        <article className="panel">
          <div className="panel-head">
            <h2>Capital</h2>
            <span>Simulator ledger</span>
          </div>
          <div className="ledger-grid">
            <div>
              <span>Initial</span>
              <strong>{formatNumber(portfolio?.InitialCapital)}</strong>
            </div>
            <div>
              <span>Available</span>
              <strong>{formatNumber(portfolio?.AvailableCapital)}</strong>
            </div>
            <div>
              <span>Used Margin</span>
              <strong>{formatNumber(portfolio?.UsedMargin)}</strong>
            </div>
            <div>
              <span>Realized</span>
              <strong className={toneClass(portfolio?.RealizedPnL)}>
                {formatNumber(portfolio?.RealizedPnL)}
              </strong>
            </div>
            <div>
              <span>Unrealized</span>
              <strong className={toneClass(portfolio?.UnrealizedPnL)}>
                {formatNumber(portfolio?.UnrealizedPnL)}
              </strong>
            </div>
          </div>
        </article>

        <article className="panel">
          <div className="panel-head">
            <h2>Position Blotter</h2>
            <span>{openPositions.length ? "Live exposure" : "No active positions"}</span>
          </div>
          <div className="table-wrap">
            <table className="mini-table">
              <thead>
                <tr>
                  <th>Symbol</th>
                  <th>Type</th>
                  <th>Strike</th>
                  <th>Side</th>
                  <th>Qty</th>
                  <th>Avg</th>
                  <th>uPnL</th>
                  <th>Time</th>
                  <th>Action</th>
                </tr>
              </thead>
              <tbody>
                {openPositions.length ? (
                  openPositions.map((position, index) => (
                    <PositionRow
                      key={`${position.Strike}-${position.OptionType}-${index}`}
                      position={position}
                      onExit={handleExit}
                    />
                  ))
                ) : (
                  <tr>
                    <td colSpan={9} className="empty-cell">
                      Waiting for simulator entries...
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </article>
      </section>

      <section className="panel chain-panel">
        <div className="panel-head">
          <h2>Options Chain Radar</h2>
          <span>{displayStrikes.length} live focus strikes</span>
        </div>
        {displayStrikes.length ? (
          <div className="table-wrap">
            <table className="chain-table">
              <thead>
                <tr>
                  <th>Call Stack</th>
                  <th>Strike</th>
                  <th>Put Stack</th>
                </tr>
              </thead>
              <tbody>
                {displayStrikes.length >= 2 ? (
                  <>
                    {displayStrikes
                      .filter((strike) => strike < spot)
                      .sort((a, b) => a - b)
                      .map((strike) => (
                        <ChainRow
                          key={strike}
                          strike={strike}
                          pair={strikeMap[strike]}
                          spot={spot}
                          history={oiHistoryMap[strike]}
                          onFeedback={setTradeMessage}
                        />
                      ))}
                    <SpotRow spot={spot} />
                    {displayStrikes
                      .filter((strike) => strike > spot)
                      .sort((a, b) => a - b)
                      .map((strike) => (
                        <ChainRow
                          key={strike}
                          strike={strike}
                          pair={strikeMap[strike]}
                          spot={spot}
                          history={oiHistoryMap[strike]}
                          onFeedback={setTradeMessage}
                        />
                      ))}
                  </>
                ) : (
                  displayStrikes.map((strike) => (
                    <ChainRow
                      key={strike}
                      strike={strike}
                      pair={strikeMap[strike]}
                      spot={spot}
                      history={oiHistoryMap[strike]}
                      onFeedback={setTradeMessage}
                    />
                  ))
                )}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="empty-state">
            <p>Waiting for the backend snapshot from {API_BASE_URL} and the configured websocket feed.</p>
            <p>The UI is now wired to `spot`, `pairs`, `open_positions`, `closed_positions`, and `portfolio`.</p>
          </div>
        )}
      </section>

      <section className="dashboard-grid bottom">
        <article className="panel">
          <div className="panel-head">
            <h2>Closed Trades</h2>
            <span>{closedPositions.length} archived fills</span>
          </div>
          <div className="table-wrap">
            <table className="mini-table">
              <thead>
                <tr>
                  <th>Type</th>
                  <th>Strike</th>
                  <th>Side</th>
                  <th>Qty</th>
                  <th>Realized</th>
                  <th>Exit</th>
                </tr>
              </thead>
              <tbody>
                {closedPositions.length ? (
                  closedPositions.slice(-6).reverse().map((position, index) => (
                    <tr key={`${position.Strike}-${position.OptionType}-${index}`}>
                      <td>{position.OptionType}</td>
                      <td>{formatNumber(position.Strike, 0)}</td>
                      <td>{position.Side}</td>
                      <td>{position.Qty}</td>
                      <td className={toneClass(position.RealizedPnL)}>{formatNumber(position.RealizedPnL)}</td>
                      <td>{formatTime(position.LastUpdate || position.EntryTime)}</td>
                    </tr>
                  ))
                ) : (
                  <tr>
                    <td colSpan={6} className="empty-cell">
                      No closed positions yet.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </article>

        <article className="panel">
          <div className="panel-head">
            <h2>Terminal Notes</h2>
            <span>Backend-driven fields</span>
          </div>
          <div className="notes">
            <p>`PairSignal.Strike`, `CE`, `PE`, `Bias`, `Score`, and `Strength` drive the chain rows.</p>
            <p>`StrikeAnalytics` powers LTP series, OI change, current OI, velocity, acceleration, and signal badges.</p>
            <p>`portfolio`, `open_positions`, and `closed_positions` populate the blotter and ledger without mock data.</p>
          </div>
        </article>
      </section>
    </div>
  )
}
