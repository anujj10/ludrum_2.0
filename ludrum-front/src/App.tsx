import { useEffect, useState } from "react"
import "./App.css"
import AdminDashboard from "./components/AdminDashboard"
import AdminLoginPage from "./components/AdminLoginPage"
import MarketClosedScreen from "./components/MarketClosedScreen"
import OptionTable from "./components/OptionTable"
import { ADMIN_CLIENT_ID, ADMIN_PASSWORD, ADMIN_SESSION_STORAGE_KEY } from "./config"
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
  const [adminAuthenticated, setAdminAuthenticated] = useState(() => window.localStorage.getItem(ADMIN_SESSION_STORAGE_KEY) === "active")
  const hostname = window.location.hostname.toLowerCase()
  const isAdminHost = hostname === "admin.ludrum.online" || hostname.startsWith("admin.")

  useEffect(() => {
    const interval = window.setInterval(() => {
      setIsMarketOpen(isIndianMarketOpen(new Date()))
    }, 60000)

    return () => window.clearInterval(interval)
  }, [])

  useEffect(() => {
    if (isAdminHost || !isMarketOpen) {
      return
    }

    const ws = initWebSocket()

    return () => {
      if (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING) {
        ws.close()
      }
    }
  }, [isAdminHost, isMarketOpen])

  function handleAdminLogin(clientId: string, password: string) {
    const isValid = clientId === ADMIN_CLIENT_ID && password === ADMIN_PASSWORD
    if (!isValid) {
      return false
    }

    window.localStorage.setItem(ADMIN_SESSION_STORAGE_KEY, "active")
    setAdminAuthenticated(true)
    return true
  }

  function handleAdminLogout() {
    window.localStorage.removeItem(ADMIN_SESSION_STORAGE_KEY)
    setAdminAuthenticated(false)
  }

  if (isAdminHost) {
    return adminAuthenticated ? <AdminDashboard onLogout={handleAdminLogout} /> : <AdminLoginPage onLogin={handleAdminLogin} />
  }

  return isMarketOpen ? <OptionTable /> : <MarketClosedScreen />
}
