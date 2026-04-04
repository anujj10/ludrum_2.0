import { useEffect, useState } from "react"
import "./App.css"
import AdminDashboard from "./components/AdminDashboard"
import AdminLoginPage from "./components/AdminLoginPage"
import FyersConnectPage from "./components/FyersConnectPage"
import LoginPage from "./components/LoginPage"
import MarketClosedScreen from "./components/MarketClosedScreen"
import OptionTable from "./components/OptionTable"
import { ADMIN_SESSION_STORAGE_KEY, API_BASE_URL, AUTH_TOKEN_STORAGE_KEY, authHeaders } from "./config"
import { initWebSocket } from "./ws/socket"

type AppUser = {
  client_id?: string
  email?: string
  full_name?: string
}

type FyersStatus = {
  connected: boolean
  status: string
  broker_user_id?: string
  token_expires_at?: string
  last_connected_at?: string
}

export default function App() {
  const [marketOverrideReason, setMarketOverrideReason] = useState("")
  const [adminAuthState, setAdminAuthState] = useState<"idle" | "loading" | "ready" | "blocked">("idle")
  const [adminClientId, setAdminClientId] = useState("")
  const [userAuthState, setUserAuthState] = useState<"idle" | "loading" | "ready" | "blocked">("idle")
  const [user, setUser] = useState<AppUser | null>(null)
  const [fyersStatus, setFyersStatus] = useState<FyersStatus | null>(null)
  const hostname = window.location.hostname.toLowerCase()
  const isAdminHost = hostname === "admin.ludrum.online" || hostname.startsWith("admin.")

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
    if (isAdminHost) {
      return
    }

    const token = window.localStorage.getItem(AUTH_TOKEN_STORAGE_KEY)
    if (!token) {
      setUserAuthState("blocked")
      setUser(null)
      setFyersStatus(null)
      return
    }

    let active = true
    setUserAuthState("loading")

    const loadUser = fetch(`${API_BASE_URL}/auth/me`, {
      headers: authHeaders(),
    }).then(async (response) => {
      const payload = (await response.json().catch(() => ({}))) as { user?: AppUser }
      if (!response.ok || !payload.user?.client_id) {
        throw new Error("invalid session")
      }
      return payload.user
    })

    const loadFyersStatus = fetch(`${API_BASE_URL}/broker/fyers/status`, {
      headers: authHeaders(),
    }).then(async (response) => {
      const payload = (await response.json().catch(() => ({}))) as FyersStatus
      if (!response.ok) {
        throw new Error("broker status unavailable")
      }
      return payload
    })

    Promise.all([loadUser, loadFyersStatus])
      .then(([nextUser, nextStatus]) => {
        if (!active) return
        setUser(nextUser)
        setFyersStatus(nextStatus)
        setUserAuthState("ready")
      })
      .catch(() => {
        if (!active) return
        window.localStorage.removeItem(AUTH_TOKEN_STORAGE_KEY)
        setUser(null)
        setFyersStatus(null)
        setUserAuthState("blocked")
      })

    return () => {
      active = false
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
    if (isAdminHost || marketOverrideReason || userAuthState !== "ready" || !fyersStatus?.connected) {
      return
    }

    const ws = initWebSocket()

    return () => {
      if (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING) {
        ws.close()
      }
    }
  }, [isAdminHost, marketOverrideReason, userAuthState, fyersStatus?.connected])

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

  async function handleUserLogout() {
    const token = window.localStorage.getItem(AUTH_TOKEN_STORAGE_KEY)
    if (token) {
      await fetch(`${API_BASE_URL}/auth/logout`, {
        method: "POST",
        headers: authHeaders(),
      }).catch(() => undefined)
    }

    window.localStorage.removeItem(AUTH_TOKEN_STORAGE_KEY)
    setUser(null)
    setFyersStatus(null)
    setUserAuthState("blocked")
  }

  async function handleFyersConnectStart() {
    try {
      const response = await fetch(`${API_BASE_URL}/broker/fyers/connect/start`, {
        method: "POST",
        headers: authHeaders(),
      })
      const payload = (await response.json().catch(() => ({}))) as { error?: string; login_url?: string }
      if (!response.ok || !payload.login_url) {
        return {
          ok: false,
          error: payload.error || "Unable to start FYERS auth right now.",
        }
      }

      window.location.href = payload.login_url
      return { ok: true }
    } catch {
      return {
        ok: false,
        error: "Unable to reach the broker auth service right now.",
      }
    }
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

  if (userAuthState === "loading") {
    return (
      <main className="login-shell">
        <section className="login-panel auth-loading-panel">
          <div className="login-brand">
            <p className="eyebrow">Private Access</p>
            <h1>Checking your session</h1>
            <p>Verifying your platform login and broker-link status before opening the terminal.</p>
          </div>
        </section>
      </main>
    )
  }

  if (userAuthState !== "ready" || !user?.client_id) {
    return <LoginPage onAuthenticated={() => window.location.reload()} />
  }

  if (!fyersStatus?.connected) {
    return (
      <FyersConnectPage
        clientId={user.client_id}
        onLogout={handleUserLogout}
        onStartConnect={handleFyersConnectStart}
        status={fyersStatus}
      />
    )
  }

  return marketOverrideReason ? <MarketClosedScreen overrideReason={marketOverrideReason} /> : <OptionTable />
}
