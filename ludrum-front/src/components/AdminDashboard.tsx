import { useEffect, useState } from "react"

import { API_BASE_URL } from "../config"

type AdminDashboardProps = {
  onLogout: () => void
  clientId: string
  adminToken: string
}

const intakeQueue = [
  {
    name: "Rohit Sharma",
    email: "rohit.trades@gmail.com",
    phone: "+91 98xxxx1304",
    style: "Intraday index options",
    status: "Review",
  },
  {
    name: "Manya Jain",
    email: "manya.market@outlook.com",
    phone: "+91 88xxxx4290",
    style: "Analysis only",
    status: "Approved",
  },
  {
    name: "Arjun Mehta",
    email: "arjun.delta@yahoo.com",
    phone: "+91 97xxxx7712",
    style: "Short swing options",
    status: "Pending",
  },
] as const

const auditFeed = [
  "New beta request received from Manya Jain.",
  "API domain healthy and serving HTTPS.",
  "Market collector in after-hours standby mode.",
  "Admin subdomain verified and SSL active.",
] as const

type FyersOverviewUser = {
  user_id: number
  full_name: string
  email: string
  client_id: string
  fyers_account_id?: number
  fyers_status: string
  broker_user_id?: string
  runtime_state: string
  token_expires_at?: string
  last_connected_at?: string
  runtime_last_tick_at?: string
  runtime_last_error?: string
}

type FyersOverview = {
  total_users: number
  linked_accounts: number
  stored_tokens: number
  active_runtimes: number
  users: FyersOverviewUser[]
}

function formatAdminDate(value?: string) {
  if (!value) return "-"
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return "-"
  return date.toLocaleString("en-IN", {
    dateStyle: "medium",
    timeStyle: "short",
    timeZone: "Asia/Kolkata",
  })
}

function runtimeTone(value: string) {
  switch (value) {
    case "running":
    case "linked":
    case "active":
      return "up"
    case "error":
    case "failed":
      return "down"
    default:
      return "flat"
  }
}

export default function AdminDashboard({ onLogout, clientId, adminToken }: AdminDashboardProps) {
  const updatedAt = new Intl.DateTimeFormat("en-IN", {
    dateStyle: "medium",
    timeStyle: "short",
    timeZone: "Asia/Kolkata",
  }).format(new Date())
  const [overrideEnabled, setOverrideEnabled] = useState(false)
  const [overrideReason, setOverrideReason] = useState("Markets are down right now. Please check back shortly.")
  const [overrideMessage, setOverrideMessage] = useState("")
  const [overrideSaving, setOverrideSaving] = useState(false)
  const [fyersOverview, setFyersOverview] = useState<FyersOverview | null>(null)

  async function loadFyersOverview() {
    const response = await fetch(`${API_BASE_URL}/auth/admin/fyers/overview`, {
      headers: {
        Authorization: `Bearer ${adminToken}`,
      },
    })
    const payload = (await response.json().catch(() => null)) as FyersOverview | null
    if (!response.ok || !payload) {
      throw new Error("Unable to load FYERS overview")
    }
    setFyersOverview(payload)
  }

  useEffect(() => {
    let active = true

    fetch(`${API_BASE_URL}/auth/admin/market-override`, {
      headers: {
        Authorization: `Bearer ${adminToken}`,
      },
    })
      .then(async (response) => {
        const payload = (await response.json().catch(() => ({}))) as {
          enabled?: boolean
          reason?: string
        }
        if (!active || !response.ok) return
        setOverrideEnabled(Boolean(payload.enabled))
        if (payload.reason) {
          setOverrideReason(payload.reason)
        }
      })
      .catch(() => undefined)

    return () => {
      active = false
    }
  }, [adminToken])

  useEffect(() => {
    let active = true

    loadFyersOverview().catch(() => {
      if (active) {
        setFyersOverview(null)
      }
    })

    return () => {
      active = false
    }
  }, [adminToken])

  async function handleOverrideSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setOverrideSaving(true)
    setOverrideMessage("")

    try {
      const response = await fetch(`${API_BASE_URL}/auth/admin/market-override`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${adminToken}`,
        },
        body: JSON.stringify({
          enabled: !overrideEnabled,
          reason: overrideReason,
        }),
      })

      const payload = (await response.json().catch(() => ({}))) as {
        enabled?: boolean
        reason?: string
        error?: string
      }

      if (!response.ok) {
        setOverrideMessage(payload.error || "Unable to update market override right now.")
        return
      }

      setOverrideEnabled(Boolean(payload.enabled))
      if (typeof payload.reason === "string") {
        setOverrideReason(payload.reason || "Markets are down right now. Please check back shortly.")
      }
      setOverrideMessage(payload.enabled ? "Manual market-down page is now active." : "Manual market-down page has been cleared.")
    } catch {
      setOverrideMessage("Unable to update market override right now.")
    } finally {
      setOverrideSaving(false)
    }
  }

  return (
    <main className="admin-shell">
      <section className="admin-console">
        <header className="admin-topbar">
          <div>
            <p className="eyebrow">Ludrum Admin</p>
            <h1>Operations dashboard</h1>
            <p className="admin-subcopy">Review beta onboarding, keep an eye on system state, and manage the private rollout from one place.</p>
          </div>
          <div className="admin-topbar-actions">
            <div className="admin-updated-at">
              <span>Last refresh</span>
              <strong>{updatedAt}</strong>
            </div>
            <div className="admin-updated-at">
              <span>Signed in as</span>
              <strong>{clientId}</strong>
            </div>
            <button type="button" className="hero-btn secondary" onClick={onLogout}>
              Log out
            </button>
          </div>
        </header>

        <section className="admin-metrics">
          <article>
            <span>Beta requests</span>
            <strong>24</strong>
            <small>7 awaiting review</small>
          </article>
          <article>
            <span>Active client IDs</span>
            <strong>11</strong>
            <small>3 issued today</small>
          </article>
          <article>
            <span>API status</span>
            <strong>Healthy</strong>
            <small>After-hours standby</small>
          </article>
          <article>
            <span>Market state</span>
            <strong>Closed</strong>
            <small>Live collectors paused</small>
          </article>
        </section>

        <section className="admin-panel fyers-overview-panel">
          <div className="panel-head">
            <h2>FYERS runtime architecture</h2>
            <span>{fyersOverview ? `${fyersOverview.users.length} recent users` : "Loading..."}</span>
          </div>

          <div className="fyers-overview-cards">
            <article>
              <span>Platform users</span>
              <strong>{fyersOverview?.total_users ?? "-"}</strong>
            </article>
            <article>
              <span>Linked FYERS accounts</span>
              <strong>{fyersOverview?.linked_accounts ?? "-"}</strong>
            </article>
            <article>
              <span>Stored token sets</span>
              <strong>{fyersOverview?.stored_tokens ?? "-"}</strong>
            </article>
            <article>
              <span>Active runtimes</span>
              <strong>{fyersOverview?.active_runtimes ?? "-"}</strong>
            </article>
          </div>

          <div className="admin-table-wrap">
            <table className="admin-table">
              <thead>
                <tr>
                  <th>User</th>
                  <th>Client ID</th>
                  <th>FYERS Link</th>
                  <th>Runtime</th>
                  <th>Token expiry</th>
                  <th>Last tick</th>
                </tr>
              </thead>
              <tbody>
                {fyersOverview?.users?.length ? (
                  fyersOverview.users.map((user) => (
                    <tr key={`${user.user_id}-${user.client_id}`}>
                      <td>
                        <div className="admin-user-cell">
                          <strong>{user.full_name}</strong>
                          <span>{user.email}</span>
                        </div>
                      </td>
                      <td>{user.client_id}</td>
                      <td>
                        <div className="admin-user-cell">
                          <span className={`admin-status-pill ${runtimeTone(user.fyers_status)}`}>{user.fyers_status}</span>
                          <small>{user.broker_user_id || "No broker user id yet"}</small>
                        </div>
                      </td>
                      <td>
                        <div className="admin-user-cell">
                          <span className={`admin-status-pill ${runtimeTone(user.runtime_state)}`}>{user.runtime_state}</span>
                          <small>{user.runtime_last_error || "Healthy"}</small>
                        </div>
                      </td>
                      <td>{formatAdminDate(user.token_expires_at)}</td>
                      <td>{formatAdminDate(user.runtime_last_tick_at || user.last_connected_at)}</td>
                    </tr>
                  ))
                ) : (
                  <tr>
                    <td colSpan={6} className="empty-cell">
                      No FYERS-linked users yet. This table will fill up as we connect each beta user to their own FYERS account.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </section>

        <section className="admin-grid">
          <article className="admin-panel">
            <div className="panel-head">
              <h2>Beta intake queue</h2>
              <span>Latest registrations</span>
            </div>

            <div className="admin-table-wrap">
              <table className="admin-table">
                <thead>
                  <tr>
                    <th>Name</th>
                    <th>Email</th>
                    <th>Phone</th>
                    <th>Style</th>
                    <th>Status</th>
                  </tr>
                </thead>
                <tbody>
                  {intakeQueue.map((user) => (
                    <tr key={user.email}>
                      <td>{user.name}</td>
                      <td>{user.email}</td>
                      <td>{user.phone}</td>
                      <td>{user.style}</td>
                      <td>
                        <span className="admin-status-pill">{user.status}</span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </article>

          <article className="admin-panel admin-panel-stack">
            <div className="panel-head">
              <h2>Operations feed</h2>
              <span>Latest notes</span>
            </div>

            <div className="admin-feed">
              {auditFeed.map((item) => (
                <div className="admin-feed-item" key={item}>
                  <span className="admin-feed-dot" />
                  <p>{item}</p>
                </div>
              ))}
            </div>

            <div className="admin-actions-card">
              <h3>Manual market override</h3>
              <form className="admin-override-form" onSubmit={handleOverrideSubmit}>
                <label>
                  <span>Closed-page message</span>
                  <textarea
                    value={overrideReason}
                    onChange={(event) => setOverrideReason(event.target.value)}
                    placeholder="Markets are down right now. Please check back shortly."
                    rows={4}
                  />
                </label>
                <div className="admin-override-actions">
                  <span className={`admin-override-state ${overrideEnabled ? "enabled" : ""}`}>
                    {overrideEnabled ? "Override active" : "Override inactive"}
                  </span>
                  <button type="submit" className="hero-btn primary" disabled={overrideSaving}>
                    {overrideSaving ? "Saving..." : overrideEnabled ? "Disable Markets Down Page" : "Enable Markets Down Page"}
                  </button>
                </div>
                {overrideMessage ? <div className="admin-override-message">{overrideMessage}</div> : null}
              </form>
            </div>
          </article>
        </section>
      </section>
    </main>
  )
}
