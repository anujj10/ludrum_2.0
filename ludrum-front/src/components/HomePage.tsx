const tickerRows = [
  ["NIFTY 50", "+1.24%", "BANKNIFTY", "+0.88%", "FINNIFTY", "+0.64%", "MIDCAP", "+1.92%"],
  ["SENSEX", "+0.94%", "VIX", "-2.10%", "AUTO", "+0.57%", "IT", "+1.11%"],
]

const featureCards = [
  {
    title: "Live chain focus",
    copy: "Track spot, ATM, OI flow, and tape rhythm in a terminal tuned for fast directional reads.",
  },
  {
    title: "Closed beta onboarding",
    copy: "Users apply with email and phone number first, then receive one-time credentials for access.",
  },
  {
    title: "After-hours awareness",
    copy: "The experience automatically shifts into a clean market-closed state outside Indian cash market hours.",
  },
]

export default function HomePage() {
  return (
    <main className="marketing-shell">
      <section className="hero-panel">
        <div className="hero-copy">
          <p className="eyebrow">Ludrum Terminal</p>
          <h1>Index options intelligence for traders who read flow, not noise.</h1>
          <p className="hero-text">
            A darker, faster view into NIFTY option movement with OI tape, strike focus, and session-aware market rhythm built for
            a private beta release.
          </p>
          <div className="hero-actions">
            <a className="hero-btn primary" href="/login">
              Client Login
            </a>
            <a className="hero-btn secondary" href="#beta-form">
              Join Beta Waitlist
            </a>
          </div>
          <div className="hero-stats">
            <div>
              <span>Market</span>
              <strong>NSE Index Options</strong>
            </div>
            <div>
              <span>Mode</span>
              <strong>Closed Beta</strong>
            </div>
            <div>
              <span>Experience</span>
              <strong>Real-time flow board</strong>
            </div>
          </div>
        </div>

        <div className="hero-visual" aria-hidden="true">
          <div className="hero-orbit orbit-a" />
          <div className="hero-orbit orbit-b" />
          <div className="hero-terminal-card">
            <div className="hero-terminal-header">
              <span>Signal deck</span>
              <strong>Session pulse</strong>
            </div>
            <div className="hero-bars">
              <span className="bar rise" />
              <span className="bar fall" />
              <span className="bar rise tall" />
              <span className="bar rise" />
              <span className="bar fall short" />
              <span className="bar rise medium" />
            </div>
            <div className="hero-grid-lines" />
          </div>
        </div>
      </section>

      <section className="ticker-strip" aria-label="Market ticker">
        <div className="ticker-track">
          {tickerRows.concat(tickerRows).map((row, rowIndex) => (
            <div className="ticker-row" key={`${rowIndex}-${row.join("-")}`}>
              {row.map((item, index) => (
                <span key={`${item}-${index}`} className={item.startsWith("+") ? "up" : item.startsWith("-") ? "down" : ""}>
                  {item}
                </span>
              ))}
            </div>
          ))}
        </div>
      </section>

      <section className="feature-grid">
        {featureCards.map((card) => (
          <article className="feature-card" key={card.title}>
            <h2>{card.title}</h2>
            <p>{card.copy}</p>
          </article>
        ))}
      </section>

      <section className="beta-panel" id="beta-form">
        <div className="beta-copy">
          <p className="eyebrow">Beta Access</p>
          <h2>Join the waitlist and receive private login credentials after review.</h2>
          <p>
            Submit your trading email and phone number. Once approved, you can confirm your client credentials and access the terminal.
          </p>
        </div>

        <form className="beta-form">
          <label>
            <span>Full name</span>
            <input type="text" placeholder="Your name" />
          </label>
          <label>
            <span>Email address</span>
            <input type="email" placeholder="you@example.com" />
          </label>
          <label>
            <span>Phone number</span>
            <input type="tel" placeholder="+91 98xxxxxx10" />
          </label>
          <label>
            <span>Trading style</span>
            <select defaultValue="intraday">
              <option value="intraday">Intraday index options</option>
              <option value="swing">Short swing options</option>
              <option value="analysis">Market analysis only</option>
            </select>
          </label>
          <button type="button" className="hero-btn primary submit">
            Request Beta Access
          </button>
        </form>
      </section>
    </main>
  )
}
