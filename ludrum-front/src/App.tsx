import { useEffect, useState } from "react"
import "./App.css"
import HomePage from "./components/HomePage"
import LoginPage from "./components/LoginPage"
import MarketClosedScreen from "./components/MarketClosedScreen"
import OptionTable from "./components/OptionTable"
import { initWebSocket } from "./ws/socket"

const MARKET_TIMEZONE = "Asia/Kolkata"
const MARKET_OPEN_MINUTES = 9 * 60 + 15
const MARKET_CLOSE_MINUTES = 15 * 60 + 30

function isIndianMarketOpen(now: Date) {
  const formatter = new Intl.DateTimeFormat("en-GB", {
    timeZone: MARKET_TIMEZONE,
    weekday: "short",
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  })

  const parts = formatter.formatToParts(now)
  const map = Object.fromEntries(parts.map((part) => [part.type, part.value]))
  const weekday = map.weekday ?? ""
  const hour = Number(map.hour ?? "0")
  const minute = Number(map.minute ?? "0")
  const minutes = hour * 60 + minute
  const isWeekday = ["Mon", "Tue", "Wed", "Thu", "Fri"].includes(weekday)

  return isWeekday && minutes >= MARKET_OPEN_MINUTES && minutes < MARKET_CLOSE_MINUTES
}

export default function App() {
  const [isMarketOpen, setIsMarketOpen] = useState(() => isIndianMarketOpen(new Date()))
  const pathname = window.location.pathname.replace(/\/+$/, "") || "/"

  const isTerminalRoute = pathname === "/terminal"
  const isLoginRoute = pathname === "/login"

  useEffect(() => {
    const interval = window.setInterval(() => {
      setIsMarketOpen(isIndianMarketOpen(new Date()))
    }, 60000)

    return () => window.clearInterval(interval)
  }, [])

  useEffect(() => {
    if (!isMarketOpen || !isTerminalRoute) {
      return
    }

    const ws = initWebSocket()

    return () => {
      if (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING) {
        ws.close()
      }
    }
  }, [isMarketOpen, isTerminalRoute])

  if (isLoginRoute) {
    return <LoginPage />
  }

  if (!isTerminalRoute) {
    return <HomePage />
  }

  return isMarketOpen ? <OptionTable /> : <MarketClosedScreen />
}
