const CONTACT_EMAIL = import.meta.env.VITE_CONTACT_EMAIL || 'support@manawise.app'

export default function LegalFooter({ messages }) {
  return (
    <footer className="site-footer">
      <div className="container footer-inner">
        <div className="footer-top">
          <div className="footer-brand">ManaWise AI</div>
          <nav className="footer-links" aria-label={messages.footerLegalNavAria}>
            <a href="/privacy">{messages.footerPrivacy}</a>
            <a href="/cookie">{messages.footerCookie}</a>
            <a href="/contatti">{messages.footerContacts}</a>
          </nav>
        </div>

        <div className="footer-meta-row">
          <p className="footer-pill">
            {messages.footerContactLabel}: <a href={`mailto:${CONTACT_EMAIL}`}>{CONTACT_EMAIL}</a>
          </p>
          <p className="footer-pill">{messages.footerDataSource}</p>
        </div>

        <details className="footer-disclaimer">
          <summary>{messages.footerDisclaimerToggle}</summary>
          <p className="wizards-disclaimer">{messages.wizardsDisclaimer}</p>
        </details>

        <div className="footer-bottom">
          <span>{messages.footerFanMade}</span>
        </div>
      </div>
    </footer>
  )
}
