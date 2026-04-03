const MARKET_TIMEZONE = "Asia/Kolkata"

function getIndianTimeParts(date: Date) {
  const formatter = new Intl.DateTimeFormat("en-GB", {
    timeZone: MARKET_TIMEZONE,
    weekday: "short",
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  })

  const parts = formatter.formatToParts(date)
  const map = Object.fromEntries(parts.map((part) => [part.type, part.value]))

  return {
    weekday: map.weekday ?? "",
    hour: Number(map.hour ?? "0"),
    minute: Number(map.minute ?? "0"),
  }
}

function getNextOpenLabel(now: Date) {
  const { weekday, hour, minute } = getIndianTimeParts(now)
  const minutes = hour * 60 + minute
  const openMinutes = 9 * 60 + 15
  const weekdayOrder = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"]
  const currentIndex = weekdayOrder.indexOf(weekday)

  if (currentIndex >= 0 && currentIndex <= 4 && minutes < openMinutes) {
    return "Reopens today at 9:15 AM IST"
  }

  if (currentIndex === 4 || currentIndex === 5) {
    return "Reopens Monday at 9:15 AM IST"
  }

  if (currentIndex === 6) {
    return "Reopens tomorrow at 9:15 AM IST"
  }

  return "Reopens tomorrow at 9:15 AM IST"
}

type MarketClosedScreenProps = {
  overrideReason?: string
}

export default function MarketClosedScreen({ overrideReason }: MarketClosedScreenProps) {
  const now = new Date()
  const nextOpenLabel = getNextOpenLabel(now)
  const isOverride = Boolean(overrideReason)

  return (
    <main className="market-closed-shell">
      <div className="market-closed-ambient" aria-hidden="true">
        <span className="ambient-orb orb-one" />
        <span className="ambient-orb orb-two" />
        <span className="ambient-grid" />
      </div>

      <section className="market-closed-card">
        <p className="market-closed-kicker">Ludrum Terminal</p>
        <h1>{isOverride ? "Markets are down" : "Indian market is closed"}</h1>
        <p className="market-closed-copy">
          {isOverride ? overrideReason : "Market hours are done. Live options flow will resume automatically in the next session."}
        </p>
        <div className="market-closed-meta">
          <span>{isOverride ? "Manual admin override is active" : nextOpenLabel}</span>
        </div>
      </section>
    </main>
  )
}
