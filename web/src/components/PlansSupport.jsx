import { useState } from 'react'

const PAYPAL_DONATE_URL = import.meta.env.VITE_PAYPAL_DONATE_URL || 'https://paypal.me/gigliofr'
const API = '/api/v1'

export default function PlansSupport({ token, user, messages, onSessionUpdate }) {
  const [planError, setPlanError] = useState('')
  const [donationReference, setDonationReference] = useState('')

  const planRows = [
    {
      label: messages.planFeatureAnalyses,
      free: messages.planFeatureAnalysesFree,
      pro: messages.planFeatureAnalysesPro,
    },
    {
      label: messages.planFeatureDeckSlots,
      free: messages.planFeatureDeckSlotsFree,
      pro: messages.planFeatureDeckSlotsPro,
    },
    {
      label: messages.planFeatureTools,
      free: messages.planFeatureToolsFree,
      pro: messages.planFeatureToolsPro,
    },
  ]

  const currentPlan = (user?.plan || 'free').toLowerCase()
  const proUntil = user?.pro_until

  async function downgradeToFree() {
    setPlanError('')
    if (!token || currentPlan === 'free') return
    const res = await fetch(`${API}/auth/plan`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({ plan: 'free' }),
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || messages.planSwitchError)
    onSessionUpdate?.(data.token, data.user)
  }

  async function activatePro(tier) {
    setPlanError('')
    if (!token) return
    if (!donationReference.trim()) {
      setPlanError(messages.planDonationReferenceRequired)
      return
    }
    const res = await fetch(`${API}/auth/plan`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({
        plan: 'pro',
        donation_tier: tier,
        donation_reference: donationReference.trim(),
      }),
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || messages.planSwitchError)
    onSessionUpdate?.(data.token, data.user)
  }

  function donateWithAmount(amount) {
    const normalized = String(amount).replace(',', '.')
    window.open(`${PAYPAL_DONATE_URL}/${normalized}`, '_blank', 'noopener,noreferrer')
  }

  return (
    <div className="card plan-support-card">
      <h2>{messages.planSectionTitle}</h2>
      <p className="plan-support-subtitle">{messages.planSectionSubtitle}</p>

      <div className="plan-grid">
        <div className="plan-box free">
          <div className="plan-box-head">
            <strong>{messages.planFreeTitle}</strong>
            <span>{messages.planFreeBadge}</span>
          </div>
          <ul className="plan-list">
            {planRows.map(item => <li key={`free-${item.label}`}><strong>{item.label}:</strong> {item.free}</li>)}
          </ul>
          <button
            type="button"
            className="btn-ghost"
            onClick={() => downgradeToFree().catch(err => setPlanError(err.message))}
            disabled={currentPlan === 'free'}
            style={{ width: '100%', marginTop: 10 }}
          >
            {currentPlan === 'free' ? messages.planCurrent : messages.planSelect}
          </button>
        </div>

        <div className="plan-box pro">
          <div className="plan-box-head">
            <strong>{messages.planProTitle}</strong>
            <span>{messages.planProBadge}</span>
          </div>
          <p className="plan-beta-note">{messages.planBetaNotice}</p>
          <ul className="plan-list">
            {planRows.map(item => <li key={`pro-${item.label}`}><strong>{item.label}:</strong> {item.pro}</li>)}
          </ul>
          {proUntil && <p className="plan-pro-until">{messages.planActiveUntil(proUntil)}</p>}

          <div className="plan-donation-tiers">
            <button type="button" className="btn-ghost" onClick={() => donateWithAmount('1')}>
              {messages.planDonateMonth}
            </button>
            <button type="button" className="btn-ghost" onClick={() => donateWithAmount('1.9')}>
              {messages.planDonateYear}
            </button>
          </div>

          <div className="form-row" style={{ marginTop: 10 }}>
            <label>{messages.planDonationReferenceLabel}</label>
            <input
              value={donationReference}
              onChange={e => setDonationReference(e.target.value)}
              placeholder={messages.planDonationReferencePlaceholder}
            />
          </div>

          <div className="plan-donation-tiers">
            <button
              type="button"
              className="btn-primary"
              onClick={() => activatePro('beta_month_1eur').catch(err => setPlanError(err.message))}
            >
              {messages.planActivateMonth}
            </button>
            <button
              type="button"
              className="btn-primary"
              onClick={() => activatePro('beta_year_190eur').catch(err => setPlanError(err.message))}
            >
              {messages.planActivateYear}
            </button>
          </div>
        </div>
      </div>

      {planError && <div className="banner banner-error" style={{ marginTop: 10 }}>{planError}</div>}

      <div className="donate-box">
        <div>
          <strong>{messages.donateTitle}</strong>
          <p>{messages.donateBody}</p>
        </div>
        <button type="button" className="btn-primary" onClick={() => donateWithAmount('1')}>
          {messages.donateButton}
        </button>
      </div>
    </div>
  )
}
