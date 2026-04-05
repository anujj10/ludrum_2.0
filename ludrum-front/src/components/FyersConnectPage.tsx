import { useEffect, useState } from "react"

type FyersConnectStatus = {
  connected: boolean
  status: string
  broker_user_id?: string
  token_expires_at?: string
  last_connected_at?: string
}

type FyersConnectPageProps = {
  clientId: string
  status: FyersConnectStatus | null
  onStartConnect: () => Promise<{ ok: boolean; error?: string }>
  onLogout: () => void
  initialMessage?: string
}

const CONNECT_COOLDOWN_MS = 45_000

function formatConnectDate(value?: string) {
  if (!value) return "-"
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return "-"
  return date.toLocaleString("en-IN", {
    dateStyle: "medium",
    timeStyle: "short",
    timeZone: "Asia/Kolkata",
  })
}

function parseErrorMessage(message?: string) {
  switch (message) {
    case "token_exchange_failed":
      return "FYERS did not finish the token exchange. Wait a bit and try connect again."
    case "invalid_broker_state":
      return "The FYERS session expired or was reused. Start a fresh broker connect."
    case "missing_broker_callback_params":
      return "FYERS did not return the required auth callback values."
    case "account_setup_failed":
    case "account_store_failed":
    case "runtime_status_failed":
      return "Your broker account could not be stored yet. Please try again."
    case "broker_app_not_configured":
      return "FYERS broker auth is not configured on the server."
    default:
      return message || ""
  }
}

export default function FyersConnectPage({ clientId, status, onStartConnect, onLogout, initialMessage }: FyersConnectPageProps) {
  const [submitting, setSubmitting] = useState(false)
  const [message, setMessage] = useState("")
  const [cooldownUntil, setCooldownUntil] = useState(0)

  useEffect(() => {
    const parsed = parseErrorMessage(initialMessage)
    if (parsed) {
      setMessage(parsed)
    }
  }, [initialMessage])

  useEffect(() => {
    if (!cooldownUntil) return
    const interval = window.setInterval(() => {
      if (Date.now() >= cooldownUntil) {
        setCooldownUntil(0)
        window.clearInterval(interval)
      }
    }, 1000)

    return () => window.clearInterval(interval)
  }, [cooldownUntil])

  async function handleConnect() {
    if (cooldownUntil && Date.now() < cooldownUntil) {
      const seconds = Math.max(1, Math.ceil((cooldownUntil - Date.now()) / 1000))
      setMessage(`Please wait ${seconds}s before starting FYERS auth again.`)
      return
    }

    setSubmitting(true)
    setMessage("")
    const result = await onStartConnect()
    setSubmitting(false)
    if (!result.ok) {
      setMessage(result.error || "Unable to start FYERS connection right now.")
      setCooldownUntil(Date.now() + CONNECT_COOLDOWN_MS)
    }
  }

  const cooldownSeconds = cooldownUntil && Date.now() < cooldownUntil ? Math.max(1, Math.ceil((cooldownUntil - Date.now()) / 1000)) : 0

  return (
    <main className="login-shell">
      <section className="login-panel">
        <div className="login-brand">
          <p className="eyebrow">Broker Access</p>
          <h1>Connect your FYERS account</h1>
          <p>
            Your platform session is active as <strong>{clientId}</strong>. Link your FYERS broker account so we can exchange the auth
            code, store your token set, and attach your market runtime to your own session.
          </p>
          <div className="login-note">
            <strong>What happens next</strong>
            <span>We redirect you to FYERS, receive the auth code on callback, exchange it for tokens, and store that account against your user.</span>
          </div>
        </div>

        <div className="login-card">
          <div className="admin-login-card-head">
            <strong>FYERS connection status</strong>
            <span>{status?.connected ? "Linked" : "Not linked"}</span>
          </div>

          <div className="fyers-overview-cards">
            <article>
              <span>Status</span>
              <strong>{status?.status || "unlinked"}</strong>
            </article>
            <article>
              <span>Broker user</span>
              <strong>{status?.broker_user_id || "-"}</strong>
            </article>
            <article>
              <span>Token expiry</span>
              <strong>{formatConnectDate(status?.token_expires_at)}</strong>
            </article>
            <article>
              <span>Last linked</span>
              <strong>{formatConnectDate(status?.last_connected_at)}</strong>
            </article>
          </div>

          <div className="otp-actions">
            <button type="button" className="hero-btn secondary" onClick={onLogout}>
              Log out
            </button>
            <button type="button" className="hero-btn primary submit" onClick={handleConnect} disabled={submitting || cooldownSeconds > 0}>
              {submitting ? "Redirecting..." : cooldownSeconds > 0 ? `Retry in ${cooldownSeconds}s` : status?.connected ? "Reconnect FYERS" : "Connect FYERS"}
            </button>
          </div>

          {message ? <div className="login-message">{message}</div> : null}
        </div>
      </section>
    </main>
  )
}
