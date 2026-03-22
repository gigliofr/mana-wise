import { useState } from 'react'

const PAYPAL_DONATE_URL = import.meta.env.VITE_PAYPAL_DONATE_URL || 'https://paypal.me/gigliofr'
const API = '/api/v1'

export default function PlansSupport({ token, user, messages, onSessionUpdate }) {
  const [planError, setPlanError] = useState('')

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

  async function switchPlan(nextPlan) {
    setPlanError('')
    if (!token || !nextPlan || nextPlan === currentPlan) return
    const res = await fetch(`${API}/auth/plan`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({ plan: nextPlan }),
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || messages.planSwitchError)
    onSessionUpdate?.(data.token, data.user)
  }

  function donateCoffee() {
    window.open(PAYPAL_DONATE_URL, '_blank', 'noopener,noreferrer')
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
            onClick={() => switchPlan('free').catch(err => setPlanError(err.message))}
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
          <ul className="plan-list">
            {planRows.map(item => <li key={`pro-${item.label}`}><strong>{item.label}:</strong> {item.pro}</li>)}
          </ul>
          <button
            type="button"
            className="btn-primary"
            onClick={() => switchPlan('pro').catch(err => setPlanError(err.message))}
            disabled={currentPlan === 'pro'}
            style={{ width: '100%', marginTop: 10 }}
          >
            {currentPlan === 'pro' ? messages.planCurrent : messages.planSelect}
          </button>
        </div>
      </div>

      {planError && <div className="banner banner-error" style={{ marginTop: 10 }}>{planError}</div>}

      <div className="donate-box">
        <div>
          <strong>{messages.donateTitle}</strong>
          <p>{messages.donateBody}</p>
        </div>
        <button type="button" className="btn-primary" onClick={donateCoffee}>
          {messages.donateButton}
        </button>
      </div>
    </div>
  )
}
