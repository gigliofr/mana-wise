import { useEffect, useRef, useState } from 'react'
import { apiRequest, throwIfNotOK } from '../lib/apiClient'

const PAYPAL_DONATE_URL = import.meta.env.VITE_PAYPAL_DONATE_URL || 'https://paypal.me/gigliofr'

export default function PlansSupport({ token, user, messages, onSessionUpdate, focusProActivationKey = 0 }) {
  const [planError, setPlanError] = useState('')
  const [donationReference, setDonationReference] = useState('')
  const [adminSecret, setAdminSecret] = useState('')
  const [metricsLoading, setMetricsLoading] = useState(false)
  const [metricsError, setMetricsError] = useState('')
  const [metricsSnapshot, setMetricsSnapshot] = useState(null)
  const [previousMetricsSnapshot, setPreviousMetricsSnapshot] = useState(null)
  const donationReferenceInputRef = useRef(null)

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
  const hasActiveProWindow = Boolean(proUntil && new Date(proUntil) > new Date())

  async function downgradeToFree() {
    setPlanError('')
    if (!token || currentPlan === 'free') return
    if (hasActiveProWindow) {
      setPlanError(messages.planDowngradeBlockedActivePro?.(proUntil) || messages.planSwitchError)
      return
    }
    const confirmed = window.confirm(messages.planDowngradeConfirm || 'Confermi il downgrade al piano Free?')
    if (!confirmed) return
    const { res, data } = await apiRequest('/auth/plan', {
      token,
      method: 'POST',
      body: { plan: 'free' },
    })
    throwIfNotOK(res, data, messages.planSwitchError)
    onSessionUpdate?.(data.token, data.refresh_token || '', data.user)
  }

  async function activatePro(tier) {
    setPlanError('')
    if (!token) return
    if (!donationReference.trim()) {
      setPlanError(messages.planDonationReferenceRequired)
      return
    }
    const { res, data } = await apiRequest('/auth/plan', {
      token,
      method: 'POST',
      body: {
        plan: 'pro',
        donation_tier: tier,
        donation_reference: donationReference.trim(),
      },
    })
    throwIfNotOK(res, data, messages.planSwitchError)
    onSessionUpdate?.(data.token, data.refresh_token || '', data.user)
  }

    useEffect(() => {
      if (!focusProActivationKey) return
      const inputEl = donationReferenceInputRef.current
      if (!inputEl) return
      inputEl.scrollIntoView({ behavior: 'smooth', block: 'center' })
      inputEl.focus()
    }, [focusProActivationKey])

  function donateWithAmount(amount) {
    const normalized = String(amount).replace(',', '.')
    window.open(`${PAYPAL_DONATE_URL}/${normalized}`, '_blank', 'noopener,noreferrer')
  }

  async function loadFunnelMetrics() {
    setMetricsError('')
    if (!adminSecret.trim()) {
      setMetricsError(messages.metricsSecretRequired)
      return
    }

    setMetricsLoading(true)
    try {
      const { res, data } = await apiRequest('/admin/metrics/funnel', {
        token,
        headers: {
          'X-Admin-Secret': adminSecret.trim(),
        }
      })
      throwIfNotOK(res, data, messages.metricsLoadError)
      setPreviousMetricsSnapshot(metricsSnapshot)
      setMetricsSnapshot(data?.snapshot || null)
    } catch (err) {
      setMetricsError(err.message || messages.metricsLoadError)
    } finally {
      setMetricsLoading(false)
    }
  }

  function downloadFunnelSnapshot() {
    if (!metricsSnapshot) return
    const payload = {
      exported_at: new Date().toISOString(),
      snapshot: metricsSnapshot,
    }
    const blob = new Blob([JSON.stringify(payload, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    const ts = new Date().toISOString().replace(/[:.]/g, '-')
    a.href = url
    a.download = `manawise-funnel-metrics-${ts}.json`
    document.body.appendChild(a)
    a.click()
    a.remove()
    URL.revokeObjectURL(url)
  }

  function downloadFunnelSnapshotCSV() {
    if (!metricsSnapshot) return

    const lines = []
    lines.push('section,key,value')
    lines.push(`summary,total_events,${Number(metricsSnapshot.total_events || 0)}`)
    lines.push(`summary,analysis_fallbacks,${Number(metricsSnapshot.analysis_fallbacks || 0)}`)
    lines.push(`summary,forwarding_errors,${Number(metricsSnapshot.forwarding_errors || 0)}`)
    lines.push(`summary,last_event_at_unix_ms,${Number(metricsSnapshot.last_event_at_unix_ms || 0)}`)

    const eventCounts = metricsSnapshot.event_counts || {}
    Object.entries(eventCounts).forEach(([key, value]) => {
      lines.push(`event_counts,"${String(key).replace(/"/g, '""')}",${Number(value || 0)}`)
    })

    const bySource = metricsSnapshot.analysis_by_ai_source || {}
    Object.entries(bySource).forEach(([key, value]) => {
      lines.push(`analysis_by_ai_source,"${String(key).replace(/"/g, '""')}",${Number(value || 0)}`)
    })

    const blob = new Blob([lines.join('\n')], { type: 'text/csv;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    const ts = new Date().toISOString().replace(/[:.]/g, '-')
    a.href = url
    a.download = `manawise-funnel-metrics-${ts}.csv`
    document.body.appendChild(a)
    a.click()
    a.remove()
    URL.revokeObjectURL(url)
  }

  const eventCounts = metricsSnapshot?.event_counts || {}
  const sourceCounts = metricsSnapshot?.analysis_by_ai_source || {}

  function metricDelta(current, previous) {
    const c = Number(current || 0)
    const p = Number(previous || 0)
    return c - p
  }

  function renderDelta(delta) {
    if (delta === 0) {
      return (
        <span style={{ marginLeft: 6, fontSize: '.72rem', color: 'var(--muted)' }}>
          {messages.metricsDeltaStable}
        </span>
      )
    }

    const positive = delta > 0
    const color = positive ? 'var(--green)' : 'var(--red)'
    const sign = positive ? '+' : ''
    return (
      <span style={{ marginLeft: 6, fontSize: '.72rem', color, fontWeight: 700 }}>
        {`${sign}${delta}`}
      </span>
    )
  }

  const totalEventsDelta = metricDelta(metricsSnapshot?.total_events, previousMetricsSnapshot?.total_events)
  const fallbackDelta = metricDelta(metricsSnapshot?.analysis_fallbacks, previousMetricsSnapshot?.analysis_fallbacks)
  const forwardingDelta = metricDelta(metricsSnapshot?.forwarding_errors, previousMetricsSnapshot?.forwarding_errors)

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
            disabled={currentPlan === 'free' || hasActiveProWindow}
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
              ref={donationReferenceInputRef}
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

      <div className="card" style={{ marginTop: 14 }}>
        <h3 style={{ marginTop: 0 }}>{messages.metricsSectionTitle}</h3>
        <p style={{ color: 'var(--muted)', marginTop: 0 }}>{messages.metricsSectionSubtitle}</p>

        <div className="form-row">
          <label>{messages.metricsSecretLabel}</label>
          <input
            type="password"
            value={adminSecret}
            onChange={e => setAdminSecret(e.target.value)}
            placeholder={messages.metricsSecretPlaceholder}
          />
        </div>

        <button
          type="button"
          className="btn-primary"
          onClick={() => loadFunnelMetrics()}
          disabled={metricsLoading}
        >
          {metricsLoading ? messages.loading : messages.metricsRefresh}
        </button>

        {metricsSnapshot && (
          <>
            <button
              type="button"
              className="btn-ghost"
              style={{ marginLeft: 8 }}
              onClick={downloadFunnelSnapshot}
            >
              {messages.metricsDownloadJson}
            </button>
            <button
              type="button"
              className="btn-ghost"
              style={{ marginLeft: 8 }}
              onClick={downloadFunnelSnapshotCSV}
            >
              {messages.metricsDownloadCsv}
            </button>
          </>
        )}

        {metricsError && <div className="banner banner-error" style={{ marginTop: 10 }}>{metricsError}</div>}

        {metricsSnapshot && (
          <div style={{ marginTop: 12 }}>
            <div className="stats-grid" style={{ marginTop: 0 }}>
              <div className="stat-item">
                <div className="stat-value">
                  {metricsSnapshot.total_events || 0}
                  {previousMetricsSnapshot && renderDelta(totalEventsDelta)}
                </div>
                <div className="stat-label">{messages.metricsTotalEvents}</div>
              </div>
              <div className="stat-item">
                <div className="stat-value">
                  {metricsSnapshot.analysis_fallbacks || 0}
                  {previousMetricsSnapshot && renderDelta(fallbackDelta)}
                </div>
                <div className="stat-label">{messages.metricsFallbacks}</div>
              </div>
              <div className="stat-item">
                <div className="stat-value">
                  {metricsSnapshot.forwarding_errors || 0}
                  {previousMetricsSnapshot && renderDelta(forwardingDelta)}
                </div>
                <div className="stat-label">{messages.metricsForwardingErrors}</div>
              </div>
            </div>

            <div style={{ marginTop: 10 }}>
              <strong>{messages.metricsEventCounts}</strong>
              <div style={{ marginTop: 6, display: 'flex', gap: 8, flexWrap: 'wrap' }}>
                {Object.keys(eventCounts).length === 0 && <span style={{ color: 'var(--muted)' }}>{messages.metricsNoData}</span>}
                {Object.entries(eventCounts).map(([key, value]) => (
                  <span
                    key={`evt-${key}`}
                    style={{
                      fontSize: '.75rem',
                      border: '1px solid var(--border)',
                      borderRadius: 999,
                      padding: '2px 8px',
                      color: 'var(--muted)',
                    }}
                  >
                    {key}: {value}
                  </span>
                ))}
              </div>
            </div>

            <div style={{ marginTop: 10 }}>
              <strong>{messages.metricsByAISource}</strong>
              <div style={{ marginTop: 6, display: 'flex', gap: 8, flexWrap: 'wrap' }}>
                {Object.keys(sourceCounts).length === 0 && <span style={{ color: 'var(--muted)' }}>{messages.metricsNoData}</span>}
                {Object.entries(sourceCounts).map(([key, value]) => (
                  <span
                    key={`src-${key}`}
                    style={{
                      fontSize: '.75rem',
                      border: '1px solid var(--border)',
                      borderRadius: 999,
                      padding: '2px 8px',
                      color: 'var(--muted)',
                    }}
                  >
                    {key}: {value}
                  </span>
                ))}
              </div>
            </div>

            {metricsSnapshot.last_event_at_unix_ms > 0 && (
              <p style={{ marginTop: 10, color: 'var(--muted)', fontSize: '.85rem' }}>
                {messages.metricsLastEvent(new Date(metricsSnapshot.last_event_at_unix_ms).toLocaleString())}
              </p>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
