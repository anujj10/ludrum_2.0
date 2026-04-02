type AdminDashboardProps = {
  onLogout: () => void
  clientId: string
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

export default function AdminDashboard({ onLogout, clientId }: AdminDashboardProps) {
  const updatedAt = new Intl.DateTimeFormat("en-IN", {
    dateStyle: "medium",
    timeStyle: "short",
    timeZone: "Asia/Kolkata",
  }).format(new Date())

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
              <h3>Next steps</h3>
              <ul>
                <li>Wire this dashboard to live beta-user records.</li>
                <li>Add approve / revoke controls for client IDs.</li>
                <li>Move admin auth to the backend for real protection.</li>
              </ul>
            </div>
          </article>
        </section>
      </section>
    </main>
  )
}
