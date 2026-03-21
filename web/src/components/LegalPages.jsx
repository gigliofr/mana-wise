import { useMemo, useState } from 'react'

const SITE_OWNER = import.meta.env.VITE_SITE_OWNER || 'ManaWise Team'
const CONTACT_EMAIL = import.meta.env.VITE_CONTACT_EMAIL || 'support@manawise.app'

export function LegalPage({ path, messages }) {
  if (path === '/privacy') return <PrivacyPage messages={messages} />
  if (path === '/cookie') return <CookiePage messages={messages} />
  if (path === '/contatti') return <ContactsPage messages={messages} />
  return null
}

function PrivacyPage({ messages }) {
  return (
    <div className="card legal-card">
      <h2>{messages.privacyTitle}</h2>
      <p className="legal-intro">{messages.privacyIntro}</p>

      <section className="legal-section">
        <h3>{messages.privacyOwnerTitle}</h3>
        <p>{SITE_OWNER} - <a href={`mailto:${CONTACT_EMAIL}`}>{CONTACT_EMAIL}</a></p>
      </section>

      <section className="legal-section">
        <h3>{messages.privacyDataTitle}</h3>
        <ul>
          <li>{messages.privacyDataIp}</li>
          <li>{messages.privacyDataEmail}</li>
          <li>{messages.privacyDataLocal}</li>
        </ul>
      </section>

      <section className="legal-section">
        <h3>{messages.privacyPurposeTitle}</h3>
        <p>{messages.privacyPurposeBody}</p>
      </section>

      <section className="legal-section">
        <h3>{messages.privacyThirdPartyTitle}</h3>
        <p>{messages.privacyThirdPartyBody}</p>
      </section>

      <section className="legal-section">
        <h3>{messages.privacyRightsTitle}</h3>
        <p>{messages.privacyRightsBody(CONTACT_EMAIL)}</p>
      </section>

      <section className="legal-section">
        <h3>{messages.privacySecurityTitle}</h3>
        <ul>
          <li>{messages.privacySecurityHttps}</li>
          <li>{messages.privacySecurityLogs}</li>
          <li>{messages.privacySecurityMinimization}</li>
        </ul>
      </section>
    </div>
  )
}

function CookiePage({ messages }) {
  return (
    <div className="card legal-card">
      <h2>{messages.cookieTitle}</h2>
      <p>{messages.cookieBody}</p>

      <section className="legal-section">
        <h3>{messages.cookieTypesTitle}</h3>
        <ul>
          <li>{messages.cookieTechItem}</li>
          <li>{messages.cookieAnalyticsItem}</li>
        </ul>
      </section>

      <section className="legal-section">
        <h3>{messages.cookieBannerTitle}</h3>
        <p>{messages.cookieBannerBody}</p>
      </section>
    </div>
  )
}

function ContactsPage({ messages }) {
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [message, setMessage] = useState('')
  const [privacyAccepted, setPrivacyAccepted] = useState(false)

  const mailtoHref = useMemo(() => {
    const subject = encodeURIComponent(`ManaWise Contact - ${name || 'User'}`)
    const body = encodeURIComponent(`${messages.contactFormName}: ${name}\n${messages.contactFormEmail}: ${email}\n\n${message}`)
    return `mailto:${CONTACT_EMAIL}?subject=${subject}&body=${body}`
  }, [name, email, message, messages])

  return (
    <div className="card legal-card">
      <h2>{messages.contactsTitle}</h2>
      <p className="legal-intro">{messages.contactsIntro(CONTACT_EMAIL)}</p>

      <form className="contact-form" action={mailtoHref} method="get">
        <div className="form-row">
          <label>{messages.contactFormName}</label>
          <input value={name} onChange={e => setName(e.target.value)} required />
        </div>

        <div className="form-row">
          <label>{messages.contactFormEmail}</label>
          <input type="email" value={email} onChange={e => setEmail(e.target.value)} required />
        </div>

        <div className="form-row">
          <label>{messages.contactFormMessage}</label>
          <textarea value={message} onChange={e => setMessage(e.target.value)} required style={{ minHeight: 130 }} />
        </div>

        <label className="contact-privacy-check">
          <input
            type="checkbox"
            checked={privacyAccepted}
            onChange={e => setPrivacyAccepted(e.target.checked)}
            required
          />
          <span>{messages.contactPrivacyConsent}</span>
        </label>

        <button className="btn-primary" type="submit" disabled={!privacyAccepted}>
          {messages.contactSend}
        </button>
      </form>
    </div>
  )
}
