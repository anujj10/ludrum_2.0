import { useState } from "react"

import { API_BASE_URL, AUTH_TOKEN_STORAGE_KEY } from "../config"

export default function LoginPage() {
  const [step, setStep] = useState<"credentials" | "otp">("credentials")
  const [clientId, setClientId] = useState("")
  const [password, setPassword] = useState("")
  const [otp, setOTP] = useState("")
  const [message, setMessage] = useState("")
  const [submitting, setSubmitting] = useState(false)

  async function handleCredentialsSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setSubmitting(true)
    setMessage("")

    try {
      const response = await fetch(`${API_BASE_URL}/auth/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          client_id: clientId,
          password,
        }),
      })

      const payload = (await response.json()) as {
        error?: string
        message?: string
        otp_preview?: string
        warning?: string
      }

      if (!response.ok) {
        setMessage(payload.error || "Unable to continue")
        return
      }

      setStep("otp")
      setMessage(payload.warning ? `${payload.message} ${payload.warning}${payload.otp_preview ? ` OTP: ${payload.otp_preview}` : ""}` : payload.message || "OTP sent")
    } catch {
      setMessage("Login request failed")
    } finally {
      setSubmitting(false)
    }
  }

  async function handleOTPSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setSubmitting(true)
    setMessage("")

    try {
      const response = await fetch(`${API_BASE_URL}/auth/verify-otp`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          client_id: clientId,
          otp,
        }),
      })

      const payload = (await response.json()) as {
        error?: string
        token?: string
      }

      if (!response.ok || !payload.token) {
        setMessage(payload.error || "OTP verification failed")
        return
      }

      window.localStorage.setItem(AUTH_TOKEN_STORAGE_KEY, payload.token)
      window.location.href = "/terminal"
    } catch {
      setMessage("OTP verification failed")
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <main className="login-shell">
      <section className="login-panel">
        <div className="login-brand">
          <a className="back-link" href="/">
            Back to home
          </a>
          <p className="eyebrow">Private Access</p>
          <h1>Client login</h1>
          <p>
            Enter the credentials issued after your beta request is verified. A one-time code is then sent to your registered email
            for final access confirmation.
          </p>
        </div>

        <div className="login-card">
          <div className="login-stepper">
            <span className={step === "credentials" ? "active" : ""}>1. Credentials</span>
            <span className={step === "otp" ? "active" : ""}>2. Email OTP</span>
          </div>

          {step === "credentials" ? (
            <form className="login-form" onSubmit={handleCredentialsSubmit}>
              <label>
                <span>Client ID</span>
                <input type="text" placeholder="IDX-10284" value={clientId} onChange={(event) => setClientId(event.target.value)} />
              </label>
              <label>
                <span>Password</span>
                <input
                  type="password"
                  placeholder="Enter your password"
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                />
              </label>
              <button type="submit" className="hero-btn primary submit" disabled={submitting}>
                {submitting ? "Checking..." : "Continue to OTP"}
              </button>
            </form>
          ) : (
            <form className="login-form" onSubmit={handleOTPSubmit}>
              <label>
                <span>Email OTP</span>
                <input type="text" inputMode="numeric" placeholder="Enter 6-digit code" value={otp} onChange={(event) => setOTP(event.target.value)} />
              </label>
              <div className="otp-actions">
                <button type="button" className="hero-btn secondary" onClick={() => setStep("credentials")}>
                  Change credentials
                </button>
                <button type="submit" className="hero-btn primary submit" disabled={submitting}>
                  {submitting ? "Verifying..." : "Verify & Enter"}
                </button>
              </div>
            </form>
          )}

          {message ? <div className="login-message">{message}</div> : null}

          <div className="login-note">
            <strong>Current scope</strong>
            <span>This flow now submits credentials, sends OTP, verifies the code, and stores a terminal session token for market access.</span>
          </div>
        </div>
      </section>
    </main>
  )
}
