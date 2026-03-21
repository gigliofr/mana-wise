const CONTACT_EMAIL = import.meta.env.VITE_CONTACT_EMAIL || 'support@manawise.app'

export default function LegalFooter({ messages }) {
  return (
    <footer className="site-footer">
      <div className="container footer-inner">
        <nav className="footer-links" aria-label={messages.footerLegalNavAria}>
          <a href="/privacy">{messages.footerPrivacy}</a>
          <a href="/cookie">{messages.footerCookie}</a>
          <a href="/contatti">{messages.footerContacts}</a>
        </nav>

        <div className="footer-meta">
          <p>{messages.footerContactLabel}: <a href={`mailto:${CONTACT_EMAIL}`}>{CONTACT_EMAIL}</a></p>
          <p>{messages.footerDataSource}</p>
          <p className="wizards-disclaimer">{messages.wizardsDisclaimer}</p>
        </div>
      </div>
    </footer>
  )
}
