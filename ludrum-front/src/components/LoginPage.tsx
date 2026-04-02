import { useState } from "react"

export default function LoginPage() {
  const [step, setStep] = useState<"credentials" | "otp">("credentials")

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
            <form
              className="login-form"
              onSubmit={(event) => {
                event.preventDefault()
                setStep("otp")
              }}
            >
              <label>
                <span>Client ID</span>
                <input type="text" placeholder="IDX-10284" />
              </label>
              <label>
                <span>Password</span>
                <input type="password" placeholder="Enter your password" />
              </label>
              <button type="submit" className="hero-btn primary submit">
                Continue to OTP
              </button>
            </form>
          ) : (
            <form className="login-form">
              <label>
                <span>Email OTP</span>
                <input type="text" inputMode="numeric" placeholder="Enter 6-digit code" />
              </label>
              <div className="otp-actions">
                <button type="button" className="hero-btn secondary" onClick={() => setStep("credentials")}>
                  Change credentials
                </button>
                <button type="button" className="hero-btn primary submit">
                  Verify & Enter
                </button>
              </div>
            </form>
          )}

          <div className="login-note">
            <strong>Current scope</strong>
            <span>This page is ready for real auth wiring next: credential lookup, email OTP send, OTP verify, and session creation.</span>
          </div>
        </div>
      </section>
    </main>
  )
}
