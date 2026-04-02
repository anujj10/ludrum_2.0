import { useState } from "react"

type AdminLoginPageProps = {
  onLogin: (clientId: string, password: string) => boolean
}

export default function AdminLoginPage({ onLogin }: AdminLoginPageProps) {
  const [clientId, setClientId] = useState("")
  const [password, setPassword] = useState("")
  const [error, setError] = useState("")

  function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const ok = onLogin(clientId.trim(), password)
    if (!ok) {
      setError("Invalid admin credentials. Double-check the client ID and password.")
      return
    }

    setError("")
  }

  return (
    <main className="admin-shell">
      <section className="admin-login-panel">
        <div className="admin-login-copy">
          <p className="eyebrow">Admin Console</p>
          <h1>Private operations access</h1>
          <p>
            Sign in with your admin client ID and password to review platform activity, access requests, and terminal health from a
            single screen.
          </p>
          <div className="admin-login-badges">
            <span>Beta intake</span>
            <span>Terminal health</span>
            <span>Access control</span>
          </div>
        </div>

        <form className="admin-login-card" onSubmit={handleSubmit}>
          <div className="admin-login-card-head">
            <strong>Admin sign in</strong>
            <span>Restricted domain</span>
          </div>

          <label>
            <span>Client ID</span>
            <input type="text" placeholder="admin" value={clientId} onChange={(event) => setClientId(event.target.value)} />
          </label>

          <label>
            <span>Password</span>
            <input
              type="password"
              placeholder="Enter admin password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
            />
          </label>

          <button type="submit" className="hero-btn primary submit">
            Enter Admin Dashboard
          </button>

          {error ? <div className="admin-login-error">{error}</div> : null}
        </form>
      </section>
    </main>
  )
}
