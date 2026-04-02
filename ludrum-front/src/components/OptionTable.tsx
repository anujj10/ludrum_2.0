import { useEffect, useState } from "react"
import { useOptionStore } from "../store/useOptionStore"
import { API_BASE_URL, authHeaders } from "../config"
import type { OIChangeEvent, PairSignal, StrikeAnalytics } from "../types/option"

const STRIKE_STEP = 50

type OIChangeEntry = {
  value: number
  time: string
}

type OIHistoryMap = Record<number, { CE: OIChangeEntry[]; PE: OIChangeEntry[] }>

function appendOIHistoryEntry(entries: OIChangeEntry[] | undefined, value: number | undefined) {
  if (value === undefined || value === null || Number.isNaN(value) || value === 0) {
    return entries ?? []
  }

  const nextEntry = {
    value,
    time: formatEventTime(new Date().toISOString()),
  }

  if (!entries?.length) {
    return [nextEntry]
  }

  const lastEntry = entries[entries.length - 1]
  if (lastEntry.value === value) {
    return entries
  }

  return [...entries.slice(-11), nextEntry]
}

function mergeOIHistoryMaps(apiHistory: OIHistoryMap, runtimeHistory: OIHistoryMap) {
  const merged: OIHistoryMap = { ...apiHistory }

  for (const [rawStrike, runtimeEntry] of Object.entries(runtimeHistory)) {
    const strike = Number(rawStrike)
    const existing = merged[strike] ?? { CE: [], PE: [] }

    merged[strike] = {
      CE: existing.CE.length ? existing.CE : runtimeEntry.CE,
      PE: existing.PE.length ? existing.PE : runtimeEntry.PE,
    }
  }

  return merged
}

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

function OptionLegCell({
  leg,
  history,
  align,
}: {
  leg: StrikeAnalytics
  history: OIChangeEntry[]
  align: "left" | "right"
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
    </div>
  )
}

function ChainRow({
  pair,
  strike,
  spot,
  history,
}: {
  pair?: PairSignal
  strike: number
  spot: number
  history?: { CE: OIChangeEntry[]; PE: OIChangeEntry[] }
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
        <OptionLegCell leg={resolvedPair.CE} history={resolvedHistory.CE} align="left" />
      </td>
      <td className="chain-strike">
        <div className={`strike-chip ${distanceClass}`}>
          <strong>{formatNumber(strike, 0)}</strong>
          <span>{distance >= 0 ? `+${formatNumber(distance)}` : formatNumber(distance)}</span>
        </div>
      </td>
      <td className="chain-leg">
        <OptionLegCell leg={resolvedPair.PE} history={resolvedHistory.PE} align="right" />
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
  const [oiHistoryMap, setOIHistoryMap] = useState<OIHistoryMap>({})
  const [runtimeOIHistoryMap, setRuntimeOIHistoryMap] = useState<OIHistoryMap>({})
  const hydrate = useOptionStore((state) => state.hydrate)
  const strikeMap = useOptionStore((state) => state.strikeMap)
  const spot = useOptionStore((state) => state.spot)
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
  const mergedOIHistoryMap = mergeOIHistoryMaps(oiHistoryMap, runtimeOIHistoryMap)

  useEffect(() => {
    let active = true

    async function hydrateFromHttp() {
      try {
        const pairsResponse = await fetch(`${API_BASE_URL}/pairs`, {
          headers: authHeaders(),
        })
        const nextState: { pairs?: PairSignal[] } = {}

        if (pairsResponse.ok) {
          nextState.pairs = (await pairsResponse.json()) as PairSignal[]
        }

        if (!active) return

        if (nextState.pairs?.length) {
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
        const response = await fetch(`${API_BASE_URL}/oi-events?${params.toString()}`, {
          headers: authHeaders(),
        })
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

  useEffect(() => {
    if (!displayStrikes.length) return

    setRuntimeOIHistoryMap((previous) => {
      let changed = false
      const next: OIHistoryMap = { ...previous }

      for (const strike of displayStrikes) {
        const pair = strikeMap[strike]
        if (!pair) continue

        const current = next[strike] ?? { CE: [], PE: [] }
        const nextCE = appendOIHistoryEntry(current.CE, pair.CE?.OIChange)
        const nextPE = appendOIHistoryEntry(current.PE, pair.PE?.OIChange)

        if (nextCE !== current.CE || nextPE !== current.PE) {
          next[strike] = { CE: nextCE, PE: nextPE }
          changed = true
        }
      }

      return changed ? next : previous
    })
  }, [displayStrikes.join(","), strikeMap])

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
        </div>
      </header>

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
                          history={mergedOIHistoryMap[strike]}
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
                          history={mergedOIHistoryMap[strike]}
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
                      history={mergedOIHistoryMap[strike]}
                    />
                  ))
                )}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="empty-state">
            <p>Waiting for the backend snapshot from {API_BASE_URL} and the configured websocket feed.</p>
            <p>The UI is now wired to `spot` and `pairs` for a read-only live market view.</p>
          </div>
        )}
      </section>
    </div>
  )
}
