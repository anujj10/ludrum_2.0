import { useEffect, useState } from "react"
import "./App.css"
import AdminDashboard from "./components/AdminDashboard"
import AdminLoginPage from "./components/AdminLoginPage"
import MarketClosedScreen from "./components/MarketClosedScreen"
import OptionTable from "./components/OptionTable"
import { ADMIN_SESSION_STORAGE_KEY, API_BASE_URL } from "./config"
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
  const [marketOverrideReason, setMarketOverrideReason] = useState("")
  const [adminAuthState, setAdminAuthState] = useState<"idle" | "loading" | "ready" | "blocked">("idle")
  const [adminClientId, setAdminClientId] = useState("")
  const hostname = window.location.hostname.toLowerCase()
  const isAdminHost = hostname === "admin.ludrum.online" || hostname.startsWith("admin.")

  useEffect(() => {
    const interval = window.setInterval(() => {
      setIsMarketOpen(isIndianMarketOpen(new Date()))
    }, 60000)

    return () => window.clearInterval(interval)
  }, [])

  useEffect(() => {
    if (isAdminHost) {
      return
    }

    let active = true

    const refreshStatus = () => {
      fetch(`${API_BASE_URL}/market-status`)
        .then(async (response) => {
          const payload = (await response.json().catch(() => ({}))) as {
            forced_closed?: boolean
            reason?: string
          }
          if (!active || !response.ok) return
          setMarketOverrideReason(payload.forced_closed ? payload.reason || "Markets are down right now. Please check back shortly." : "")
        })
        .catch(() => {
          if (active) {
            setMarketOverrideReason("")
          }
        })
    }

    refreshStatus()
    const interval = window.setInterval(refreshStatus, 30000)

    return () => {
      active = false
      window.clearInterval(interval)
    }
  }, [isAdminHost])

  useEffect(() => {
    if (!isAdminHost) {
      setAdminAuthState("idle")
      setAdminClientId("")
      return
    }

    const token = window.localStorage.getItem(ADMIN_SESSION_STORAGE_KEY)
    if (!token) {
      setAdminAuthState("blocked")
      return
    }

    let active = true
    setAdminAuthState("loading")

    fetch(`${API_BASE_URL}/auth/admin/me`, {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    })
      .then(async (response) => {
        if (!active) return
        const payload = (await response.json().catch(() => ({}))) as {
          admin?: { client_id?: string }
        }
        if (!response.ok || !payload.admin?.client_id) {
          window.localStorage.removeItem(ADMIN_SESSION_STORAGE_KEY)
          setAdminAuthState("blocked")
          setAdminClientId("")
          return
        }
        setAdminClientId(payload.admin.client_id)
        setAdminAuthState("ready")
      })
      .catch(() => {
        if (!active) return
        window.localStorage.removeItem(ADMIN_SESSION_STORAGE_KEY)
        setAdminAuthState("blocked")
        setAdminClientId("")
      })

    return () => {
      active = false
    }
  }, [isAdminHost])

  useEffect(() => {
    if (isAdminHost || !isMarketOpen || marketOverrideReason) {
      return
    }

    const ws = initWebSocket()

    return () => {
      if (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING) {
        ws.close()
      }
    }
  }, [isAdminHost, isMarketOpen, marketOverrideReason])

  async function handleAdminLogin(clientId: string, password: string) {
    try {
      const response = await fetch(`${API_BASE_URL}/auth/admin/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          client_id: clientId,
          password,
        }),
      })

      const payload = (await response.json().catch(() => ({}))) as {
        error?: string
        token?: string
        admin?: { client_id?: string }
      }

      if (!response.ok || !payload.token || !payload.admin?.client_id) {
        return {
          ok: false,
          error: payload.error || "Unable to verify admin access right now.",
        }
      }

      window.localStorage.setItem(ADMIN_SESSION_STORAGE_KEY, payload.token)
      setAdminClientId(payload.admin.client_id)
      setAdminAuthState("ready")
      return { ok: true }
    } catch {
      return {
        ok: false,
        error: "Admin login failed. Check the API and try again.",
      }
    }
  }

  async function handleAdminLogout() {
    const token = window.localStorage.getItem(ADMIN_SESSION_STORAGE_KEY)
    if (token) {
      await fetch(`${API_BASE_URL}/auth/admin/logout`, {
        method: "POST",
        headers: {
          Authorization: `Bearer ${token}`,
        },
      }).catch(() => undefined)
    }

    window.localStorage.removeItem(ADMIN_SESSION_STORAGE_KEY)
    setAdminClientId("")
    setAdminAuthState("blocked")
  }

  if (isAdminHost) {
    if (adminAuthState === "loading") {
      return (
        <main className="admin-shell">
          <section className="admin-login-panel">
            <div className="admin-login-copy">
              <p className="eyebrow">Admin Console</p>
              <h1>Checking admin session</h1>
              <p>Verifying your backend session before opening the operations dashboard.</p>
            </div>
          </section>
        </main>
      )
    }

    return adminAuthState === "ready" ? (
      <AdminDashboard adminToken={window.localStorage.getItem(ADMIN_SESSION_STORAGE_KEY) || ""} clientId={adminClientId} onLogout={handleAdminLogout} />
    ) : (
      <AdminLoginPage onLogin={handleAdminLogin} />
    )
  }

  return isMarketOpen && !marketOverrideReason ? <OptionTable /> : <MarketClosedScreen overrideReason={marketOverrideReason} />
}
